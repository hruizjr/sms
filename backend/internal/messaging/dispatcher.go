package messaging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hruizjr/sms/backend/internal/database"
	"github.com/hruizjr/sms/backend/internal/database/sqlcgen"
	"github.com/hruizjr/sms/backend/internal/messaging/provider"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// DispatcherConfig configures the Dispatcher. The defaults (config package) are engineering values
// for local development, not telco terms: real limits come from provider agreements (AGENTS.md
// §11.4).
type DispatcherConfig struct {
	// PollInterval is the pause between dispatch cycles.
	PollInterval time.Duration
	// BatchSize is the maximum number of jobs, receipts or sweeps handled per tenant and cycle.
	BatchSize int
	// Concurrency bounds the submissions in flight in this process.
	Concurrency int
	// SubmitTimeout bounds one provider hand-off.
	SubmitTimeout time.Duration
	// Lease is how long a claimed job is reserved for this dispatcher. It must exceed
	// SubmitTimeout. An attempt still in progress after Lease is considered abandoned (a crashed
	// dispatcher) and becomes uncertain.
	Lease time.Duration
	// MaxAttempts bounds the hand-off attempts after definite "not sent" outcomes.
	MaxAttempts int
	// RetryBaseDelay and RetryMaxDelay shape the exponential backoff (with jitter) between attempts.
	RetryBaseDelay time.Duration
	RetryMaxDelay  time.Duration
	// MaxMessageAge: a message not yet handed to the provider this long after acceptance fails
	// with expired_before_submission instead of being sent late.
	MaxMessageAge time.Duration
	// ReportWindow is the configured provider's reporting window: a submitted message without a
	// final report after it becomes unknown. Zero disables the transition (there is no universal
	// default; messaging reliability §5.3).
	ReportWindow time.Duration
}

// Dispatcher hands queued messages to the provider and applies submission results and receipts.
// Several dispatchers may run against one database: jobs are claimed with SKIP LOCKED and a lease.
type Dispatcher struct {
	db       *database.DB
	provider provider.Provider
	cfg      DispatcherConfig
	logger   *slog.Logger
	// jitter returns a factor in [0.5, 1) applied to backoff delays; replaceable in tests.
	jitter func() float64
}

// NewDispatcher returns a Dispatcher.
func NewDispatcher(db *database.DB, p provider.Provider, cfg DispatcherConfig, logger *slog.Logger) (*Dispatcher, error) {
	if cfg.Lease <= cfg.SubmitTimeout {
		return nil, fmt.Errorf("dispatcher lease (%s) must exceed the submission timeout (%s)", cfg.Lease, cfg.SubmitTimeout)
	}
	if cfg.BatchSize < 1 || cfg.Concurrency < 1 || cfg.MaxAttempts < 1 {
		return nil, errors.New("dispatcher batch size, concurrency and max attempts must be positive")
	}
	return &Dispatcher{db: db, provider: p, cfg: cfg, logger: logger, jitter: func() float64 {
		return 0.5 + rand.Float64()/2 //nolint:gosec // G404: backoff jitter, not security
	}}, nil
}

// Run receives the provider's receipts and runs dispatch cycles until ctx is done. It owns the
// receipt goroutine and waits for it before returning.
func (d *Dispatcher) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	receiptErr := make(chan error, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				receiptErr <- fmt.Errorf("provider receipt loop panicked: %v", r)
			}
		}()
		receiptErr <- d.provider.Run(ctx, &InboxSink{db: d.db})
	}()
	defer func() {
		cancel()
		wg.Wait()
	}()

	ticker := time.NewTicker(d.cfg.PollInterval)
	defer ticker.Stop()
	for {
		if _, err := d.RunOnce(ctx); err != nil && ctx.Err() == nil {
			d.logger.ErrorContext(ctx, "dispatch cycle failed", slog.String("error", err.Error()))
		}
		select {
		case <-ctx.Done():
			return nil
		case err := <-receiptErr:
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("provider receipts: %w", err)
		case <-ticker.C:
		}
	}
}

// CycleStats counts what one cycle did.
type CycleStats struct {
	Recovered, Dispatched, Receipts, Overdue int
}

// RunOnce runs one cycle over every tenant: recover abandoned attempts, dispatch due jobs (active
// tenants only), apply correlated receipts, and move overdue submitted messages to unknown. Each
// tenant is processed in its own transaction-local context; nothing crosses tenants.
func (d *Dispatcher) RunOnce(ctx context.Context) (CycleStats, error) {
	var stats CycleStats
	tenants := tenancy.NewRepository(d.db.Pool())
	var after tenancy.ID
	for {
		page, err := tenants.List(ctx, after, tenancy.MaxPageSize)
		if err != nil {
			return stats, err
		}
		for _, t := range page {
			if err := d.runTenant(ctx, t, &stats); err != nil {
				return stats, fmt.Errorf("tenant %s: %w", t.ID, err)
			}
		}
		if len(page) < tenancy.MaxPageSize {
			return stats, nil
		}
		after = page[len(page)-1].ID
	}
}

func (d *Dispatcher) runTenant(ctx context.Context, t tenancy.Tenant, stats *CycleStats) error {
	n, err := d.recoverAbandoned(ctx, t.ID)
	stats.Recovered += n
	if err != nil {
		return fmt.Errorf("recover abandoned attempts: %w", err)
	}
	// Messages of a pending or suspended tenant stay queued; they are not failed.
	if t.Status == tenancy.StatusActive {
		n, err = d.dispatchDue(ctx, t.ID)
		stats.Dispatched += n
		if err != nil {
			return fmt.Errorf("dispatch: %w", err)
		}
	}
	n, err = d.applyReceipts(ctx, t.ID)
	stats.Receipts += n
	if err != nil {
		return fmt.Errorf("apply receipts: %w", err)
	}
	n, err = d.markOverdue(ctx, t.ID)
	stats.Overdue += n
	if err != nil {
		return fmt.Errorf("mark overdue: %w", err)
	}
	return nil
}

// dispatchDue claims the tenant's due jobs and hands them to the provider, with bounded
// concurrency.
func (d *Dispatcher) dispatchDue(ctx context.Context, tenantID tenancy.ID) (int, error) {
	var ids []pgtype.UUID
	err := d.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		ids, err = sqlcgen.New(tx).ClaimDispatchJobs(ctx, sqlcgen.ClaimDispatchJobsParams{
			TenantID: pgUUID(tenantID), BatchSize: int32(d.cfg.BatchSize), LeaseSeconds: d.cfg.Lease.Seconds(), //nolint:gosec // G115: small configured value
		})
		return err
	})
	if err != nil {
		return 0, err
	}
	sem := make(chan struct{}, d.cfg.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	for _, id := range ids {
		sem <- struct{}{}
		wg.Add(1)
		go func(id MessageID) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := d.dispatchOne(ctx, tenantID, id); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(MessageID(id.Bytes))
	}
	wg.Wait()
	return len(ids), firstErr
}

// attemptStart is what the first transaction of a dispatch hands to the provider call.
type attemptStart struct {
	attemptID pgtype.UUID
	number    int
	sub       provider.Submission
}

// dispatchOne hands one message to the provider:
//
//  1. transaction: lock the message; if it is still queued, record a new attempt (in_progress) and
//     move it to submitting. A message in any other state is not submitted again.
//  2. call the provider (outside any transaction) with SubmitTimeout;
//  3. transaction: record the classified outcome on the attempt, transition the message, and
//     complete or reschedule the job.
//
// A crash between 1 and 3 leaves the attempt in progress; recoverAbandoned then marks it uncertain.
func (d *Dispatcher) dispatchOne(ctx context.Context, tenantID tenancy.ID, id MessageID) error {
	start, err := d.startAttempt(ctx, tenantID, id)
	if err != nil || start == nil {
		return err
	}
	submitCtx, cancel := context.WithTimeout(ctx, d.cfg.SubmitTimeout)
	result := d.safeSubmit(submitCtx, start.sub)
	cancel()
	return d.finishAttempt(ctx, tenantID, id, start, result)
}

func (d *Dispatcher) safeSubmit(ctx context.Context, sub provider.Submission) (result provider.SubmitResult) {
	defer func() {
		if r := recover(); r != nil {
			// The adapter may have transmitted the message before it failed.
			result = provider.SubmitResult{Outcome: provider.OutcomeUncertain}
		}
	}()
	return d.provider.Submit(ctx, sub)
}

func (d *Dispatcher) startAttempt(ctx context.Context, tenantID tenancy.ID, id MessageID) (*attemptStart, error) {
	var start *attemptStart
	err := d.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		tid, mid := pgUUID(tenantID), pgUUID(id)
		m, err := q.LockMessageForDispatch(ctx, sqlcgen.LockMessageForDispatchParams{TenantID: tid, ID: mid})
		if err != nil {
			return err
		}
		switch State(m.Status) {
		case StateQueued:
		case StateSubmitting:
			// Another dispatcher is handing it over, or crashed doing so (recoverAbandoned handles
			// that). Leave the job to the lease.
			return nil
		default:
			// Already handed over or final: nothing left to do for this job.
			return q.CompleteDispatchJob(ctx, sqlcgen.CompleteDispatchJobParams{TenantID: tid, MessageID: mid})
		}
		if d.cfg.MaxMessageAge > 0 && time.Since(m.AcceptedAt) > d.cfg.MaxMessageAge {
			if err := transition(ctx, q, tid, mid, StateQueued, StateFailed, reasonPtr(ReasonExpiredBeforeSubmission), "dispatcher", pgtype.UUID{}, pgtype.UUID{}, nil); err != nil {
				return err
			}
			return q.CompleteDispatchJob(ctx, sqlcgen.CompleteDispatchJobParams{TenantID: tid, MessageID: mid})
		}
		number := int(m.Attempts) + 1
		attemptID, err := q.StartSubmissionAttempt(ctx, sqlcgen.StartSubmissionAttemptParams{
			TenantID: tid, MessageID: mid, AttemptNumber: int32(number), ProviderID: d.provider.ID(), //nolint:gosec // G115: bounded by MaxAttempts
		})
		if err != nil {
			return err
		}
		providerID := d.provider.ID()
		if err := transition(ctx, q, tid, mid, StateQueued, StateSubmitting, nil, "dispatcher", attemptID, pgtype.UUID{}, &providerID); err != nil {
			return err
		}
		start = &attemptStart{attemptID: attemptID, number: number, sub: provider.Submission{
			AttemptID: attemptID.String(), Destination: m.Destination, SenderID: m.SenderID, Body: m.Body,
			Encoding: m.Encoding, SegmentCount: int(m.SegmentCount),
		}}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("start attempt for %s: %w", id, err)
	}
	return start, nil
}

func (d *Dispatcher) finishAttempt(ctx context.Context, tenantID tenancy.ID, id MessageID, start *attemptStart, result provider.SubmitResult) error {
	attemptState, to, reason := "uncertain", StateSubmitUncertain, reasonPtr(ReasonSubmissionOutcomeUnknown)
	var ref *string
	reschedule := false
	switch result.Outcome {
	case provider.OutcomeAccepted:
		if result.ProviderMessageRef != "" {
			r := result.ProviderMessageRef
			attemptState, to, reason, ref = "accepted", StateSubmitted, nil, &r
		}
		// An acceptance without a reference cannot be correlated: treat it as uncertain.
	case provider.OutcomeRejected:
		attemptState, to, reason = "rejected", StateRejected, reasonPtr(ReasonProviderRejected)
	case provider.OutcomeNotSent:
		switch {
		case !result.Retryable:
			attemptState, to, reason = "not_sent_permanent", StateFailed, reasonPtr(ReasonPermanentSubmissionError)
		case start.number >= d.cfg.MaxAttempts:
			attemptState, to, reason = "not_sent_retryable", StateFailed, reasonPtr(ReasonRetriesExhausted)
		default:
			attemptState, to, reason, reschedule = "not_sent_retryable", StateQueued, nil, true
		}
	}
	var code *string
	if result.ProviderCode != "" {
		code = &result.ProviderCode
	}

	err := d.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		tid, mid := pgUUID(tenantID), pgUUID(id)
		n, err := q.FinishSubmissionAttempt(ctx, sqlcgen.FinishSubmissionAttemptParams{
			TenantID: tid, ID: start.attemptID, State: attemptState, ProviderMessageRef: ref, ProviderResultCode: code,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			// Recovery already marked this attempt uncertain (the result came after the lease).
			// The message stays uncertain; it is not changed here.
			d.logger.WarnContext(ctx, "submission result arrived after the attempt was recovered as uncertain",
				slog.String("tenant_id", tenantID.String()), slog.String("message_id", id.String()),
				slog.String("outcome", result.Outcome.String()))
			return nil
		}
		if err := transition(ctx, q, tid, mid, StateSubmitting, to, reason, "dispatcher", start.attemptID, pgtype.UUID{}, nil); err != nil {
			return err
		}
		if reschedule {
			return q.RescheduleDispatchJob(ctx, sqlcgen.RescheduleDispatchJobParams{
				TenantID: tid, MessageID: mid, DelaySeconds: d.backoff(start.number).Seconds(),
			})
		}
		return q.CompleteDispatchJob(ctx, sqlcgen.CompleteDispatchJobParams{TenantID: tid, MessageID: mid})
	})
	if err != nil {
		// The attempt stays in progress and becomes uncertain after the lease: it is never resent.
		return fmt.Errorf("record outcome for %s: %w", id, err)
	}
	d.logger.InfoContext(ctx, "submission attempt finished",
		slog.String("tenant_id", tenantID.String()), slog.String("message_id", id.String()),
		slog.Int("attempt", start.number), slog.String("provider", d.provider.ID()),
		slog.String("outcome", result.Outcome.String()), slog.String("state", string(to)))
	return nil
}

// backoff returns the delay before attempt number+1.
func (d *Dispatcher) backoff(number int) time.Duration {
	delay := d.cfg.RetryBaseDelay
	for i := 1; i < number && delay < d.cfg.RetryMaxDelay; i++ {
		delay *= 2
	}
	delay = min(delay, d.cfg.RetryMaxDelay)
	return time.Duration(float64(delay) * d.jitter())
}

// recoverAbandoned marks attempts that are still in progress after the lease as uncertain, and
// their messages submit_uncertain (messaging reliability §4, requirement 2). They are never
// resubmitted.
func (d *Dispatcher) recoverAbandoned(ctx context.Context, tenantID tenancy.ID) (int, error) {
	count := 0
	err := d.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		tid := pgUUID(tenantID)
		rows, err := q.ClaimStaleAttempts(ctx, sqlcgen.ClaimStaleAttemptsParams{
			TenantID: tid, StaleAfterSeconds: d.cfg.Lease.Seconds(), BatchSize: int32(d.cfg.BatchSize), //nolint:gosec // G115: small configured value
		})
		if err != nil {
			return err
		}
		for _, a := range rows {
			if _, err := q.FinishSubmissionAttempt(ctx, sqlcgen.FinishSubmissionAttemptParams{TenantID: tid, ID: a.ID, State: "uncertain"}); err != nil {
				return err
			}
			if err := transition(ctx, q, tid, a.MessageID, StateSubmitting, StateSubmitUncertain, reasonPtr(ReasonSubmissionOutcomeUnknown), "recovery", a.ID, pgtype.UUID{}, nil); err != nil {
				return err
			}
			if err := q.CompleteDispatchJob(ctx, sqlcgen.CompleteDispatchJobParams{TenantID: tid, MessageID: a.MessageID}); err != nil {
				return err
			}
			d.logger.WarnContext(ctx, "abandoned submission attempt marked uncertain; it will not be resent",
				slog.String("tenant_id", tenantID.String()), slog.String("message_id", MessageID(a.MessageID.Bytes).String()))
			count++
		}
		return nil
	})
	return count, err
}

// markOverdue moves submitted messages without a final report within the reporting window to
// unknown (not proof of non-delivery; a late report can still change it).
func (d *Dispatcher) markOverdue(ctx context.Context, tenantID tenancy.ID) (int, error) {
	if d.cfg.ReportWindow <= 0 {
		return 0, nil
	}
	count := 0
	err := d.db.InTenantTx(ctx, tenantID, func(tx pgx.Tx) error {
		q := sqlcgen.New(tx)
		tid := pgUUID(tenantID)
		providerID := d.provider.ID()
		ids, err := q.ClaimOverdueSubmitted(ctx, sqlcgen.ClaimOverdueSubmittedParams{
			TenantID: tid, ProviderID: &providerID, WindowSeconds: d.cfg.ReportWindow.Seconds(), BatchSize: int32(d.cfg.BatchSize), //nolint:gosec // G115: small configured value
		})
		if err != nil {
			return err
		}
		for _, id := range ids {
			if err := transition(ctx, q, tid, id, StateSubmitted, StateUnknown, reasonPtr(ReasonNoFinalReportReceived), "sweeper", pgtype.UUID{}, pgtype.UUID{}, nil); err != nil {
				return err
			}
			count++
		}
		return nil
	})
	return count, err
}

// errStateChanged means the message was no longer in the expected state.
var errStateChanged = errors.New("message state changed concurrently")

// transition validates and applies one state change and records it in the status history.
func transition(ctx context.Context, q *sqlcgen.Queries, tid, mid pgtype.UUID, from, to State, reason *Reason,
	source string, attemptID, receiptID pgtype.UUID, providerID *string,
) error {
	if !CanTransition(from, to) {
		return fmt.Errorf("transition %s -> %s is not allowed", from, to)
	}
	var reasonText *string
	if reason != nil {
		s := string(*reason)
		reasonText = &s
	}
	n, err := q.TransitionMessage(ctx, sqlcgen.TransitionMessageParams{
		TenantID: tid, ID: mid, FromStatus: string(from), ToStatus: string(to), StatusReason: reasonText, ProviderID: providerID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return errStateChanged
	}
	fromText := string(from)
	return q.InsertStatusHistory(ctx, sqlcgen.InsertStatusHistoryParams{
		TenantID: tid, MessageID: mid, FromStatus: &fromText, ToStatus: string(to), StatusReason: reasonText,
		Source: source, AttemptID: attemptID, ReceiptID: receiptID,
	})
}

func reasonPtr(r Reason) *Reason { return &r }
