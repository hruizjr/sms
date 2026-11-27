// Package provider defines the boundary between the messaging engine and SMS providers (ADR-0012,
// accepted with a limited scope: a PROVISIONAL interface, capability declarations, the adapter
// abstraction and the simulator).
//
// Every provider — the simulator today, a direct telco integration later — is an adapter that
// implements Provider. The messaging engine depends only on this package: it never imports an
// adapter, never branches on a provider's identity, and never interprets raw provider codes. An
// adapter translates its provider's protocol into the normalized types below and classifies every
// submission outcome itself, from the provider's official documentation.
//
// The interface is provisional: it is built against the simulator and is expected to change when
// the first official telco specification is received (decision log §15, S5).
package provider

import (
	"context"
	"time"
)

// Provider is implemented by every provider adapter.
type Provider interface {
	// ID is the stable identifier of the configured provider (for example "simulator"). It is
	// stored with every submission attempt and receipt.
	ID() string

	// Capabilities declares what the provider supports. Capabilities that are not documented by
	// the provider are declared SupportUnknown and treated conservatively by the engine.
	Capabilities() Capabilities

	// Submit hands one message to the provider and classifies the outcome. It never returns an
	// unclassified error: every failure is an Outcome. If ctx ends before the provider answered
	// and the message may already have been transmitted, the outcome is OutcomeUncertain.
	Submit(ctx context.Context, s Submission) SubmitResult

	// Run receives delivery receipts from the provider and passes each one, normalized, to sink.
	// It blocks until ctx is done (returning nil) or the receipt channel fails. Adapters without
	// receipts return nil when ctx is done.
	Run(ctx context.Context, sink ReceiptSink) error
}

// Submission is one message to hand to a provider. It contains everything an adapter needs and
// nothing about the tenant.
type Submission struct {
	// AttemptID identifies this hand-off attempt (unique; recorded before Submit is called).
	AttemptID string
	// Destination is an E.164 phone number.
	Destination string
	// SenderID is the approved sender ID.
	SenderID string
	// Body is the message text. Adapters must never log it.
	Body string
	// Encoding and SegmentCount are the platform's own calculation (they may differ from a
	// provider's; D-117).
	Encoding     string
	SegmentCount int
}

// Outcome classifies a submission (messaging reliability §4).
type Outcome int

const (
	// OutcomeAccepted: the provider acknowledged and accepted the message; ProviderMessageRef is
	// set. This is not delivery.
	OutcomeAccepted Outcome = iota + 1
	// OutcomeRejected: the provider definitively refused the message. It was not sent.
	OutcomeRejected
	// OutcomeNotSent: the message was definitely NOT transmitted (for example no connection was
	// available). Retryable says whether trying again later may succeed.
	OutcomeNotSent
	// OutcomeUncertain: the message may or may not have been transmitted (for example a timeout
	// after sending). The engine never resubmits it automatically (D-095 interim rule).
	OutcomeUncertain
)

func (o Outcome) String() string {
	switch o {
	case OutcomeAccepted:
		return "accepted"
	case OutcomeRejected:
		return "rejected"
	case OutcomeNotSent:
		return "not_sent"
	case OutcomeUncertain:
		return "uncertain"
	}
	return "invalid"
}

// SubmitResult is the normalized result of Submit.
type SubmitResult struct {
	Outcome Outcome
	// ProviderMessageRef is the provider-assigned message reference, set only for
	// OutcomeAccepted. Receipts are correlated by it. It is internal and never exposed to clients.
	ProviderMessageRef string
	// Retryable applies to OutcomeNotSent only.
	Retryable bool
	// ProviderCode is the provider's raw result or error code, kept as opaque data for audit and
	// reconciliation. The engine never interprets it.
	ProviderCode string
}

// ReceiptStatus is a normalized final delivery outcome reported by a provider.
type ReceiptStatus string

// Normalized receipt statuses. Adapters map their provider's receipt states to these; receipt
// states that are not final (for example "en route") are not passed to the engine.
const (
	ReceiptDelivered   ReceiptStatus = "delivered"
	ReceiptUndelivered ReceiptStatus = "undelivered"
	ReceiptExpired     ReceiptStatus = "expired"
)

// Valid reports whether s is a known receipt status.
func (s ReceiptStatus) Valid() bool {
	return s == ReceiptDelivered || s == ReceiptUndelivered || s == ReceiptExpired
}

// Receipt is a normalized delivery receipt (messaging reliability §5.1).
type Receipt struct {
	// ProviderMessageRef correlates the receipt with a submission (SubmitResult.ProviderMessageRef).
	ProviderMessageRef string
	// IdentityKey identifies this receipt for duplicate detection; the adapter defines it. Two
	// receipts with the same key for the same message are the same report.
	IdentityKey string
	Status      ReceiptStatus
	// ProviderStatusCode is the provider's raw status or error code (opaque).
	ProviderStatusCode string
	// ProviderReportedAt is the provider's own timestamp, if it sends one.
	ProviderReportedAt *time.Time
	// Raw is the receipt as received, for audit (may contain personal data).
	Raw []byte
}

// ReceiptSink stores receipts durably. Deliver returns only after the receipt is stored; an adapter
// that receives an error should not acknowledge the receipt to its provider.
type ReceiptSink interface {
	Deliver(ctx context.Context, providerID string, r Receipt) error
}
