// Package messaging is the messaging engine (Phase 4 Group 2): it accepts messages for a tenant,
// stores them durably with their dispatch job (transactional outbox), hands them to the configured
// provider through the provider-independent interface (package provider), applies normalized
// delivery receipts, and reports the customer-visible status of the public API.
//
// Every operation on tenant data runs inside the tenant's transaction-local context
// (database.DB.InTenantTx), so row-level security applies to every query, and every query also
// filters by tenant_id explicitly (AGENTS.md §10.7). Nothing here operates across tenants.
//
// Guarantees and non-guarantees (messaging reliability §1–§5): acceptance means the message is
// durably stored, not that a provider accepted or delivered it. Dispatch is at-least-once inside
// the platform, guarded so that a message is handed to the provider again only after a definite
// "not sent" outcome. An uncertain submission is never resubmitted or failed automatically (D-095
// interim rule). The platform never claims exactly-once SMS delivery.
package messaging

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/sqlcgen"
	"github.com/hruizjr/sms/backend/internal/messaging/provider"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// IdempotencyRetention is how long an idempotency key is honoured after acceptance (ADR-0018,
// D-094d: seven days).
const IdempotencyRetention = 7 * 24 * time.Hour

// MaxBodyCharacters is the maximum message length in characters (D-144).
const MaxBodyCharacters = 1600

// operationCreateMessage is the idempotency scope's operation (D-094b: account + operation).
const operationCreateMessage = "create_message"

// supportedDestination is the destination policy: Philippine mobile numbers only (D-043).
var supportedDestination = regexp.MustCompile(`^\+639[0-9]{9}$`)

// Errors returned by Service. Each maps to one problem type of the public API.
var (
	ErrNotFound                = errors.New("message not found")
	ErrSenderIDNotApproved     = errors.New("sender ID not approved")
	ErrDestinationNotSupported = errors.New("destination not supported")
	ErrMessageTooLong          = errors.New("message needs more segments than allowed")
	ErrInvalidBody             = errors.New("invalid message body")
	ErrIdempotencyKeyReused    = errors.New("idempotency key reused with a different request")
	ErrIdempotencyInProgress   = errors.New("a request with this idempotency key is in progress")
)

// AcceptRequest is a validated request to accept one message.
type AcceptRequest struct {
	Destination     string
	SenderID        string
	Body            string
	ClientReference *string
	// IdempotencyKey is the normalized key (without surrounding quotes).
	IdempotencyKey string
}

// Message is a message as the public API reports it (never its text).
type Message struct {
	ID              MessageID
	ClientReference *string
	Destination     string
	SenderID        string
	State           State
	Reason          *Reason
	Encoding        Encoding
	SegmentCount    int
	AcceptedAt      time.Time
	StatusUpdatedAt time.Time
}

// AcceptResult is the result of Accept.
type AcceptResult struct {
	// Message is the message as it was at acceptance (also for a replay).
	Message Message
	// Replayed is true when an earlier request with the same key and body was found.
	Replayed bool
}

// Service accepts and reads messages. It is safe for concurrent use.
type Service struct {
	db   *database.DB
	caps provider.Capabilities
}

// NewService returns a Service. caps are the capabilities of the configured provider; a known
// MaxSegments limit is enforced at acceptance.
func NewService(db *database.DB, caps provider.Capabilities) *Service {
	return &Service{db: db, caps: caps}
}

// errLostRace makes Accept look the key up again after a concurrent request recorded it first.
var errLostRace = errors.New("idempotency key recorded concurrently")

// Accept stores a message for tenantID and its dispatch job in one transaction, or replays the
// earlier acceptance for the same idempotency key and normalized request (ADR-0018):
//
//  1. a live record with a different request: ErrIdempotencyKeyReused;
//  2. a concurrent request with the same key still in progress: ErrIdempotencyInProgress;
//  3. a live record with the same request: the original acceptance, Replayed = true.
//
// Rejections (sender ID, destination, length) record nothing, so the client can fix the request
// and retry with the same key.
func (s *Service) Accept(ctx context.Context, tenantID tenancy.ID, req AcceptRequest) (AcceptResult, error) {
	fingerprint := Fingerprint(req)
	for range 3 {
		res, err := s.tryAccept(ctx, tenantID, req, fingerprint)
		if !errors.Is(err, errLostRace) {
			return res, err
		}
	}
	return AcceptResult{}, ErrIdempotencyInProgress
}

func (s *Service) tryAccept(ctx context.Context, tenantID tenancy.ID, req AcceptRequest, fingerprint [32]byte) (AcceptResult, error) {
	var res AcceptResult
	err := s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		tid := pgUUID(tenantID)
		// A concurrent request with the same key holds the idempotency row lock until it commits;
		// wait briefly, then report it as in progress (409) rather than block the client.
		if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '2s'"); err != nil {
			return err
		}

		existing, err := q.GetLiveIdempotencyRecord(ctx, sqlcgen.GetLiveIdempotencyRecordParams{
			TenantID: tid, Operation: operationCreateMessage, IdempotencyKey: req.IdempotencyKey,
		})
		switch {
		case err == nil:
			if string(existing.RequestFingerprint) != string(fingerprint[:]) {
				return ErrIdempotencyKeyReused
			}
			row, err := q.GetMessage(ctx, sqlcgen.GetMessageParams{TenantID: tid, ID: existing.MessageID})
			if err != nil {
				return fmt.Errorf("load replayed message: %w", err)
			}
			res = AcceptResult{Message: acceptanceView(messageFromRow(row)), Replayed: true}
			return nil
		case !errors.Is(err, pgx.ErrNoRows):
			return err
		}

		seg, err := s.checkAcceptable(ctx, q, tid, req)
		if err != nil {
			return err
		}
		row, err := q.InsertMessage(ctx, sqlcgen.InsertMessageParams{
			TenantID: tid, ClientReference: req.ClientReference, Destination: req.Destination,
			SenderID: req.SenderID, Encoding: string(seg.Encoding), SegmentCount: int32(seg.Segments), //nolint:gosec // G115: at most 1,600 characters, so at most 1,600 segments.
		})
		if err != nil {
			return err
		}
		if err := q.InsertMessageBody(ctx, sqlcgen.InsertMessageBodyParams{TenantID: tid, MessageID: row.ID, Body: req.Body}); err != nil {
			return err
		}
		if err := q.InsertDispatchJob(ctx, sqlcgen.InsertDispatchJobParams{TenantID: tid, MessageID: row.ID}); err != nil {
			return err
		}
		if err := q.InsertStatusHistory(ctx, sqlcgen.InsertStatusHistoryParams{
			TenantID: tid, MessageID: row.ID, ToStatus: string(StateQueued), Source: "api",
		}); err != nil {
			return err
		}
		_, err = q.RecordIdempotencyKey(ctx, sqlcgen.RecordIdempotencyKeyParams{
			TenantID: tid, Operation: operationCreateMessage, IdempotencyKey: req.IdempotencyKey,
			RequestFingerprint: fingerprint[:], MessageID: row.ID, RetentionSeconds: IdempotencyRetention.Seconds(),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return errLostRace // rolls back this attempt's message
		}
		if err != nil {
			return err
		}
		res = AcceptResult{Message: messageFromRow(sqlcgen.GetMessageRow(row))}
		return nil
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "55P03" { // lock_not_available
		return AcceptResult{}, ErrIdempotencyInProgress
	}
	if err != nil && !isDomainError(err) {
		return AcceptResult{}, fmt.Errorf("accept message: %w", err)
	}
	return res, err
}

// checkAcceptable applies the acceptance rules that depend on configuration and tenant data.
func (s *Service) checkAcceptable(ctx context.Context, q *sqlcgen.Queries, tid pgtype.UUID, req AcceptRequest) (Segmentation, error) {
	if n := utf8.RuneCountInString(req.Body); n < 1 || n > MaxBodyCharacters || !utf8.ValidString(req.Body) {
		return Segmentation{}, ErrInvalidBody
	}
	if !supportedDestination.MatchString(req.Destination) {
		return Segmentation{}, ErrDestinationNotSupported
	}
	seg := Segment(req.Body)
	if s.caps.MaxSegments > 0 && seg.Segments > s.caps.MaxSegments {
		return Segmentation{}, ErrMessageTooLong
	}
	approved, err := q.IsSenderIDApproved(ctx, sqlcgen.IsSenderIDApprovedParams{TenantID: tid, SenderID: req.SenderID})
	if err != nil {
		return Segmentation{}, err
	}
	if !approved {
		return Segmentation{}, ErrSenderIDNotApproved
	}
	return seg, nil
}

func isDomainError(err error) bool {
	for _, e := range []error{ErrSenderIDNotApproved, ErrDestinationNotSupported, ErrMessageTooLong,
		ErrInvalidBody, ErrIdempotencyKeyReused, ErrIdempotencyInProgress, ErrNotFound, errLostRace} {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

// Fingerprint returns the SHA-256 of the normalized request body: the parsed field values in a
// fixed order, so whitespace and key order of the original JSON do not matter (ADR-0018). The
// idempotency key and tracing headers are not part of it.
func Fingerprint(req AcceptRequest) [32]byte {
	b, _ := json.Marshal(struct { // a struct of strings cannot fail to encode
		To              string  `json:"to"`
		SenderID        string  `json:"sender_id"`
		Body            string  `json:"body"`
		ClientReference *string `json:"client_reference"`
	}{req.Destination, req.SenderID, req.Body, req.ClientReference})
	return sha256.Sum256(append([]byte(operationCreateMessage+"\n"), b...))
}

// acceptanceView is a message as it was at acceptance: status accepted, no reason (contract: a
// replay shows the acceptance-time state).
func acceptanceView(m Message) Message {
	m.State, m.Reason, m.StatusUpdatedAt = StateQueued, nil, m.AcceptedAt
	return m
}

// Get returns a message of tenantID, or ErrNotFound (also for another tenant's message).
func (s *Service) Get(ctx context.Context, tenantID tenancy.ID, id MessageID) (Message, error) {
	var m Message
	err := s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		row, err := sqlcgen.New(tx).GetMessage(ctx, sqlcgen.GetMessageParams{TenantID: pgUUID(tenantID), ID: pgUUID(id)})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		m = messageFromRow(row)
		return nil
	})
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Message{}, fmt.Errorf("get message: %w", err)
	}
	return m, err
}

// ApproveSenderID approves senderID for tenantID (local administration; there is no public
// sender-ID management API).
func (s *Service) ApproveSenderID(ctx context.Context, tenantID tenancy.ID, senderID string) error {
	if n := utf8.RuneCountInString(senderID); n < 1 || n > 32 {
		return fmt.Errorf("sender ID must be 1 to 32 characters")
	}
	return s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		return sqlcgen.New(tx).ApproveSenderID(ctx, sqlcgen.ApproveSenderIDParams{TenantID: pgUUID(tenantID), SenderID: senderID})
	})
}

func messageFromRow(row sqlcgen.GetMessageRow) Message {
	m := Message{
		ID:              MessageID(row.ID.Bytes),
		ClientReference: row.ClientReference,
		Destination:     row.Destination,
		SenderID:        row.SenderID,
		State:           State(row.Status),
		Encoding:        Encoding(row.Encoding),
		SegmentCount:    int(row.SegmentCount),
		AcceptedAt:      row.AcceptedAt.UTC(),
		StatusUpdatedAt: row.StatusUpdatedAt.UTC(),
	}
	if row.StatusReason != nil {
		r := Reason(*row.StatusReason)
		m.Reason = &r
	}
	return m
}

func pgUUID(b [16]byte) pgtype.UUID { return pgtype.UUID{Bytes: b, Valid: true} }
