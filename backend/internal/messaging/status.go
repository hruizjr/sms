package messaging

// State is an internal message state (messaging reliability §3.1, ADR-0021). It is stored in
// messages.status; clients see the derived Status.
type State string

// Internal states. ACCEPTED is not stored: a message and its dispatch job are committed together,
// so a stored message starts in StateQueued.
const (
	StateQueued          State = "queued"
	StateSubmitting      State = "submitting"
	StateSubmitUncertain State = "submit_uncertain"
	StateSubmitted       State = "submitted"
	StateRejected        State = "rejected"
	StateFailed          State = "failed"
	StateDelivered       State = "delivered"
	StateUndelivered     State = "undelivered"
	StateUnknown         State = "unknown"
)

// Status is the customer-visible status of the public API (contract MessageStatus, D-099).
type Status string

// Customer-visible statuses.
const (
	StatusAccepted            Status = "accepted"
	StatusProviderAccepted    Status = "provider_accepted"
	StatusSubmissionUncertain Status = "submission_uncertain"
	StatusDelivered           Status = "delivered"
	StatusUndelivered         Status = "undelivered"
	StatusUnknown             Status = "unknown"
	StatusRejected            Status = "rejected"
	StatusFailed              Status = "failed"
)

// Category groups statuses (contract StatusCategory).
type Category string

// Status categories.
const (
	CategoryInProgress Category = "in_progress"
	CategoryUncertain  Category = "uncertain"
	CategoryFinal      Category = "final"
)

// Reason is a platform-defined, provider-independent status reason code (contract StatusReason).
type Reason string

// Reason codes (initial owner-approved set, D-099).
const (
	ReasonRecipientOptedOut           Reason = "recipient_opted_out"
	ReasonPolicyViolation             Reason = "policy_violation"
	ReasonProviderRejected            Reason = "provider_rejected"
	ReasonRetriesExhausted            Reason = "retries_exhausted"
	ReasonExpiredBeforeSubmission     Reason = "expired_before_submission"
	ReasonPermanentSubmissionError    Reason = "permanent_submission_error"
	ReasonProviderReportedUndelivered Reason = "provider_reported_undelivered"
	ReasonProviderReportedExpired     Reason = "provider_reported_expired"
	ReasonSubmissionOutcomeUnknown    Reason = "submission_outcome_unknown"
	ReasonNoFinalReportReceived       Reason = "no_final_report_received"
)

// reasonDescriptions are the human-readable texts of the reason codes (English; clients must not
// parse them).
var reasonDescriptions = map[Reason]string{
	ReasonRecipientOptedOut:           "The recipient is on a suppression list, so the message was not sent.",
	ReasonPolicyViolation:             "The message was not sent because it violates a sending policy.",
	ReasonProviderRejected:            "The provider refused the message; it was not sent.",
	ReasonRetriesExhausted:            "The message could not be handed to the provider before the attempt limit was reached; it was not sent.",
	ReasonExpiredBeforeSubmission:     "The message could not be handed to the provider in time; it was not sent.",
	ReasonPermanentSubmissionError:    "Handing the message to the provider failed permanently; it was not sent.",
	ReasonProviderReportedUndelivered: "The provider reported that the message was not delivered.",
	ReasonProviderReportedExpired:     "The provider reported that the message expired before it could be delivered.",
	ReasonSubmissionOutcomeUnknown:    "The provider did not confirm whether it accepted the message. It may already have been delivered, so it is not resent automatically.",
	ReasonNoFinalReportReceived:       "The provider accepted the message but no final delivery report arrived within the reporting window. A late report can still update this status.",
}

// Description returns the human-readable text of r.
func (r Reason) Description() string { return reasonDescriptions[r] }

// Status returns the customer-visible status of s (contracts/openapi/README.md §3).
func (s State) Status() Status {
	switch s {
	case StateQueued, StateSubmitting:
		return StatusAccepted
	case StateSubmitted:
		return StatusProviderAccepted
	case StateSubmitUncertain:
		return StatusSubmissionUncertain
	case StateDelivered:
		return StatusDelivered
	case StateUndelivered:
		return StatusUndelivered
	case StateUnknown:
		return StatusUnknown
	case StateRejected:
		return StatusRejected
	case StateFailed:
		return StatusFailed
	}
	// Every stored state is listed above (a CHECK constraint enforces it). An unexpected value is
	// shown as in progress, which is how the contract tells clients to treat unknown values.
	return StatusAccepted
}

// Category returns the status category of s.
func (s Status) Category() Category {
	switch s {
	case StatusSubmissionUncertain, StatusUnknown:
		return CategoryUncertain
	case StatusDelivered, StatusUndelivered, StatusRejected, StatusFailed:
		return CategoryFinal
	}
	return CategoryInProgress
}

// States returns the internal states shown as s, or nil if s is not a known status.
func (s Status) States() []State {
	switch s {
	case StatusAccepted:
		return []State{StateQueued, StateSubmitting}
	case StatusProviderAccepted:
		return []State{StateSubmitted}
	case StatusSubmissionUncertain:
		return []State{StateSubmitUncertain}
	case StatusDelivered:
		return []State{StateDelivered}
	case StatusUndelivered:
		return []State{StateUndelivered}
	case StatusUnknown:
		return []State{StateUnknown}
	case StatusRejected:
		return []State{StateRejected}
	case StatusFailed:
		return []State{StateFailed}
	}
	return nil
}

// IsFinal reports whether s is a final state.
func (s State) IsFinal() bool {
	switch s {
	case StateRejected, StateFailed, StateDelivered, StateUndelivered:
		return true
	}
	return false
}

// transitions lists every allowed state change (messaging reliability §3–§5). Precedence rules:
// a final state is never left; uncertain states (submit_uncertain, unknown) may be replaced only
// by a provider-reported final outcome; an uncertain submission is never moved back to the queue
// or failed automatically (D-095 interim rule).
var transitions = map[State][]State{
	StateQueued:          {StateSubmitting, StateFailed},
	StateSubmitting:      {StateSubmitted, StateRejected, StateFailed, StateQueued, StateSubmitUncertain},
	StateSubmitted:       {StateDelivered, StateUndelivered, StateUnknown},
	StateSubmitUncertain: {StateDelivered, StateUndelivered},
	StateUnknown:         {StateDelivered, StateUndelivered},
}

// CanTransition reports whether a message may move from one state to another.
func CanTransition(from, to State) bool {
	for _, allowed := range transitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}
