package messaging

import (
	"strings"
	"testing"
)

func TestSegment(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		body     string
		encoding Encoding
		segments int
	}{
		{"short GSM-7", "Your verification code is 000000.", EncodingGSM7, 1},
		{"160 GSM-7 characters fit one segment", strings.Repeat("a", 160), EncodingGSM7, 1},
		{"161 GSM-7 characters need two", strings.Repeat("a", 161), EncodingGSM7, 2},
		{"306 GSM-7 characters fill two", strings.Repeat("a", 306), EncodingGSM7, 2},
		{"307 GSM-7 characters need three", strings.Repeat("a", 307), EncodingGSM7, 3},
		{"extension characters count twice", strings.Repeat("€", 80), EncodingGSM7, 1},
		{"81 extension characters exceed one segment", strings.Repeat("€", 81), EncodingGSM7, 2},
		{"an escape pair is not split", strings.Repeat("a", 152) + "€" + strings.Repeat("a", 10), EncodingGSM7, 2},
		{"GSM-7 accented letters", "Ñandu é à ü ø", EncodingGSM7, 1},
		{"ú is not in GSM-7", "Ñandú", EncodingUCS2, 1},
		{"non-GSM character switches to UCS-2", "Salamat po ✓", EncodingUCS2, 1},
		{"70 UCS-2 characters fit one segment", strings.Repeat("✓", 70), EncodingUCS2, 1},
		{"71 UCS-2 characters need two", strings.Repeat("✓", 71), EncodingUCS2, 2},
		{"emoji take two UTF-16 units", strings.Repeat("😀", 35), EncodingUCS2, 1},
		{"a surrogate pair is not split", strings.Repeat("a", 66) + "😀" + strings.Repeat("a", 10), EncodingUCS2, 2},
		{"1600 characters", strings.Repeat("a", 1600), EncodingGSM7, 11},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Segment(tc.body)
			if got.Encoding != tc.encoding || got.Segments != tc.segments {
				t.Errorf("Segment() = %+v, want %s with %d segments", got, tc.encoding, tc.segments)
			}
		})
	}
}

func TestStatusMapping(t *testing.T) {
	t.Parallel()
	tests := []struct {
		state    State
		status   Status
		category Category
	}{
		{StateQueued, StatusAccepted, CategoryInProgress},
		{StateSubmitting, StatusAccepted, CategoryInProgress},
		{StateSubmitted, StatusProviderAccepted, CategoryInProgress},
		{StateSubmitUncertain, StatusSubmissionUncertain, CategoryUncertain},
		{StateUnknown, StatusUnknown, CategoryUncertain},
		{StateDelivered, StatusDelivered, CategoryFinal},
		{StateUndelivered, StatusUndelivered, CategoryFinal},
		{StateRejected, StatusRejected, CategoryFinal},
		{StateFailed, StatusFailed, CategoryFinal},
	}
	for _, tc := range tests {
		if got := tc.state.Status(); got != tc.status || got.Category() != tc.category {
			t.Errorf("%s: status %s (%s), want %s (%s)", tc.state, got, got.Category(), tc.status, tc.category)
		}
		found := false
		for _, s := range tc.status.States() {
			found = found || s == tc.state
		}
		if !found {
			t.Errorf("%s.States() does not include %s", tc.status, tc.state)
		}
	}
	if Status("no_such_status").States() != nil {
		t.Error("an unknown status must match no state")
	}
}

func TestTransitions(t *testing.T) {
	t.Parallel()
	allowed := []struct{ from, to State }{
		{StateQueued, StateSubmitting},
		{StateQueued, StateFailed},
		{StateSubmitting, StateSubmitted},
		{StateSubmitting, StateRejected},
		{StateSubmitting, StateFailed},
		{StateSubmitting, StateQueued},
		{StateSubmitting, StateSubmitUncertain},
		{StateSubmitted, StateDelivered},
		{StateSubmitted, StateUndelivered},
		{StateSubmitted, StateUnknown},
		{StateSubmitUncertain, StateDelivered},
		{StateUnknown, StateUndelivered},
	}
	for _, tc := range allowed {
		if !CanTransition(tc.from, tc.to) {
			t.Errorf("%s -> %s must be allowed", tc.from, tc.to)
		}
	}
	forbidden := []struct{ from, to State }{
		// An uncertain submission is never retried or failed automatically (D-095 interim rule).
		{StateSubmitUncertain, StateQueued},
		{StateSubmitUncertain, StateSubmitting},
		{StateSubmitUncertain, StateFailed},
		// A final state is never left.
		{StateDelivered, StateUndelivered},
		{StateDelivered, StateSubmitted},
		{StateUndelivered, StateDelivered},
		{StateRejected, StateSubmitting},
		{StateFailed, StateQueued},
		// No delivery without a provider acceptance.
		{StateQueued, StateDelivered},
		{StateSubmitting, StateDelivered},
		{StateUnknown, StateSubmitted},
	}
	for _, tc := range forbidden {
		if CanTransition(tc.from, tc.to) {
			t.Errorf("%s -> %s must not be allowed", tc.from, tc.to)
		}
	}
}

func TestMessageIDRoundTrip(t *testing.T) {
	t.Parallel()
	id := MessageID{0x01, 0x93, 0x4f, 0x2a, 0xff, 0x00, 0x7c, 0x11, 0x82, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0xfe}
	s := id.String()
	if !strings.HasPrefix(s, "msg_") || len(s) != 30 {
		t.Fatalf("String() = %q", s)
	}
	back, err := ParseMessageID(s)
	if err != nil || back != id {
		t.Fatalf("ParseMessageID(%q) = %v, %v", s, back, err)
	}
	for _, bad := range []string{"", "msg_", "msg_" + strings.ToLower(s[4:]), "id_" + s[4:], s + "0", s[:29] + "Z", "msg_ILOU0000000000000000000000"} {
		if _, err := ParseMessageID(bad); err == nil {
			t.Errorf("ParseMessageID(%q) accepted", bad)
		}
	}
}

func TestFingerprintNormalizesTheRequest(t *testing.T) {
	t.Parallel()
	ref := "order-1"
	base := AcceptRequest{Destination: "+639000000001", SenderID: "EXAMPLECO", Body: "hello", ClientReference: &ref, IdempotencyKey: "k1"}
	sameKeyOtherHeaderForm := base
	sameKeyOtherHeaderForm.IdempotencyKey = "different key"
	if Fingerprint(base) != Fingerprint(sameKeyOtherHeaderForm) {
		t.Error("the idempotency key must not be part of the fingerprint")
	}
	for name, change := range map[string]func(*AcceptRequest){
		"to":               func(r *AcceptRequest) { r.Destination = "+639000000002" },
		"sender_id":        func(r *AcceptRequest) { r.SenderID = "OTHER" },
		"body":             func(r *AcceptRequest) { r.Body = "hello!" },
		"client_reference": func(r *AcceptRequest) { r.ClientReference = nil },
	} {
		other := base
		change(&other)
		if Fingerprint(other) == Fingerprint(base) {
			t.Errorf("changing %s must change the fingerprint", name)
		}
	}
}
