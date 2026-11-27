// Package simulator is the simulated SMS provider (decision log §15: simulator-only development;
// ADR-0012). It implements provider.Provider like any future telco adapter, runs in-process, makes
// no network connection, and never sends an SMS. Nothing in it describes or emulates a real
// network: its capabilities are synthetic or unknown (S2), and its results are "verified against
// the simulator" only (S7).
//
// # Scenarios
//
// The outcome is chosen deterministically from the destination. Numbers in the synthetic test
// range +639000000000 to +639000000099 select a scenario by their last two digits; every other
// number behaves like the default scenario.
//
//	00-09, any other number  accepted; one "delivered" receipt
//	10                       rejected by the provider (definite refusal; nothing sent)
//	20                       not sent, retryable (failure before transmission, every attempt)
//	25                       not sent, permanent (failure before transmission)
//	30                       uncertain (simulated timeout after transmission; no answer)
//	40                       accepted; one "undelivered" receipt
//	41                       accepted; one "expired" receipt
//	50                       accepted; the same "delivered" receipt twice (duplicate report)
//	60                       accepted; no receipt at all (missing report)
package simulator

import (
	"container/heap"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hruizjr/sms/backend/internal/messaging/provider"
)

// ID is the provider identifier of the simulator.
const ID = "simulator"

// SyntheticRangePrefix is the prefix of the destination numbers that select a scenario.
const SyntheticRangePrefix = "+6390000000"

// Scenario names the simulated behaviour for one destination.
type Scenario string

// Scenarios (see the package documentation).
const (
	ScenarioDelivered        Scenario = "delivered"
	ScenarioRejected         Scenario = "rejected"
	ScenarioNotSentRetryable Scenario = "not_sent_retryable"
	ScenarioNotSentPermanent Scenario = "not_sent_permanent"
	ScenarioUncertain        Scenario = "uncertain"
	ScenarioUndelivered      Scenario = "undelivered"
	ScenarioExpired          Scenario = "expired"
	ScenarioDuplicateReceipt Scenario = "duplicate_receipt"
	ScenarioNoReceipt        Scenario = "no_receipt"
)

// ScenarioFor returns the scenario selected by destination.
func ScenarioFor(destination string) Scenario {
	suffix, ok := strings.CutPrefix(destination, SyntheticRangePrefix)
	if !ok || len(suffix) != 2 {
		return ScenarioDelivered
	}
	switch suffix {
	case "10":
		return ScenarioRejected
	case "20":
		return ScenarioNotSentRetryable
	case "25":
		return ScenarioNotSentPermanent
	case "30":
		return ScenarioUncertain
	case "40":
		return ScenarioUndelivered
	case "41":
		return ScenarioExpired
	case "50":
		return ScenarioDuplicateReceipt
	case "60":
		return ScenarioNoReceipt
	}
	return ScenarioDelivered
}

// Config configures the simulator. All values are synthetic.
type Config struct {
	// ReceiptDelay is how long after acceptance the simulated receipt is reported.
	ReceiptDelay time.Duration
	// MaxSegments, if positive, is declared as a synthetic segment limit; 0 declares it unknown.
	MaxSegments int
	// QueueSize bounds the receipts waiting to be reported; further receipts are dropped (a
	// missing receipt, which the engine must tolerate anyway). Zero uses 1024.
	QueueSize int
	// Now is the clock; nil uses time.Now.
	Now func() time.Time
}

// Simulator is the simulated provider. It is safe for concurrent use.
type Simulator struct {
	cfg     Config
	pending chan scheduled

	mu          sync.Mutex
	submissions []provider.Submission

	dropped atomic.Int64
}

// New returns a simulator.
func New(cfg Config) *Simulator {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 1024
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Simulator{cfg: cfg, pending: make(chan scheduled, cfg.QueueSize)}
}

// ID implements provider.Provider.
func (s *Simulator) ID() string { return ID }

// Capabilities implements provider.Provider. Every value is synthetic: it describes the simulator,
// not any real network. Duplicate detection and status queries are declared unsupported, the
// conservative defaults of decision S4.
func (s *Simulator) Capabilities() provider.Capabilities {
	return provider.Capabilities{
		DeliveryReceipts:   provider.SupportYes,
		StatusQuery:        provider.SupportNo,
		DuplicateDetection: provider.SupportNo,
		EncodingGSM7:       provider.SupportYes,
		EncodingUCS2:       provider.SupportYes,
		MaxSegments:        s.cfg.MaxSegments,
		Basis:              "synthetic (simulator; decision log §15, S2)",
	}
}

// Submit implements provider.Provider.
func (s *Simulator) Submit(ctx context.Context, sub provider.Submission) provider.SubmitResult {
	if ctx.Err() != nil {
		// Nothing was transmitted yet.
		return provider.SubmitResult{Outcome: provider.OutcomeNotSent, Retryable: true, ProviderCode: "SIM_CANCELLED"}
	}
	s.mu.Lock()
	s.submissions = append(s.submissions, sub)
	s.mu.Unlock()

	switch ScenarioFor(sub.Destination) {
	case ScenarioRejected:
		return provider.SubmitResult{Outcome: provider.OutcomeRejected, ProviderCode: "SIM_REJECTED"}
	case ScenarioNotSentRetryable:
		return provider.SubmitResult{Outcome: provider.OutcomeNotSent, Retryable: true, ProviderCode: "SIM_UNAVAILABLE"}
	case ScenarioNotSentPermanent:
		return provider.SubmitResult{Outcome: provider.OutcomeNotSent, ProviderCode: "SIM_PERMANENT_ERROR"}
	case ScenarioUncertain:
		return provider.SubmitResult{Outcome: provider.OutcomeUncertain, ProviderCode: "SIM_TIMEOUT"}
	case ScenarioUndelivered:
		return s.accept(provider.ReceiptUndelivered, 1)
	case ScenarioExpired:
		return s.accept(provider.ReceiptExpired, 1)
	case ScenarioDuplicateReceipt:
		return s.accept(provider.ReceiptDelivered, 2)
	case ScenarioNoReceipt:
		return s.accept(provider.ReceiptDelivered, 0)
	}
	return s.accept(provider.ReceiptDelivered, 1)
}

// accept returns an accepted result and schedules receipts copies of a receipt with status.
func (s *Simulator) accept(status provider.ReceiptStatus, receipts int) provider.SubmitResult {
	ref := "sim-" + randomHex()
	at := s.cfg.Now().Add(s.cfg.ReceiptDelay)
	for range receipts {
		code := "SIM_" + strings.ToUpper(string(status))
		raw, _ := json.Marshal(map[string]string{"ref": ref, "status": string(status), "code": code}) // cannot fail
		r := provider.Receipt{
			ProviderMessageRef: ref,
			IdentityKey:        ref + ":" + string(status),
			Status:             status,
			ProviderStatusCode: code,
			ProviderReportedAt: &at,
			Raw:                raw,
		}
		select {
		case s.pending <- scheduled{at: at, receipt: r}:
		default:
			s.dropped.Add(1)
		}
	}
	return provider.SubmitResult{Outcome: provider.OutcomeAccepted, ProviderMessageRef: ref, ProviderCode: "SIM_OK"}
}

// Run implements provider.Provider: it reports each scheduled receipt to sink when it is due. A
// receipt the sink fails to store is dropped (the engine tolerates missing receipts).
func (s *Simulator) Run(ctx context.Context, sink provider.ReceiptSink) error {
	var due receiptHeap
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	for {
		wait := time.Hour
		if len(due) > 0 {
			wait = max(due[0].at.Sub(s.cfg.Now()), 0)
		}
		timer.Reset(wait)
		select {
		case <-ctx.Done():
			return nil
		case item := <-s.pending:
			heap.Push(&due, item)
		case <-timer.C:
		}
		for len(due) > 0 && !due[0].at.After(s.cfg.Now()) {
			item := heap.Pop(&due).(scheduled) //nolint:forcetypeassert // the heap holds only scheduled values
			if err := sink.Deliver(ctx, ID, item.receipt); err != nil {
				s.dropped.Add(1)
			}
		}
	}
}

// DeliverPending reports every scheduled receipt to sink at once, ignoring the delay, and returns
// how many it delivered. It is for deterministic tests; do not combine it with a running Run.
func (s *Simulator) DeliverPending(ctx context.Context, sink provider.ReceiptSink) (int, error) {
	n := 0
	for {
		select {
		case item := <-s.pending:
			if err := sink.Deliver(ctx, ID, item.receipt); err != nil {
				return n, err
			}
			n++
		default:
			return n, nil
		}
	}
}

// Submissions returns a copy of every submission received, in order (for tests and demos).
func (s *Simulator) Submissions() []provider.Submission {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]provider.Submission(nil), s.submissions...)
}

// DroppedReceipts returns how many receipts were dropped (queue full or sink failure).
func (s *Simulator) DroppedReceipts() int64 { return s.dropped.Load() }

type scheduled struct {
	at      time.Time
	receipt provider.Receipt
}

type receiptHeap []scheduled

func (h receiptHeap) Len() int           { return len(h) }
func (h receiptHeap) Less(i, j int) bool { return h[i].at.Before(h[j].at) }
func (h receiptHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *receiptHeap) Push(x any)        { *h = append(*h, x.(scheduled)) } //nolint:forcetypeassert // heap.Interface
func (h *receiptHeap) Pop() any {
	old := *h
	item := old[len(old)-1]
	*h = old[:len(old)-1]
	return item
}

func randomHex() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b) // crypto/rand.Read never fails on supported platforms
	return hex.EncodeToString(b)
}
