package provider

// Support is a declared capability value. The zero value is SupportUnknown, so a capability an
// adapter does not declare is never mistaken for confirmed support.
type Support int

const (
	// SupportUnknown: not documented by the provider. The engine treats it conservatively.
	SupportUnknown Support = iota
	// SupportYes: documented as supported.
	SupportYes
	// SupportNo: documented as not supported.
	SupportNo
)

func (s Support) String() string {
	switch s {
	case SupportYes:
		return "yes"
	case SupportNo:
		return "no"
	}
	return "unknown"
}

// Capabilities is what a provider declares about itself (ADR-0012). Each declared value of a real
// provider must cite the provider's official documentation in Basis; the simulator's values are
// synthetic (decision log §15, S2) and say so.
//
// What the engine does with them today:
//   - MaxSegments, when known, is enforced at acceptance (422 message-too-long).
//   - DuplicateDetection and StatusQuery are not relied on: uncertain submissions are never
//     resubmitted whatever they say (D-095 interim rule).
type Capabilities struct {
	// DeliveryReceipts: whether the provider reports final delivery outcomes.
	DeliveryReceipts Support
	// StatusQuery: whether the provider can be asked for the status of a submission.
	StatusQuery Support
	// DuplicateDetection: whether the provider itself discards duplicate submissions.
	DuplicateDetection Support
	// EncodingGSM7 and EncodingUCS2: supported message encodings.
	EncodingGSM7 Support
	EncodingUCS2 Support
	// MaxSegments is the largest number of segments per message, or 0 when unknown.
	MaxSegments int
	// Basis states where these values come from: an official document and version for a real
	// provider, or "synthetic" for the simulator.
	Basis string
}
