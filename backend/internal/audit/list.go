package audit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/sqlcgen"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// ErrInvalidCursor is returned by List for a malformed cursor or one issued for other filters.
var ErrInvalidCursor = errors.New("invalid cursor")

// Service records events in their own transaction and lists an organization's events. It is safe
// for concurrent use.
type Service struct {
	db *database.DB
}

// NewService returns a Service using db (connected as the application role).
func NewService(db *database.DB) *Service { return &Service{db: db} }

// Record writes ev for tenantID in a transaction of its own. Use Insert instead when ev records a
// change made in a transaction, so that both commit together.
func (s *Service) Record(ctx context.Context, tenantID tenancy.ID, ev Event) error {
	return s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error { return Insert(ctx, tx, tenantID, ev) })
}

// RecordPlatform writes a platform event (no organization).
func (s *Service) RecordPlatform(ctx context.Context, ev PlatformEvent) error {
	return InsertPlatform(ctx, s.db.Pool(), ev)
}

// Filter selects events for List. Nil fields do not filter.
type Filter struct {
	Action         *Action
	Outcome        *Outcome
	OccurredFrom   *time.Time // inclusive
	OccurredBefore *time.Time // exclusive
	// Limit is the page size (the HTTP layer enforces 1 to 100).
	Limit int
	// Cursor is the opaque next cursor of a previous page, or "".
	Cursor string
}

// Party is the actor or target of a listed event.
type Party struct {
	Type string
	ID   [16]byte
	// Email is the portal user's email address (portal users only, "" otherwise).
	Email string
}

// Entry is a listed tenant event.
type Entry struct {
	ID         [16]byte
	OccurredAt time.Time
	Action     Action
	Outcome    Outcome
	Actor      Party
	Target     *Party
	RequestID  string
	SourceIP   netip.Addr // invalid when unknown
	UserAgent  string
	Details    Details
}

type cursor struct {
	OccurredAt time.Time `json:"o"`
	ID         string    `json:"i"`
	Filters    string    `json:"f"`
}

// List returns one page of tenantID's events, newest first, and the cursor of the next page ("" when
// there are no more). It runs under the tenant's row-level security context.
func (s *Service) List(ctx context.Context, tenantID tenancy.ID, f Filter) ([]Entry, string, error) {
	params := sqlcgen.ListSecurityAuditEventsParams{
		TenantID:       pgtype.UUID{Bytes: tenantID, Valid: true},
		OccurredFrom:   f.OccurredFrom,
		OccurredBefore: f.OccurredBefore,
		PageLimit:      int32(f.Limit) + 1, //nolint:gosec // G115: the HTTP layer bounds Limit to 1..100.
	}
	if f.Action != nil {
		a := string(*f.Action)
		params.Action = &a
	}
	if f.Outcome != nil {
		o := string(*f.Outcome)
		params.Outcome = &o
	}
	filters := filterHash(f)
	if f.Cursor != "" {
		c, err := decodeCursor(f.Cursor)
		if err != nil || c.Filters != filters {
			return nil, "", ErrInvalidCursor
		}
		id, err := hex.DecodeString(c.ID)
		if err != nil || len(id) != 16 {
			return nil, "", ErrInvalidCursor
		}
		at := c.OccurredAt
		params.CursorOccurredAt = &at
		params.CursorID = pgtype.UUID{Bytes: [16]byte(id), Valid: true}
	}

	var rows []sqlcgen.ListSecurityAuditEventsRow
	err := s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		rows, err = sqlcgen.New(tx).ListSecurityAuditEvents(ctx, params)
		return err
	})
	if err != nil {
		return nil, "", fmt.Errorf("list audit events: %w", err)
	}
	next := ""
	if len(rows) > f.Limit {
		rows = rows[:f.Limit]
		last := rows[len(rows)-1]
		next = encodeCursor(cursor{OccurredAt: last.OccurredAt, ID: hex.EncodeToString(last.ID.Bytes[:]), Filters: filters})
	}
	entries := make([]Entry, len(rows))
	for i, r := range rows {
		e := Entry{
			ID: r.ID.Bytes, OccurredAt: r.OccurredAt, Action: Action(r.Action), Outcome: Outcome(r.Outcome),
			Actor:     Party{Type: r.ActorType, ID: r.ActorID.Bytes, Email: deref(r.ActorEmail)},
			RequestID: deref(r.RequestID), UserAgent: deref(r.UserAgent),
		}
		if r.SourceIp != nil {
			e.SourceIP = *r.SourceIp
		}
		if r.TargetType != nil {
			e.Target = &Party{Type: *r.TargetType, ID: r.TargetID.Bytes, Email: deref(r.TargetEmail)}
		}
		if err := json.Unmarshal(r.Details, &e.Details); err != nil {
			return nil, "", fmt.Errorf("list audit events: details of %x: %w", r.ID.Bytes, err)
		}
		entries[i] = e
	}
	return entries, next, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func filterHash(f Filter) string {
	b, _ := json.Marshal([]any{f.Action, f.Outcome, f.OccurredFrom, f.OccurredBefore}) // cannot fail
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}

func encodeCursor(c cursor) string {
	b, _ := json.Marshal(c) // cannot fail
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(s string) (cursor, error) {
	var c cursor
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return c, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return c, err
	}
	return c, nil
}
