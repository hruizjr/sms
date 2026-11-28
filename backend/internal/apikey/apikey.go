// Package apikey issues, verifies and revokes the API keys of the public API (contract security
// scheme api_key, D-143; security baseline: random, high-entropy, visible prefix, stored only as a
// hash, shown once, scoped, revocable).
//
// A key has the form
//
//	smsk_<tenant>_<key id>_<secret>
//
// where <tenant> and <key id> are the tenant's and the key's UUIDs and <secret> is 32 random bytes,
// all in lower-case Crockford base32. Everything before the secret is a non-secret, visible
// identifier. Only the SHA-256 hash of the secret is stored; the full key is returned once, at
// creation. Because the key names its tenant, authentication sets the tenant's transaction-local
// context first and looks the key up under row-level security: a key whose tenant part was
// altered simply does not exist.
//
// # Audit and last use
//
// Creating and revoking a key writes a security audit event in the same transaction (owner decision
// G4-8): the change and its event commit together or not at all. After a key has authenticated and
// been authorized for a public API request, RecordUse stores the time, at most once per key every
// LastUsedInterval (owner decision G4-6), so the stored time can be up to that interval older than the
// key's most recent request.
package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hruizjr/sms/backend/internal/audit"
	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/sqlcgen"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// Scope is a permission of an API key (contract x-required-permission).
type Scope string

// Scopes defined by the contract (D-143).
const (
	ScopeMessagesSend Scope = "messages:send"
	ScopeMessagesRead Scope = "messages:read"
)

// Valid reports whether s is a known scope.
func (s Scope) Valid() bool { return s == ScopeMessagesSend || s == ScopeMessagesRead }

const (
	tokenPrefix = "smsk_"
	secretBytes = 32
)

var encoding = base32.NewEncoding("0123456789abcdefghjkmnpqrstvwxyz").WithPadding(base32.NoPadding)

// Errors.
var (
	// ErrUnauthenticated: the key is malformed, unknown or revoked (HTTP 401). The cause is not
	// distinguished, so a caller cannot probe which keys exist.
	ErrUnauthenticated = errors.New("unauthenticated")
	// ErrTenantNotActive: the key is valid but its tenant is not active (HTTP 403).
	ErrTenantNotActive = errors.New("tenant is not active")
	// ErrInvalidScopes: no scope, or an unknown scope, was requested at creation.
	ErrInvalidScopes = errors.New("invalid scopes")
	// ErrInvalidLabel: the label is empty, too long or contains control characters.
	ErrInvalidLabel = errors.New("invalid label")
	// ErrNotFound: no such key of the tenant, or already revoked (revocation).
	ErrNotFound = errors.New("api key not found")
)

// MaxLabelCharacters is the maximum length of a key's label (ck_api_keys__label).
const MaxLabelCharacters = 100

// NormalizeLabel trims label and checks it against the stored constraint (1 to 100 characters, no
// control characters).
func NormalizeLabel(label string) (string, error) {
	label = strings.TrimSpace(label)
	if label == "" || !utf8.ValidString(label) || utf8.RuneCountInString(label) > MaxLabelCharacters ||
		strings.ContainsFunc(label, unicode.IsControl) {
		return "", ErrInvalidLabel
	}
	return label, nil
}

// LastUsedInterval is the most often a key's last-used time is written (owner decision G4-6).
const LastUsedInterval = time.Minute

// Principal is an authenticated API key.
type Principal struct {
	TenantID tenancy.ID
	KeyID    [16]byte
	Scopes   []Scope
	// LastUsedAt is the key's stored last-used time when it was authenticated (nil: never used).
	LastUsedAt *time.Time
}

// Has reports whether the key has scope.
func (p Principal) Has(scope Scope) bool { return slices.Contains(p.Scopes, scope) }

// Key describes a stored API key (never its secret).
type Key struct {
	ID        [16]byte
	TenantID  tenancy.ID
	Label     string
	Scopes    []Scope
	CreatedAt time.Time
	RevokedAt *time.Time
	// LastUsedAt is when the key last authenticated an authorized request, recorded at most once per
	// LastUsedInterval (nil: never used).
	LastUsedAt *time.Time
}

// Prefix returns the key's visible, non-secret identifier.
func (k Key) Prefix() string { return publicPart(k.TenantID, k.ID) }

// IDString returns the key ID in the form used by the key itself.
func (k Key) IDString() string { return encoding.EncodeToString(k.ID[:]) }

// Service manages API keys. It is safe for concurrent use.
type Service struct {
	db  *database.DB
	now func() time.Time
}

// Option configures a Service.
type Option func(*Service)

// WithClock sets the clock (tests).
func WithClock(now func() time.Time) Option { return func(s *Service) { s.now = now } }

// NewService returns a Service using db (connected as the application role).
func NewService(db *database.DB, opts ...Option) *Service {
	s := &Service{db: db, now: time.Now}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Create issues a key for tenant with the given label and scopes, on behalf of actor (a portal user,
// or audit.System for the local administration tool, with details naming it). It returns the stored
// key and the full key token, which is shown once and never stored. The key and its audit event are
// written in one transaction.
func (s *Service) Create(ctx context.Context, tenantID tenancy.ID, label string, scopes []Scope, actor audit.Actor, via string) (Key, string, error) {
	label, err := NormalizeLabel(label)
	if err != nil {
		return Key{}, "", err
	}
	if len(scopes) == 0 {
		return Key{}, "", fmt.Errorf("%w: at least one scope is required", ErrInvalidScopes)
	}
	names := make([]string, 0, len(scopes))
	for _, sc := range scopes {
		if !sc.Valid() {
			return Key{}, "", fmt.Errorf("%w: %q", ErrInvalidScopes, sc)
		}
		if !slices.Contains(names, string(sc)) {
			names = append(names, string(sc))
		}
	}
	secret := make([]byte, secretBytes)
	if _, err := rand.Read(secret); err != nil {
		return Key{}, "", fmt.Errorf("generate api key secret: %w", err)
	}
	hash := sha256.Sum256(secret)

	var key Key
	err = s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		row, err := sqlcgen.New(tx).CreateAPIKey(ctx, sqlcgen.CreateAPIKeyParams{
			TenantID: pgUUID(tenantID), Label: label, SecretHash: hash[:], Scopes: names,
		})
		if err != nil {
			return err
		}
		key = keyFromRow(row.ID, row.TenantID, row.Label, row.Scopes, row.CreatedAt, row.RevokedAt, row.LastUsedAt)
		return audit.Insert(ctx, tx, tenantID, audit.Event{
			Action: audit.ActionAPIKeyCreate, Outcome: audit.OutcomeSuccess, Actor: actor,
			Target:  &audit.Target{Type: audit.TargetAPIKey, ID: key.ID},
			Details: audit.Details{APIKeyName: key.Label, APIKeyScopes: names, Via: via},
		})
	})
	if err != nil {
		return Key{}, "", fmt.Errorf("create api key: %w", err)
	}
	return key, publicPart(tenantID, key.ID) + "_" + encoding.EncodeToString(secret), nil
}

// Revoke revokes a key of tenant on behalf of actor (see Create). Revoking takes effect for the next
// request. The revocation and its audit event are written in one transaction; revoking a key that is
// already revoked (ErrNotFound) changes nothing and records nothing.
func (s *Service) Revoke(ctx context.Context, tenantID tenancy.ID, keyID [16]byte, actor audit.Actor, via string) (Key, error) {
	var key Key
	err := s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		row, err := sqlcgen.New(tx).RevokeAPIKey(ctx, sqlcgen.RevokeAPIKeyParams{TenantID: pgUUID(tenantID), ID: pgUUID(keyID)})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		key = keyFromRow(row.ID, row.TenantID, row.Label, row.Scopes, row.CreatedAt, row.RevokedAt, row.LastUsedAt)
		return audit.Insert(ctx, tx, tenantID, audit.Event{
			Action: audit.ActionAPIKeyRevoke, Outcome: audit.OutcomeSuccess, Actor: actor,
			Target:  &audit.Target{Type: audit.TargetAPIKey, ID: key.ID},
			Details: audit.Details{APIKeyName: key.Label, Via: via},
		})
	})
	if err != nil {
		return Key{}, fmt.Errorf("revoke api key: %w", err)
	}
	return key, nil
}

// List returns the keys of tenant (without secrets).
func (s *Service) List(ctx context.Context, tenantID tenancy.ID) ([]Key, error) {
	var keys []Key
	err := s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		rows, err := sqlcgen.New(tx).ListAPIKeys(ctx, pgUUID(tenantID))
		for _, r := range rows {
			keys = append(keys, keyFromRow(r.ID, r.TenantID, r.Label, r.Scopes, r.CreatedAt, r.RevokedAt, r.LastUsedAt))
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	return keys, nil
}

// Authenticate verifies token and returns its principal. It returns ErrUnauthenticated for any
// malformed, unknown or revoked key, and ErrTenantNotActive if the key's tenant is not active.
func (s *Service) Authenticate(ctx context.Context, token string) (Principal, error) {
	tenantID, keyID, secret, ok := parse(token)
	if !ok {
		return Principal{}, ErrUnauthenticated
	}
	presented := sha256.Sum256(secret)

	var p Principal
	var active bool
	err := s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		row, err := q.GetAPIKeyCredential(ctx, sqlcgen.GetAPIKeyCredentialParams{TenantID: pgUUID(tenantID), ID: pgUUID(keyID)})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUnauthenticated
		}
		if err != nil {
			return err
		}
		if subtle.ConstantTimeCompare(row.SecretHash, presented[:]) != 1 || row.RevokedAt != nil {
			return ErrUnauthenticated
		}
		tenant, err := tenancy.NewRepository(tx).Get(ctx, tenantID)
		if err != nil {
			return err
		}
		active = tenant.Status == tenancy.StatusActive
		p = Principal{TenantID: tenantID, KeyID: keyID, LastUsedAt: row.LastUsedAt}
		for _, sc := range row.Scopes {
			p.Scopes = append(p.Scopes, Scope(sc))
		}
		return nil
	})
	switch {
	case errors.Is(err, ErrUnauthenticated), errors.Is(err, database.ErrInvalidTenantID):
		return Principal{}, ErrUnauthenticated
	case err != nil:
		return Principal{}, fmt.Errorf("authenticate api key: %w", err)
	case !active:
		return Principal{}, ErrTenantNotActive
	}
	return p, nil
}

// RecordUse records that the key of p was used for a request it was authenticated and authorized
// for. It writes at most once per key every LastUsedInterval: nothing when p's stored time is recent,
// and the conditional update skips the write when another request (on any API instance) has just
// recorded it. A revoked key is never updated. The update runs under the key's tenant context, so it
// can only change a key of p's tenant.
func (s *Service) RecordUse(ctx context.Context, p Principal) error {
	now := s.now()
	if p.LastUsedAt != nil && now.Sub(*p.LastUsedAt) < LastUsedInterval {
		return nil
	}
	err := s.db.InTenantTx(ctx, p.TenantID, func(tx pgx.Tx) error {
		_, err := sqlcgen.New(tx).RecordAPIKeyUse(ctx, sqlcgen.RecordAPIKeyUseParams{
			Now: now, StaleBefore: now.Add(-LastUsedInterval), TenantID: pgUUID(p.TenantID), ID: pgUUID(p.KeyID),
		})
		return err
	})
	if err != nil {
		return fmt.Errorf("record api key use: %w", err)
	}
	return nil
}

// ParseKeyID parses a key ID as printed by Key.IDString.
func ParseKeyID(s string) ([16]byte, error) {
	var id [16]byte
	b, err := decodeCanonical(strings.ToLower(strings.TrimSpace(s)))
	if err != nil || len(b) != 16 {
		return id, fmt.Errorf("%w: malformed key ID", ErrNotFound)
	}
	copy(id[:], b)
	return id, nil
}

func publicPart(tenantID tenancy.ID, keyID [16]byte) string {
	return tokenPrefix + encoding.EncodeToString(tenantID[:]) + "_" + encoding.EncodeToString(keyID[:])
}

// decodeCanonical decodes s and rejects any non-canonical encoding.
func decodeCanonical(s string) ([]byte, error) {
	b, err := encoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if encoding.EncodeToString(b) != s {
		return nil, errors.New("non-canonical encoding")
	}
	return b, nil
}

// parse splits a key token into its parts; ok is false for anything malformed.
func parse(token string) (tenantID tenancy.ID, keyID [16]byte, secret []byte, ok bool) {
	rest, found := strings.CutPrefix(token, tokenPrefix)
	if !found {
		return tenantID, keyID, nil, false
	}
	parts := strings.Split(rest, "_")
	if len(parts) != 3 {
		return tenantID, keyID, nil, false
	}
	t, err1 := decodeCanonical(parts[0])
	k, err2 := decodeCanonical(parts[1])
	sec, err3 := decodeCanonical(parts[2])
	if err1 != nil || err2 != nil || err3 != nil || len(t) != 16 || len(k) != 16 || len(sec) != secretBytes {
		return tenantID, keyID, nil, false
	}
	copy(tenantID[:], t)
	copy(keyID[:], k)
	return tenantID, keyID, sec, true
}

func keyFromRow(id, tenantID pgtype.UUID, label string, scopes []string, createdAt time.Time, revokedAt, lastUsedAt *time.Time) Key {
	k := Key{ID: id.Bytes, TenantID: tenancy.ID(tenantID.Bytes), Label: label, CreatedAt: createdAt, RevokedAt: revokedAt, LastUsedAt: lastUsedAt}
	for _, sc := range scopes {
		k.Scopes = append(k.Scopes, Scope(sc))
	}
	return k
}

func pgUUID(b [16]byte) pgtype.UUID { return pgtype.UUID{Bytes: b, Valid: true} }
