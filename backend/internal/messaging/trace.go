package messaging

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hruizjr/sms/backend/internal/database/sqlcgen"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// Trace is the internal audit trail of one message: its status history, submission attempts and
// correlated receipts. It contains provider details and is never returned by the public API; it
// is for operators (smsctl) and tests.
type Trace struct {
	Message  Message
	History  []HistoryEntry
	Attempts []Attempt
	Receipts []ReceiptRecord
}

// HistoryEntry is one status change.
type HistoryEntry struct {
	From       *State
	To         State
	Reason     *Reason
	Source     string
	OccurredAt time.Time
}

// Attempt is one hand-off attempt.
type Attempt struct {
	Number             int
	ProviderID         string
	State              string
	ProviderMessageRef *string
	ProviderResultCode *string
	StartedAt          time.Time
	FinishedAt         *time.Time
}

// ReceiptRecord is one receipt correlated with the message and the effect it had.
type ReceiptRecord struct {
	IdentityKey string
	Status      string
	Effect      string
	ProcessedAt time.Time
}

// Trace returns the audit trail of a message of tenantID, or ErrNotFound.
func (s *Service) Trace(ctx context.Context, tenantID tenancy.ID, id MessageID) (Trace, error) {
	var t Trace
	err := s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		tid, mid := pgUUID(tenantID), pgUUID(id)
		row, err := q.GetMessage(ctx, sqlcgen.GetMessageParams{TenantID: tid, ID: mid})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		t.Message = messageFromRow(row)
		if t.History, err = loadHistory(ctx, q, tid, mid); err != nil {
			return err
		}
		attempts, err := q.ListSubmissionAttempts(ctx, sqlcgen.ListSubmissionAttemptsParams{TenantID: tid, MessageID: mid})
		if err != nil {
			return err
		}
		for _, a := range attempts {
			t.Attempts = append(t.Attempts, Attempt{
				Number: int(a.AttemptNumber), ProviderID: a.ProviderID, State: a.State,
				ProviderMessageRef: a.ProviderMessageRef, ProviderResultCode: a.ProviderResultCode,
				StartedAt: a.StartedAt.UTC(), FinishedAt: a.FinishedAt,
			})
		}
		receipts, err := q.ListMessageReceipts(ctx, sqlcgen.ListMessageReceiptsParams{TenantID: tid, MessageID: mid})
		if err != nil {
			return err
		}
		for _, r := range receipts {
			t.Receipts = append(t.Receipts, ReceiptRecord{IdentityKey: r.IdentityKey, Status: r.ReceiptStatus, Effect: r.Effect, ProcessedAt: r.ProcessedAt.UTC()})
		}
		return nil
	})
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Trace{}, fmt.Errorf("trace message: %w", err)
	}
	return t, err
}

// loadHistory reads the internal status history of a message, oldest first.
func loadHistory(ctx context.Context, q *sqlcgen.Queries, tid, mid pgtype.UUID) ([]HistoryEntry, error) {
	rows, err := q.ListStatusHistory(ctx, sqlcgen.ListStatusHistoryParams{TenantID: tid, MessageID: mid})
	if err != nil {
		return nil, err
	}
	history := make([]HistoryEntry, 0, len(rows))
	for _, h := range rows {
		e := HistoryEntry{To: State(h.ToStatus), Source: h.Source, OccurredAt: h.OccurredAt.UTC()}
		if h.FromStatus != nil {
			from := State(*h.FromStatus)
			e.From = &from
		}
		if h.StatusReason != nil {
			r := Reason(*h.StatusReason)
			e.Reason = &r
		}
		history = append(history, e)
	}
	return history, nil
}

// StatusChange is one change of a message's customer-visible status.
type StatusChange struct {
	Status     Status
	Reason     *Reason
	OccurredAt time.Time
}

// CustomerHistory reduces an internal status history to the changes a customer can see: internal
// transitions that do not change the customer-visible status and reason (for example queued to
// submitting, both "accepted") are left out. Provider details are never included.
func CustomerHistory(history []HistoryEntry) []StatusChange {
	changes := make([]StatusChange, 0, len(history))
	for _, h := range history {
		c := StatusChange{Status: h.To.Status(), Reason: h.Reason, OccurredAt: h.OccurredAt}
		if n := len(changes); n > 0 && changes[n-1].Status == c.Status && sameReason(changes[n-1].Reason, c.Reason) {
			continue
		}
		changes = append(changes, c)
	}
	return changes
}

func sameReason(a, b *Reason) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

// GetWithHistory returns a message of tenantID with its customer-visible status history, oldest
// first, or ErrNotFound.
func (s *Service) GetWithHistory(ctx context.Context, tenantID tenancy.ID, id MessageID) (Message, []StatusChange, error) {
	var m Message
	var history []HistoryEntry
	err := s.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		tid, mid := pgUUID(tenantID), pgUUID(id)
		row, err := q.GetMessage(ctx, sqlcgen.GetMessageParams{TenantID: tid, ID: mid})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		m = messageFromRow(row)
		history, err = loadHistory(ctx, q, tid, mid)
		return err
	})
	if errors.Is(err, ErrNotFound) {
		return Message{}, nil, err
	}
	if err != nil {
		return Message{}, nil, fmt.Errorf("get message with history: %w", err)
	}
	return m, CustomerHistory(history), nil
}
