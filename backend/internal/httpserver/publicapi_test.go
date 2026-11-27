package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hruizjr/sms/backend/internal/publicapi"
)

// recordingAPI is a test-only implementation of the generated strict interface. It records the
// typed request it received and returns a canned response or error.
type recordingAPI struct {
	mu       sync.Mutex
	list     *publicapi.ListMessagesRequestObject
	create   *publicapi.CreateMessageRequestObject
	get      *publicapi.GetMessageRequestObject
	failWith error
}

// Compile-time proof that the test implementation satisfies the generated interface: if the
// specification gains, loses or changes an operation, regeneration breaks this line.
var _ publicapi.StrictServerInterface = (*recordingAPI)(nil)

func (a *recordingAPI) ListMessages(_ context.Context, req publicapi.ListMessagesRequestObject) (publicapi.ListMessagesResponseObject, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.list = &req
	if a.failWith != nil {
		return nil, a.failWith
	}
	return publicapi.ListMessages200JSONResponse{Body: publicapi.MessageList{Data: []publicapi.Message{}}}, nil
}

func (a *recordingAPI) CreateMessage(_ context.Context, req publicapi.CreateMessageRequestObject) (publicapi.CreateMessageResponseObject, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.create = &req
	if a.failWith != nil {
		return nil, a.failWith
	}
	return publicapi.CreateMessage202JSONResponse{Body: testMessage()}, nil
}

func (a *recordingAPI) GetMessage(_ context.Context, req publicapi.GetMessageRequestObject) (publicapi.GetMessageResponseObject, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.get = &req
	if a.failWith != nil {
		return nil, a.failWith
	}
	return publicapi.GetMessage200JSONResponse{Body: testMessage()}, nil
}

// testMessage is a synthetic message (synthetic number range, AGENTS.md rule 20).
func testMessage() publicapi.Message {
	at := time.Date(2026, 11, 25, 8, 0, 0, 0, time.UTC)
	return publicapi.Message{
		Id: "msg_01JDX3P8Q4ZK9R2V6T0B7N5M1C", To: "+639000000001", SenderId: "EXAMPLECO",
		Status: publicapi.Accepted, StatusCategory: publicapi.InProgress,
		Encoding: publicapi.Gsm7, SegmentCount: 1, AcceptedAt: at, StatusUpdatedAt: at,
	}
}

// publicAPIRouter returns the production middleware chain with the public API mounted the way
// the first endpoint implementation will mount it.
func publicAPIRouter(api *recordingAPI) (http.Handler, *safeBuffer) {
	deps, logs := testDeps(true)
	return newRouter(deps, func(r chi.Router) { mountPublicAPI(r, api, deps.Logger) }), logs
}

func serve(h http.Handler, req *http.Request) (*httptest.ResponseRecorder, []byte) {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec, rec.Body.Bytes()
}

func newRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequestWithContext(context.Background(), method, target, nil)
	} else {
		req = httptest.NewRequestWithContext(context.Background(), method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func decodePublicProblem(t *testing.T, rec *httptest.ResponseRecorder, body []byte) publicapi.Problem {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json (body %s)", ct, body)
	}
	var p publicapi.Problem
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		t.Fatalf("decode problem %s: %v", body, err)
	}
	if p.Status != rec.Code {
		t.Errorf("problem status %d, response status %d", p.Status, rec.Code)
	}
	if p.RequestId == nil || *p.RequestId != "req-test" {
		t.Errorf("request_id = %v, want req-test", p.RequestId)
	}
	return p
}

// Without its dependencies (RouterDeps.PublicAPI), the router does not serve /v1 (owner decision D4
// until Phase 4 Group 2): the responses stay exactly what Groups 1-3 produced for any unknown path.
func TestProductionRouterDoesNotMountPublicAPI(t *testing.T) {
	deps, _ := testDeps(true)
	h := NewRouter(deps)
	for _, tc := range []struct{ method, target string }{
		{http.MethodGet, "/v1/messages"},
		{http.MethodPost, "/v1/messages"},
		{http.MethodGet, "/v1/messages/msg_01JDX3P8Q4ZK9R2V6T0B7N5M1C"},
	} {
		rec, body := do(t, h, tc.method, tc.target)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s %s: status %d, want 404", tc.method, tc.target, rec.Code)
			continue
		}
		p := decodeProblem(t, rec, body)
		if p.Type != "about:blank" || p.Title != "Not Found" {
			t.Errorf("%s %s: problem %+v, want the unchanged about:blank 404", tc.method, tc.target, p)
		}
	}
}

func TestPublicAPIRoutesAndBindsListParameters(t *testing.T) {
	api := &recordingAPI{}
	h, _ := publicAPIRouter(api)
	req := newRequest(t, http.MethodGet, "/v1/messages?client_reference=login-7f3a2c&status=delivered"+
		"&accepted_after=2026-11-25T00:00:00Z&accepted_before=2026-11-26T00:00:00Z&limit=25&cursor=abc", "")
	req.Header.Set("X-Request-ID", "client-req-1")

	rec, body := serve(h, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, body)
	}
	if got := rec.Header().Get("X-Request-ID"); got != "client-req-1" {
		t.Errorf("X-Request-ID response header = %q; the existing middleware must still apply", got)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("security headers missing on a public API route")
	}
	p := api.list.Params
	if p.ClientReference == nil || *p.ClientReference != "login-7f3a2c" ||
		p.Status == nil || *p.Status != publicapi.Delivered ||
		p.AcceptedAfter == nil || !p.AcceptedAfter.Equal(time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC)) ||
		p.AcceptedBefore == nil || !p.AcceptedBefore.Equal(time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)) ||
		p.Limit == nil || *p.Limit != 25 ||
		p.Cursor == nil || *p.Cursor != "abc" ||
		p.XRequestID == nil || *p.XRequestID != "client-req-1" {
		t.Errorf("bound parameters = %+v", p)
	}
	var list publicapi.MessageList
	if err := json.Unmarshal(body, &list); err != nil || list.Data == nil {
		t.Errorf("response %s: %v", body, err)
	}
}

// Absent optional parameters stay distinguishable from supplied ones: they are nil, never a zero
// value (an absent limit is not limit=0; the contract's default of 50 is applied by the endpoint).
func TestPublicAPIAbsentOptionalParametersAreNil(t *testing.T) {
	api := &recordingAPI{}
	h, _ := publicAPIRouter(api)
	if rec, body := serve(h, newRequest(t, http.MethodGet, "/v1/messages", "")); rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, body)
	}
	p := api.list.Params
	if p.ClientReference != nil || p.Status != nil || p.AcceptedAfter != nil || p.AcceptedBefore != nil ||
		p.Limit != nil || p.Cursor != nil || p.XRequestID != nil {
		t.Errorf("absent parameters must be nil, got %+v", p)
	}
}

func TestPublicAPIRoutesGetMessage(t *testing.T) {
	api := &recordingAPI{}
	h, _ := publicAPIRouter(api)
	rec, body := serve(h, newRequest(t, http.MethodGet, "/v1/messages/msg_01JDX3P8Q4ZK9R2V6T0B7N5M1C", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rec.Code, body)
	}
	if api.get.MessageId != "msg_01JDX3P8Q4ZK9R2V6T0B7N5M1C" {
		t.Errorf("message_id = %q", api.get.MessageId)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	// Required-but-nullable fields are present as null, never omitted (contract: Message.required).
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"client_reference", "status_reason"} {
		if v, ok := raw[field]; !ok || string(v) != "null" {
			t.Errorf("%s = %s (present %v), want null", field, v, ok)
		}
	}
}

func TestPublicAPICreateMessageBindsHeaderAndBody(t *testing.T) {
	api := &recordingAPI{}
	h, _ := publicAPIRouter(api)
	req := newRequest(t, http.MethodPost, "/v1/messages",
		`{"to":"+639000000001","sender_id":"EXAMPLECO","body":"Your code is 000000","client_reference":"login-7f3a2c"}`)
	req.Header.Set("Idempotency-Key", "6f1c2d9e-0b7a-4c55-9d1e-3a8b7c6d5e4f")

	rec, body := serve(h, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status %d, body %s", rec.Code, body)
	}
	got := api.create
	if got.Params.IdempotencyKey != "6f1c2d9e-0b7a-4c55-9d1e-3a8b7c6d5e4f" {
		t.Errorf("Idempotency-Key = %q", got.Params.IdempotencyKey)
	}
	if got.Body == nil || got.Body.To != "+639000000001" || got.Body.SenderId != "EXAMPLECO" ||
		got.Body.ClientReference == nil || *got.Body.ClientReference != "login-7f3a2c" {
		t.Errorf("body = %+v", got.Body)
	}
}

func TestPublicAPIMissingIdempotencyKey(t *testing.T) {
	api := &recordingAPI{}
	h, _ := publicAPIRouter(api)
	rec, body := serve(h, newRequest(t, http.MethodPost, "/v1/messages",
		`{"to":"+639000000001","sender_id":"EXAMPLECO","body":"x"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, body %s", rec.Code, body)
	}
	p := decodePublicProblem(t, rec, body)
	if p.Type != problemTypeBase+"idempotency-key-missing" {
		t.Errorf("type = %q", p.Type)
	}
	if api.create != nil {
		t.Error("handler must not run without an Idempotency-Key")
	}
}

func TestPublicAPIParameterBindingErrors(t *testing.T) {
	for _, tc := range []struct {
		name, target, param, code string
		header                    http.Header
	}{
		{name: "non-integer limit", target: "/v1/messages?limit=ten", param: "limit", code: "invalid_value"},
		{name: "non-date accepted_after", target: "/v1/messages?accepted_after=yesterday", param: "accepted_after", code: "invalid_value"},
		{name: "repeated limit", target: "/v1/messages?limit=1&limit=2", param: "limit", code: "invalid_value"},
		{name: "repeated X-Request-ID", target: "/v1/messages", param: "X-Request-ID", code: "multiple_values",
			// Values the request-ID middleware ignores (invalid characters), so the generated ID applies.
			header: http.Header{"X-Request-Id": {"bad id!", "bad id?"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &recordingAPI{}
			h, _ := publicAPIRouter(api)
			req := newRequest(t, http.MethodGet, tc.target, "")
			for k, v := range tc.header {
				req.Header[k] = v
			}
			rec, body := serve(h, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status %d, body %s", rec.Code, body)
			}
			p := decodePublicProblem(t, rec, body)
			if p.Type != problemTypeBase+"validation-error" || p.Errors == nil || len(*p.Errors) != 1 {
				t.Fatalf("problem = %s", body)
			}
			fe := (*p.Errors)[0]
			if fe.Parameter == nil || *fe.Parameter != tc.param || fe.Code != tc.code || fe.Pointer != nil {
				t.Errorf("field error = %+v", fe)
			}
			// Fixed, client-safe text only: never the Go error text, which echoes request input.
			for _, leak := range []string{"ten", "yesterday", "strconv", "parsing", "error"} {
				if strings.Contains(fe.Detail, leak) || strings.Contains(*p.Detail, leak) {
					t.Errorf("problem leaks %q: %s", leak, body)
				}
			}
			if api.list != nil {
				t.Error("handler must not run after a binding error")
			}
		})
	}
}

func TestPublicAPIMalformedJSONBody(t *testing.T) {
	api := &recordingAPI{}
	h, _ := publicAPIRouter(api)
	req := newRequest(t, http.MethodPost, "/v1/messages", `{"to": "+639000000001", SECRET-ish`)
	req.Header.Set("Idempotency-Key", "k1")
	rec, body := serve(h, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, body %s", rec.Code, body)
	}
	p := decodePublicProblem(t, rec, body)
	if p.Type != problemTypeBase+"malformed-request" || strings.Contains(string(body), "SECRET") {
		t.Errorf("problem = %s", body)
	}
	if api.create != nil {
		t.Error("handler must not run with an undecodable body")
	}
}

func TestPublicAPIHandlerErrorIsGeneric500(t *testing.T) {
	api := &recordingAPI{failWith: errors.New("database password=hunter2 rejected")}
	h, logs := publicAPIRouter(api)
	rec, body := serve(h, newRequest(t, http.MethodGet, "/v1/messages/msg_1", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, body %s", rec.Code, body)
	}
	p := decodeProblem(t, rec, body)
	if p.Type != "about:blank" || strings.Contains(string(body), "hunter2") {
		t.Errorf("problem = %s", body)
	}
	if !strings.Contains(logs.String(), "public API handler failed") {
		t.Error("handler failure was not logged")
	}
}

func TestPublicAPIKeepsRouterFallbacks(t *testing.T) {
	h, _ := publicAPIRouter(&recordingAPI{})
	rec, body := serve(h, newRequest(t, http.MethodDelete, "/v1/messages", ""))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE /v1/messages: status %d, body %s", rec.Code, body)
	}
	if p := decodeProblem(t, rec, body); p.Type != "about:blank" {
		t.Errorf("405 problem = %+v", p)
	}
	if rec, _ := serve(h, newRequest(t, http.MethodGet, "/v1/unknown", "")); rec.Code != http.StatusNotFound {
		t.Errorf("GET /v1/unknown: status %d, want 404", rec.Code)
	}
}

// Parameter binding is NOT OpenAPI request validation (owner decision D6). These requests violate
// the contract but reach the handler, which proves that the first endpoint implementation must add
// validation (and authentication) before serving traffic. If a future change adds validation, these
// expectations change deliberately, together with the documentation.
func TestPublicAPIBindingIsNotValidation(t *testing.T) {
	t.Run("out-of-range limit and unknown status reach the handler", func(t *testing.T) {
		api := &recordingAPI{}
		h, _ := publicAPIRouter(api)
		rec, body := serve(h, newRequest(t, http.MethodGet, "/v1/messages?limit=1000&status=not_a_status", ""))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d, body %s", rec.Code, body)
		}
		if *api.list.Params.Limit != 1000 || api.list.Params.Status.Valid() {
			t.Errorf("params = %+v", api.list.Params)
		}
	})
	t.Run("contract-violating body reaches the handler", func(t *testing.T) {
		api := &recordingAPI{}
		h, _ := publicAPIRouter(api)
		// Missing required body, unknown field, and null client_reference: all invalid under the
		// contract (required fields, additionalProperties: false, client_reference not nullable).
		req := newRequest(t, http.MethodPost, "/v1/messages",
			`{"to":"not-a-number","sender_id":"EXAMPLECO","client_reference":null,"unexpected":true}`)
		req.Header.Set("Idempotency-Key", "k1")
		rec, body := serve(h, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status %d, body %s", rec.Code, body)
		}
		if api.create.Body.Body != "" || api.create.Body.ClientReference != nil {
			t.Errorf("body = %+v", api.create.Body)
		}
	})
	t.Run("Idempotency-Key is passed through verbatim", func(t *testing.T) {
		api := &recordingAPI{}
		h, _ := publicAPIRouter(api)
		req := newRequest(t, http.MethodPost, "/v1/messages", `{"to":"+639000000001","sender_id":"EXAMPLECO","body":"x"}`)
		req.Header.Set("Idempotency-Key", `"quoted key"`) // the space is outside the contract's pattern
		if rec, body := serve(h, req); rec.Code != http.StatusAccepted {
			t.Fatalf("status %d, body %s", rec.Code, body)
		}
		if got := api.create.Params.IdempotencyKey; got != `"quoted key"` {
			t.Errorf("Idempotency-Key = %q, want the raw header value", got)
		}
	})
}
