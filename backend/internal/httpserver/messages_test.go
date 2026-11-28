package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hruizjr/sms/backend/internal/apikey"
	"github.com/hruizjr/sms/backend/internal/messaging"
	"github.com/hruizjr/sms/backend/internal/publicapi"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// Unit tests of the public API v1 handlers with fake services: authentication and permissions,
// request validation against the contract, and the mapping of results and errors to responses.

const validToken = "valid-token"

var testTenant = tenancy.ID{0x01, 0x93, 0, 0, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0, 1}

type fakeAuth struct {
	scopes []apikey.Scope
	err    error
}

func (fakeAuth) RecordUse(context.Context, apikey.Principal) error { return nil }

func (f fakeAuth) Authenticate(_ context.Context, token string) (apikey.Principal, error) {
	if f.err != nil {
		return apikey.Principal{}, f.err
	}
	if token != validToken {
		return apikey.Principal{}, apikey.ErrUnauthenticated
	}
	return apikey.Principal{TenantID: testTenant, Scopes: f.scopes}, nil
}

type fakeMessages struct {
	accepted   []messaging.AcceptRequest
	acceptRes  messaging.AcceptResult
	acceptErr  error
	get        messaging.Message
	getErr     error
	listFilter messaging.ListFilter
	list       []messaging.Message
	listNext   string
	listErr    error
	tenant     tenancy.ID
}

func (f *fakeMessages) Accept(_ context.Context, tenantID tenancy.ID, req messaging.AcceptRequest) (messaging.AcceptResult, error) {
	f.tenant = tenantID
	f.accepted = append(f.accepted, req)
	return f.acceptRes, f.acceptErr
}

func (f *fakeMessages) Get(_ context.Context, tenantID tenancy.ID, _ messaging.MessageID) (messaging.Message, error) {
	f.tenant = tenantID
	return f.get, f.getErr
}

func (f *fakeMessages) List(_ context.Context, tenantID tenancy.ID, filter messaging.ListFilter) ([]messaging.Message, string, error) {
	f.tenant = tenantID
	f.listFilter = filter
	return f.list, f.listNext, f.listErr
}

var sampleMessage = messaging.Message{
	ID: messaging.MessageID{0x01, 0x93, 1, 2, 3, 4, 0x70, 5, 0x80, 6, 7, 8, 9, 10, 11, 12}, Destination: "+639000000001",
	SenderID: "EXAMPLECO", State: messaging.StateQueued, Encoding: messaging.EncodingGSM7, SegmentCount: 1,
	AcceptedAt: time.Date(2026, 11, 27, 8, 0, 0, 0, time.UTC), StatusUpdatedAt: time.Date(2026, 11, 27, 8, 0, 0, 0, time.UTC),
}

func messagesRouter(auth Authenticator, svc MessageService) http.Handler {
	deps, _ := testDeps(true)
	deps.PublicAPI = &PublicAPIDeps{Messages: svc, Auth: auth}
	return NewRouter(deps)
}

func allScopes() fakeAuth {
	return fakeAuth{scopes: []apikey.Scope{apikey.ScopeMessagesSend, apikey.ScopeMessagesRead}}
}

func authedRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()
	req := newRequest(t, method, target, body)
	req.Header.Set("Authorization", "Bearer "+validToken)
	if method == http.MethodPost {
		req.Header.Set("Idempotency-Key", "key-1")
	}
	return req
}

const validBody = `{"to":"+639000000001","sender_id":"EXAMPLECO","body":"Hello","client_reference":"ref-1"}`

func TestAuthentication(t *testing.T) {
	t.Parallel()
	svc := &fakeMessages{acceptRes: messaging.AcceptResult{Message: sampleMessage}}
	tests := []struct {
		name   string
		auth   fakeAuth
		header []string
		method string
		status int
		slug   string
	}{
		{"no header", allScopes(), nil, http.MethodGet, 401, "unauthenticated"},
		{"basic scheme", allScopes(), []string{"Basic " + validToken}, http.MethodGet, 401, "unauthenticated"},
		{"empty bearer", allScopes(), []string{"Bearer "}, http.MethodGet, 401, "unauthenticated"},
		{"two headers", allScopes(), []string{"Bearer " + validToken, "Bearer " + validToken}, http.MethodGet, 401, "unauthenticated"},
		{"unknown key", allScopes(), []string{"Bearer other"}, http.MethodGet, 401, "unauthenticated"},
		{"lower-case scheme", allScopes(), []string{"bearer " + validToken}, http.MethodGet, 200, ""},
		{"inactive tenant", fakeAuth{err: apikey.ErrTenantNotActive}, []string{"Bearer " + validToken}, http.MethodGet, 403, "forbidden"},
		{"read key cannot send", fakeAuth{scopes: []apikey.Scope{apikey.ScopeMessagesRead}}, []string{"Bearer " + validToken}, http.MethodPost, 403, "forbidden"},
		{"send key cannot read", fakeAuth{scopes: []apikey.Scope{apikey.ScopeMessagesSend}}, []string{"Bearer " + validToken}, http.MethodGet, 403, "forbidden"},
		// Authentication runs before parameter binding: no Idempotency-Key and no key is a 401.
		{"unauthenticated POST without Idempotency-Key", allScopes(), nil, http.MethodPost, 401, "unauthenticated"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := messagesRouter(tc.auth, svc)
			target := "/v1/messages"
			req := newRequest(t, tc.method, target, "")
			if tc.method == http.MethodPost {
				req = newRequest(t, tc.method, target, validBody)
				if tc.header != nil {
					req.Header.Set("Idempotency-Key", "k")
				}
			}
			for _, v := range tc.header {
				req.Header.Add("Authorization", v)
			}
			rec, body := serve(h, req)
			if rec.Code != tc.status {
				t.Fatalf("status %d, want %d: %s", rec.Code, tc.status, body)
			}
			if tc.slug == "" {
				return
			}
			p := decodePublicProblem(t, rec, body)
			if p.Type != problemTypeBase+tc.slug {
				t.Errorf("type %s, want %s", p.Type, tc.slug)
			}
			if tc.status == 401 && !strings.HasPrefix(rec.Header().Get("WWW-Authenticate"), "Bearer") {
				t.Errorf("WWW-Authenticate = %q", rec.Header().Get("WWW-Authenticate"))
			}
		})
	}
	if svc.tenant != testTenant && svc.tenant != (tenancy.ID{}) {
		t.Errorf("tenant %v did not come from the key", svc.tenant)
	}
}

func TestCreateMessageValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		contentType string
		body        string
		key         string
		status      int
		slug        string
		errors      []string // "pointer code"
	}{
		{"text body", "text/plain", validBody, "k", 415, "unsupported-media-type", nil},
		{"too large", "", `{"body":"` + strings.Repeat("a", 20000) + `"}`, "k", 413, "payload-too-large", nil},
		{"not JSON", "", `{"to":`, "k", 400, "malformed-request", nil},
		{"trailing data", "", validBody + ` {}`, "k", 400, "malformed-request", nil},
		{"array", "", `[1]`, "k", 400, "validation-error", []string{" invalid_type"}},
		{"empty object", "", `{}`, "k", 400, "validation-error", []string{"/to required", "/sender_id required", "/body required"}},
		{"nulls and wrong types", "", `{"to":null,"sender_id":7,"body":"x","client_reference":null}`, "k", 400, "validation-error",
			[]string{"/to invalid_type", "/sender_id invalid_type", "/client_reference invalid_type"}},
		{"unknown fields", "", `{"to":"+639000000001","sender_id":"S","body":"x","priority":1,"a/b":2}`, "k", 400, "validation-error",
			[]string{"/a~1b unknown_field", "/priority unknown_field"}},
		{"field formats", "", `{"to":"09171234567","sender_id":"","body":"` + strings.Repeat("é", 1601) + `","client_reference":"has space"}`, "k", 400,
			"validation-error", []string{"/to invalid_phone_number", "/sender_id too_short", "/body too_long", "/client_reference invalid_format"}},
		{"sender ID too long", "", `{"to":"+639000000001","sender_id":"` + strings.Repeat("S", 33) + `","body":"x"}`, "k", 400, "validation-error",
			[]string{"/sender_id too_long"}},
		{"idempotency key with a space", "", validBody, "has space", 400, "idempotency-key-invalid", nil},
		{"idempotency key too long", "", validBody, strings.Repeat("k", 256), 400, "idempotency-key-invalid", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := &fakeMessages{}
			h := messagesRouter(allScopes(), svc)
			req := authedRequest(t, http.MethodPost, "/v1/messages", tc.body)
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}
			req.Header.Set("Idempotency-Key", tc.key)
			rec, body := serve(h, req)
			if rec.Code != tc.status {
				t.Fatalf("status %d, want %d: %s", rec.Code, tc.status, body)
			}
			p := decodePublicProblem(t, rec, body)
			if p.Type != problemTypeBase+tc.slug {
				t.Errorf("type %s, want %s", p.Type, tc.slug)
			}
			var got []string
			if p.Errors != nil {
				for _, e := range *p.Errors {
					got = append(got, *e.Pointer+" "+e.Code)
				}
			}
			slices.Sort(got)
			want := slices.Clone(tc.errors)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Errorf("errors %v, want %v", got, want)
			}
			if len(svc.accepted) != 0 {
				t.Error("an invalid request reached the service")
			}
		})
	}
}

func TestCreateMessageAccepted(t *testing.T) {
	t.Parallel()
	svc := &fakeMessages{acceptRes: messaging.AcceptResult{Message: sampleMessage}}
	h := messagesRouter(allScopes(), svc)
	req := authedRequest(t, http.MethodPost, "/v1/messages", "  {\n \"body\": \"Hello\", \"to\":\"+639000000001\",\"sender_id\":\"EXAMPLECO\"}  ")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Idempotency-Key", `"quoted-key"`)
	rec, body := serve(h, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status %d: %s", rec.Code, body)
	}
	if got := rec.Header().Get("Location"); got != "/v1/messages/"+sampleMessage.ID.String() {
		t.Errorf("Location = %q", got)
	}
	if rec.Header().Get("Idempotent-Replayed") != "" {
		t.Error("an original response must not carry Idempotent-Replayed")
	}
	if len(svc.accepted) != 1 || svc.accepted[0].IdempotencyKey != "quoted-key" || svc.accepted[0].Body != "Hello" || svc.accepted[0].ClientReference != nil || svc.tenant != testTenant {
		t.Errorf("service received %+v for tenant %v", svc.accepted, svc.tenant)
	}
	var fields map[string]any
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"id", "client_reference", "to", "sender_id", "status", "status_category", "status_reason", "encoding", "segment_count", "accepted_at", "status_updated_at"} {
		if _, ok := fields[required]; !ok {
			t.Errorf("response lacks %s: %s", required, body)
		}
	}
	if fields["status"] != "accepted" || fields["status_category"] != "in_progress" || fields["status_reason"] != nil || fields["accepted_at"] != "2026-11-27T08:00:00Z" {
		t.Errorf("response %s", body)
	}
	if strings.Contains(string(body), "Hello") {
		t.Error("the response must not contain the message text")
	}

	svc.acceptRes.Replayed = true
	rec, _ = serve(h, authedRequest(t, http.MethodPost, "/v1/messages", validBody))
	if rec.Code != http.StatusAccepted || rec.Header().Get("Idempotent-Replayed") != "true" {
		t.Errorf("replay: status %d, Idempotent-Replayed %q", rec.Code, rec.Header().Get("Idempotent-Replayed"))
	}
}

func TestCreateMessageErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		err    error
		status int
		slug   string
	}{
		{messaging.ErrIdempotencyKeyReused, 422, "idempotency-key-reused"},
		{messaging.ErrIdempotencyInProgress, 409, "idempotency-request-in-progress"},
		{messaging.ErrSenderIDNotApproved, 422, "sender-id-not-approved"},
		{messaging.ErrDestinationNotSupported, 422, "destination-not-supported"},
		{messaging.ErrMessageTooLong, 422, "message-too-long"},
	}
	for _, tc := range tests {
		h := messagesRouter(allScopes(), &fakeMessages{acceptErr: tc.err})
		rec, body := serve(h, authedRequest(t, http.MethodPost, "/v1/messages", validBody))
		if rec.Code != tc.status {
			t.Errorf("%v: status %d, want %d", tc.err, rec.Code, tc.status)
			continue
		}
		if p := decodePublicProblem(t, rec, body); p.Type != problemTypeBase+tc.slug {
			t.Errorf("%v: type %s", tc.err, p.Type)
		}
		if tc.status == 409 && rec.Header().Get("Retry-After") == "" {
			t.Error("409 without Retry-After")
		}
	}
	h := messagesRouter(allScopes(), &fakeMessages{acceptErr: context.DeadlineExceeded})
	rec, body := serve(h, authedRequest(t, http.MethodPost, "/v1/messages", validBody))
	if rec.Code != 500 || strings.Contains(string(body), "deadline") {
		t.Errorf("internal error: status %d, body %s", rec.Code, body)
	}
}

func TestGetMessage(t *testing.T) {
	t.Parallel()
	uncertain := sampleMessage
	uncertain.State = messaging.StateSubmitUncertain
	reason := messaging.ReasonSubmissionOutcomeUnknown
	uncertain.Reason = &reason
	svc := &fakeMessages{get: uncertain}
	h := messagesRouter(allScopes(), svc)

	rec, body := serve(h, authedRequest(t, http.MethodGet, "/v1/messages/"+uncertain.ID.String(), ""))
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, body)
	}
	var m publicapi.Message
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	if m.Status != "submission_uncertain" || m.StatusCategory != "uncertain" || m.StatusReason == nil ||
		m.StatusReason.Code != "submission_outcome_unknown" || m.StatusReason.Description == "" {
		t.Errorf("message %+v", m)
	}
	for _, internal := range []string{"provider_id", "provider_message_ref", "attempt", "sim-"} {
		if strings.Contains(string(body), internal) {
			t.Errorf("internal provider detail %q exposed: %s", internal, body)
		}
	}

	for target, status := range map[string]int{
		"/v1/messages/has.dot":                        400,
		"/v1/messages/msg_" + strings.Repeat("A", 61): 400,
		"/v1/messages/msg_notours":                    404,
		"/v1/messages/" + uncertain.ID.String():       404, // the service reports another tenant's message as not found
	} {
		svc.getErr = messaging.ErrNotFound
		rec, body := serve(h, authedRequest(t, http.MethodGet, target, ""))
		if rec.Code != status {
			t.Errorf("%s: status %d, want %d: %s", target, rec.Code, status, body)
		}
	}
}

func TestListMessages(t *testing.T) {
	t.Parallel()
	svc := &fakeMessages{list: []messaging.Message{sampleMessage}, listNext: "next-page"}
	h := messagesRouter(allScopes(), svc)
	rec, body := serve(h, authedRequest(t, http.MethodGet, "/v1/messages?limit=10&status=delivered&client_reference=ref-1&cursor=abc&accepted_after=2026-11-01T00:00:00Z", ""))
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, body)
	}
	f := svc.listFilter
	if f.Limit != 10 || f.Status == nil || *f.Status != messaging.StatusDelivered || f.ClientReference == nil || f.Cursor != "abc" || f.AcceptedAfter == nil {
		t.Errorf("filter %+v", f)
	}
	var page publicapi.MessageList
	if err := json.Unmarshal(body, &page); err != nil || len(page.Data) != 1 || page.NextCursor == nil || *page.NextCursor != "next-page" {
		t.Errorf("page %s, %v", body, err)
	}

	svc.listNext = ""
	rec, body = serve(h, authedRequest(t, http.MethodGet, "/v1/messages", ""))
	if rec.Code != 200 || !strings.Contains(string(body), `"next_cursor":null`) || svc.listFilter.Limit != 50 {
		t.Errorf("default page: %d %s, limit %d", rec.Code, body, svc.listFilter.Limit)
	}

	for _, q := range []string{"limit=0", "limit=101", "status=sent", "client_reference=a%20b"} {
		rec, body := serve(h, authedRequest(t, http.MethodGet, "/v1/messages?"+q, ""))
		if rec.Code != 400 {
			t.Errorf("%s: status %d, want 400: %s", q, rec.Code, body)
		}
	}
	svc.listErr = messaging.ErrInvalidCursor
	rec, body = serve(h, authedRequest(t, http.MethodGet, "/v1/messages?cursor=stale", ""))
	if p := decodePublicProblem(t, rec, body); rec.Code != 400 || p.Type != problemTypeBase+"validation-error" {
		t.Errorf("invalid cursor: %d %+v", rec.Code, p)
	}
}
