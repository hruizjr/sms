package messaging

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/sqlcgen"
	"github.com/hruizjr/sms/backend/internal/messaging/provider"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// InboxSink stores normalized receipts, as received, in the platform-owned provider_receipts
// table (messaging reliability §5.2). It implements provider.ReceiptSink. A receipt carries only
// the provider's message reference, so it is stored without a tenant; the dispatcher correlates it
// inside each tenant's context.
type InboxSink struct {
	db *database.DB
}

// NewInboxSink returns an InboxSink.
func NewInboxSink(db *database.DB) *InboxSink { return &InboxSink{db: db} }

// Deliver implements provider.ReceiptSink.
func (s *InboxSink) Deliver(ctx context.Context, providerID string, r provider.Receipt) error {
	if !r.Status.Valid() || r.ProviderMessageRef == "" || r.IdentityKey == "" {
		return fmt.Errorf("invalid receipt from provider %s", providerID)
	}
	var code *string
	if r.ProviderStatusCode != "" {
		code = &r.ProviderStatusCode
	}
	_, err := sqlcgen.New(s.db.Pool()).InsertProviderReceipt(ctx, sqlcgen.InsertProviderReceiptParams{
		ProviderID: providerID, ProviderMessageRef: r.ProviderMessageRef, IdentityKey: r.IdentityKey,
		ReceiptStatus: string(r.Status), ProviderStatusCode: code, ProviderReportedAt: r.ProviderReportedAt, Raw: r.Raw,
	})
	if err != nil {
		return fmt.Errorf("store receipt: %w", err)
	}
	return nil
}

// Receipt effects recorded in message_receipts.
const (
	effectApplied    = "applied"
	effectDuplicate  = "duplicate"
	effectSuperseded = "superseded"
	effectConflict   = "conflict"
)

// applyReceipts correlates unmatched receipts with the tenant's submission attempts and applies
// them by precedence, not arrival order (messaging reliability §5.5):
//
//   - a receipt already seen for the message (same identity key) is a duplicate: recorded, no change;
//   - submitted, submit_uncertain and unknown messages take the reported final outcome;
//   - a final message keeps its status: the same outcome is recorded as superseded, a different one
//     as a conflict, and the message is flagged for reconciliation (D-098 is open);
//   - a receipt that arrives before the acknowledgment is recorded stays unmatched and is matched
//     on a later cycle.
func (d *Dispatcher) applyReceipts(ctx context.Context, tenantID tenancy.ID) (int, error) {
	count := 0
	err := d.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		tid := pgUUID(tenantID)
		rows, err := q.ClaimMatchableReceipts(ctx, sqlcgen.ClaimMatchableReceiptsParams{TenantID: tid, BatchSize: int32(d.cfg.BatchSize)}) //nolint:gosec // G115: small configured value
		if err != nil {
			return err
		}
		for _, r := range rows {
			if err := applyReceipt(ctx, q, tid, r); err != nil {
				return err
			}
			count++
		}
		return nil
	})
	return count, err
}

func applyReceipt(ctx context.Context, q *sqlcgen.Queries, tid pgtype.UUID, r sqlcgen.ClaimMatchableReceiptsRow) error {
	status, err := q.LockMessageStatus(ctx, sqlcgen.LockMessageStatusParams{TenantID: tid, ID: r.MessageID})
	if err != nil {
		return err
	}
	current := State(status)
	target, reason := receiptOutcome(provider.ReceiptStatus(r.ReceiptStatus))
	seen, err := q.ReceiptIdentitySeen(ctx, sqlcgen.ReceiptIdentitySeenParams{TenantID: tid, MessageID: r.MessageID, IdentityKey: r.IdentityKey})
	if err != nil {
		return err
	}

	effect := effectSuperseded
	switch {
	case seen:
		effect = effectDuplicate
	case CanTransition(current, target):
		if err := transition(ctx, q, tid, r.MessageID, current, target, reason, "receipt", pgtype.UUID{}, r.ID, nil); err != nil {
			if !errors.Is(err, errStateChanged) {
				return err
			}
		} else {
			effect = effectApplied
		}
	case current.IsFinal() && current != target:
		effect = effectConflict
		if err := q.FlagConflictingReports(ctx, sqlcgen.FlagConflictingReportsParams{TenantID: tid, ID: r.MessageID}); err != nil {
			return err
		}
	}
	if err := q.InsertMessageReceipt(ctx, sqlcgen.InsertMessageReceiptParams{
		TenantID: tid, MessageID: r.MessageID, ProviderReceiptID: r.ID, IdentityKey: r.IdentityKey,
		ReceiptStatus: r.ReceiptStatus, Effect: effect,
	}); err != nil {
		return err
	}
	return q.MarkProviderReceiptMatched(ctx, r.ID)
}

// receiptOutcome maps a normalized receipt status to the message state and reason it reports.
func receiptOutcome(s provider.ReceiptStatus) (State, *Reason) {
	switch s {
	case provider.ReceiptDelivered:
		return StateDelivered, nil
	case provider.ReceiptExpired:
		return StateUndelivered, reasonPtr(ReasonProviderReportedExpired)
	}
	return StateUndelivered, reasonPtr(ReasonProviderReportedUndelivered)
}
