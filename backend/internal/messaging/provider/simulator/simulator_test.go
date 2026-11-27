package simulator

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/hruizjr/sms/backend/internal/messaging/provider"
)

type recordingSink struct {
	mu       sync.Mutex
	receipts []provider.Receipt
}

func (s *recordingSink) Deliver(_ context.Context, providerID string, r provider.Receipt) error {
	if providerID != ID {
		panic("unexpected provider " + providerID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.receipts = append(s.receipts, r)
	return nil
}

func (s *recordingSink) all() []provider.Receipt {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]provider.Receipt(nil), s.receipts...)
}

func TestScenarios(t *testing.T) {
	t.Parallel()
	tests := []struct {
		destination string
		outcome     provider.Outcome
		retryable   bool
		receipts    []provider.ReceiptStatus
	}{
		{"+639000000001", provider.OutcomeAccepted, false, []provider.ReceiptStatus{provider.ReceiptDelivered}},
		{"+639171234567", provider.OutcomeAccepted, false, []provider.ReceiptStatus{provider.ReceiptDelivered}},
		{"+639000000010", provider.OutcomeRejected, false, nil},
		{"+639000000020", provider.OutcomeNotSent, true, nil},
		{"+639000000025", provider.OutcomeNotSent, false, nil},
		{"+639000000030", provider.OutcomeUncertain, false, nil},
		{"+639000000040", provider.OutcomeAccepted, false, []provider.ReceiptStatus{provider.ReceiptUndelivered}},
		{"+639000000041", provider.OutcomeAccepted, false, []provider.ReceiptStatus{provider.ReceiptExpired}},
		{"+639000000050", provider.OutcomeAccepted, false, []provider.ReceiptStatus{provider.ReceiptDelivered, provider.ReceiptDelivered}},
		{"+639000000060", provider.OutcomeAccepted, false, nil},
	}
	for _, tc := range tests {
		t.Run(tc.destination, func(t *testing.T) {
			t.Parallel()
			sim := New(Config{})
			res := sim.Submit(context.Background(), provider.Submission{AttemptID: "a1", Destination: tc.destination, Body: "x"})
			if res.Outcome != tc.outcome || res.Retryable != tc.retryable {
				t.Fatalf("Submit() = %+v, want outcome %s retryable %v", res, tc.outcome, tc.retryable)
			}
			if (res.Outcome == provider.OutcomeAccepted) != (res.ProviderMessageRef != "") {
				t.Errorf("provider reference %q for outcome %s", res.ProviderMessageRef, res.Outcome)
			}
			sink := &recordingSink{}
			if _, err := sim.DeliverPending(context.Background(), sink); err != nil {
				t.Fatal(err)
			}
			got := sink.all()
			if len(got) != len(tc.receipts) {
				t.Fatalf("%d receipts, want %d", len(got), len(tc.receipts))
			}
			for i, r := range got {
				if r.Status != tc.receipts[i] || r.ProviderMessageRef != res.ProviderMessageRef || r.IdentityKey == "" {
					t.Errorf("receipt %d = %+v", i, r)
				}
			}
			if len(got) == 2 && got[0].IdentityKey != got[1].IdentityKey {
				t.Errorf("duplicate receipts must share an identity key: %q, %q", got[0].IdentityKey, got[1].IdentityKey)
			}
		})
	}
}

func TestCapabilitiesAreSyntheticAndConservative(t *testing.T) {
	t.Parallel()
	c := New(Config{}).Capabilities()
	if c.DuplicateDetection != provider.SupportNo || c.StatusQuery != provider.SupportNo || c.MaxSegments != 0 {
		t.Errorf("capabilities = %+v, want no duplicate detection, no status query, unknown segment limit", c)
	}
	if c.Basis == "" {
		t.Error("capabilities must state their basis")
	}
	if New(Config{MaxSegments: 3}).Capabilities().MaxSegments != 3 {
		t.Error("synthetic segment limit not declared")
	}
}

func TestCancelledContextIsNotSent(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res := New(Config{}).Submit(ctx, provider.Submission{Destination: "+639000000001"})
	if res.Outcome != provider.OutcomeNotSent || !res.Retryable {
		t.Errorf("Submit() on a cancelled context = %+v, want not sent, retryable", res)
	}
}

// channelSink passes delivered receipts to a channel, so the test waits for an event instead of
// sleeping.
type channelSink chan provider.Receipt

func (c channelSink) Deliver(_ context.Context, _ string, r provider.Receipt) error {
	c <- r
	return nil
}

func TestRunDeliversReceiptsWhenDue(t *testing.T) {
	t.Parallel()
	sim := New(Config{ReceiptDelay: 20 * time.Millisecond})
	sink := make(channelSink, 1)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- sim.Run(ctx, sink) }()
	res := sim.Submit(ctx, provider.Submission{Destination: "+639000000001"})
	select {
	case r := <-sink:
		if r.ProviderMessageRef != res.ProviderMessageRef || r.Status != provider.ReceiptDelivered {
			t.Errorf("receipt = %+v", r)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no receipt delivered")
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run() = %v", err)
	}
}
