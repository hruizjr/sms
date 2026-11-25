package publicapi_test

// Tests for the generated public API types. They prove that the generated code matches the contract
// in the places where Go's JSON handling could silently change meaning (absent vs null vs a value),
// and that every operation in the specification has a generated method. Generated code itself is
// never edited (see generate.go); these tests live in a separate, hand-written file.

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hruizjr/sms/backend/internal/publicapi"
)

const specPath = "../../../contracts/openapi/public-api-v1.yaml"

func fixedTime() time.Time { return time.Date(2026, 11, 25, 8, 0, 0, 0, time.UTC) }

func strptr(s string) *string { return &s }

func baseMessage() publicapi.Message {
	return publicapi.Message{
		Id: "msg_01JDX3P8Q4ZK9R2V6T0B7N5M1C", To: "+639000000001", SenderId: "EXAMPLECO",
		Status: publicapi.Accepted, StatusCategory: publicapi.InProgress,
		Encoding: publicapi.Gsm7, SegmentCount: 1, AcceptedAt: fixedTime(), StatusUpdatedAt: fixedTime(),
	}
}

func marshalFields(t *testing.T, v any) map[string]json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func decodeStrict(t *testing.T, doc string, v any) {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(doc))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		t.Fatalf("decode %s: %v", doc, err)
	}
}

// Message: client_reference and status_reason are REQUIRED and NULLABLE in the contract. The
// generated pointers have no omitempty, so the keys are always present: null when nil, the value
// otherwise. They must never be omitted.
func TestMessageRequiredNullableFieldsAreAlwaysPresent(t *testing.T) {
	m := marshalFields(t, baseMessage())
	for _, field := range []string{"client_reference", "status_reason"} {
		if v, ok := m[field]; !ok || string(v) != "null" {
			t.Errorf("nil %s: got %s (present %v), want null", field, v, ok)
		}
	}

	populated := baseMessage()
	populated.ClientReference = strptr("login-7f3a2c")
	populated.Status, populated.StatusCategory = publicapi.Unknown, publicapi.Uncertain
	populated.StatusReason = &publicapi.StatusReason{Code: "no_final_report_received", Description: "No final report."}
	m = marshalFields(t, populated)
	if string(m["client_reference"]) != `"login-7f3a2c"` {
		t.Errorf("client_reference = %s", m["client_reference"])
	}
	if string(m["status_reason"]) != `{"code":"no_final_report_received","description":"No final report."}` {
		t.Errorf("status_reason = %s", m["status_reason"])
	}

	// Every required property of Message is emitted, whatever its value.
	want := []string{"accepted_at", "client_reference", "encoding", "id", "segment_count", "sender_id",
		"status", "status_category", "status_reason", "status_updated_at", "to"}
	got := make([]string, 0, len(m))
	for k := range m {
		got = append(got, k)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("Message fields = %v, want %v", got, want)
	}
}

func TestMessageDecodesNullAndValues(t *testing.T) {
	const doc = `{"id":"msg_1","client_reference":null,"to":"+639000000001","sender_id":"EXAMPLECO",` +
		`"status":"rejected","status_category":"final",` +
		`"status_reason":{"code":"recipient_opted_out","description":"Opted out."},` +
		`"encoding":"ucs2","segment_count":2,"accepted_at":"2026-11-25T08:00:00Z","status_updated_at":"2026-11-25T08:00:05Z"}`
	var m publicapi.Message
	decodeStrict(t, doc, &m)
	if m.ClientReference != nil {
		t.Errorf("client_reference null decoded as %q", *m.ClientReference)
	}
	if m.StatusReason == nil || m.StatusReason.Code != "recipient_opted_out" {
		t.Errorf("status_reason = %+v", m.StatusReason)
	}
	if m.Encoding != publicapi.Ucs2 || m.SegmentCount != 2 || !m.AcceptedAt.Equal(fixedTime()) {
		t.Errorf("message = %+v", m)
	}
}

// MessageList.next_cursor is REQUIRED and NULLABLE: null on the last page, a string otherwise.
func TestMessageListNextCursor(t *testing.T) {
	last := marshalFields(t, publicapi.MessageList{Data: []publicapi.Message{}})
	if string(last["next_cursor"]) != "null" || string(last["data"]) != "[]" {
		t.Errorf("last page = next_cursor %s, data %s", last["next_cursor"], last["data"])
	}
	more := marshalFields(t, publicapi.MessageList{Data: []publicapi.Message{}, NextCursor: strptr("c2")})
	if string(more["next_cursor"]) != `"c2"` {
		t.Errorf("next_cursor = %s", more["next_cursor"])
	}
}

// MessageCreateRequest.client_reference is OPTIONAL and NOT nullable. Absent decodes to nil and a
// value decodes to the value. JSON null also decodes to nil: Go's decoder cannot tell null from
// absent here, and the contract does not allow null. Rejecting null is part of full request
// validation, deferred to the first endpoint implementation (owner decision D6); see
// TestPublicAPIBindingIsNotValidation in internal/httpserver.
func TestMessageCreateRequestOptionalClientReference(t *testing.T) {
	var absent, populated, null publicapi.MessageCreateRequest
	decodeStrict(t, `{"to":"+639000000001","sender_id":"EXAMPLECO","body":"x"}`, &absent)
	decodeStrict(t, `{"to":"+639000000001","sender_id":"EXAMPLECO","body":"x","client_reference":"r1"}`, &populated)
	decodeStrict(t, `{"to":"+639000000001","sender_id":"EXAMPLECO","body":"x","client_reference":null}`, &null)
	if absent.ClientReference != nil {
		t.Error("absent client_reference must be nil")
	}
	if populated.ClientReference == nil || *populated.ClientReference != "r1" {
		t.Errorf("client_reference = %v", populated.ClientReference)
	}
	if null.ClientReference != nil {
		t.Error("null client_reference decodes to nil (rejection is deferred validation)")
	}
	// Encoding an absent optional field omits it rather than sending null.
	if _, ok := marshalFields(t, absent)["client_reference"]; ok {
		t.Error("absent client_reference must be omitted, not null")
	}
}

// Problem: only type, title and status are required; the optional members are omitted when unset,
// never sent as null (they are not nullable in the contract).
func TestProblemOptionalMembersAreOmitted(t *testing.T) {
	m := marshalFields(t, publicapi.Problem{Type: "about:blank", Title: "Not Found", Status: 404})
	got := make([]string, 0, len(m))
	for k := range m {
		got = append(got, k)
	}
	slices.Sort(got)
	if want := []string{"status", "title", "type"}; !slices.Equal(got, want) {
		t.Errorf("minimal problem fields = %v, want %v", got, want)
	}

	param := "limit"
	full := marshalFields(t, publicapi.Problem{
		Type: "https://api.example.invalid/problems/validation-error", Title: "The request is invalid", Status: 400,
		Detail: strptr("One or more parameters are invalid."), RequestId: strptr("r1"),
		Errors: &[]publicapi.FieldError{{Parameter: &param, Code: "invalid_value", Detail: "d"}},
	})
	if string(full["errors"]) != `[{"code":"invalid_value","detail":"d","parameter":"limit"}]` {
		t.Errorf("errors = %s (pointer must be omitted when unset)", full["errors"])
	}
}

// The spec's own examples decode into the generated types with no unknown fields.
func TestSpecExampleDocumentsDecode(t *testing.T) {
	var p publicapi.Problem
	decodeStrict(t, `{"type":"https://api.example.invalid/problems/validation-error","title":"The request is invalid",`+
		`"status":400,"detail":"One or more fields are invalid.","request_id":"9b2f4c1e7d3a5b60",`+
		`"errors":[{"pointer":"/to","code":"invalid_phone_number","detail":"Must be an E.164 phone number, for example +639000000001."}]}`, &p)
	if p.Errors == nil || (*p.Errors)[0].Pointer == nil || *(*p.Errors)[0].Pointer != "/to" || (*p.Errors)[0].Parameter != nil {
		t.Errorf("problem = %+v", p)
	}
}

// The generated enums carry exactly the contract's values (owner decision D-099: eight statuses).
func TestGeneratedEnums(t *testing.T) {
	statuses := []publicapi.MessageStatus{publicapi.Accepted, publicapi.ProviderAccepted, publicapi.SubmissionUncertain,
		publicapi.Delivered, publicapi.Undelivered, publicapi.Unknown, publicapi.Rejected, publicapi.Failed}
	want := []string{"accepted", "provider_accepted", "submission_uncertain", "delivered", "undelivered", "unknown", "rejected", "failed"}
	if got := enumStrings(statuses); !slices.Equal(got, want) || !slices.Equal(want, specEnum(t, "MessageStatus")) {
		t.Errorf("statuses: generated %v, spec %v, want %v", got, specEnum(t, "MessageStatus"), want)
	}
	for _, s := range statuses {
		if !s.Valid() {
			t.Errorf("%q not Valid()", s)
		}
	}
	if publicapi.MessageStatus("sent").Valid() {
		t.Error("an unknown status must not be Valid()")
	}
	categories := enumStrings([]publicapi.StatusCategory{publicapi.InProgress, publicapi.Uncertain, publicapi.Final})
	if !slices.Equal(categories, specEnum(t, "StatusCategory")) {
		t.Errorf("categories %v, spec %v", categories, specEnum(t, "StatusCategory"))
	}
}

func enumStrings[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

func readSpec(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read the specification: %v", err)
	}
	return string(bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")))
}

// specEnum reads the enum of a top-level schema from the specification. It relies on the
// specification's layout (four-space schema names, block or flow enums), which Redocly lint keeps
// stable; it fails loudly rather than guessing if the layout changes.
func specEnum(t *testing.T, schema string) []string {
	t.Helper()
	spec := readSpec(t)
	start := strings.Index(spec, "\n    "+schema+":\n")
	if start < 0 {
		t.Fatalf("schema %s not found in the specification", schema)
	}
	block := spec[start+1:]
	if end := regexp.MustCompile(`\n    [A-Za-z]+:\n`).FindStringIndex(block[1:]); end != nil {
		block = block[:end[0]+1]
	}
	if m := regexp.MustCompile(`\n      enum: \[([^\]]*)\]`).FindStringSubmatch(block); m != nil {
		var out []string
		for v := range strings.SplitSeq(m[1], ",") {
			out = append(out, strings.TrimSpace(v))
		}
		return out
	}
	i := strings.Index(block, "\n      enum:\n")
	if i < 0 {
		t.Fatalf("no enum in schema %s", schema)
	}
	var out []string
	for line := range strings.SplitSeq(block[i+len("\n      enum:\n"):], "\n") {
		v, ok := strings.CutPrefix(line, "        - ")
		if !ok {
			break
		}
		out = append(out, strings.TrimSpace(v))
	}
	return out
}

// Every operationId in the specification has exactly one method on the generated interfaces, and
// vice versa. Regeneration (checked in CI) keeps this true; the test catches a stale or partial
// generation and makes the coverage explicit.
func TestEveryOperationIsGenerated(t *testing.T) {
	var specOps []string
	for _, m := range regexp.MustCompile(`(?m)^\s+operationId:\s*(\S+)\s*$`).FindAllStringSubmatch(readSpec(t), -1) {
		specOps = append(specOps, strings.ToUpper(m[1][:1])+m[1][1:])
	}
	slices.Sort(specOps)
	if len(specOps) == 0 {
		t.Fatal("no operationId found in the specification")
	}
	for _, iface := range []reflect.Type{
		reflect.TypeFor[publicapi.StrictServerInterface](),
		reflect.TypeFor[publicapi.ServerInterface](),
	} {
		var methods []string
		for i := range iface.NumMethod() {
			methods = append(methods, iface.Method(i).Name)
		}
		if !slices.Equal(methods, specOps) {
			t.Errorf("%s methods = %v, spec operations = %v", iface.Name(), methods, specOps)
		}
	}
}
