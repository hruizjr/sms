package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/hruizjr/sms/backend/internal/messaging"
	"github.com/hruizjr/sms/backend/internal/middleware"
	"github.com/hruizjr/sms/backend/internal/publicapi"
	"github.com/hruizjr/sms/backend/internal/tenancy"
)

// MessageService is what the public API needs from the messaging engine (implemented by
// messaging.Service).
type MessageService interface {
	Accept(ctx context.Context, tenantID tenancy.ID, req messaging.AcceptRequest) (messaging.AcceptResult, error)
	Get(ctx context.Context, tenantID tenancy.ID, id messaging.MessageID) (messaging.Message, error)
	List(ctx context.Context, tenantID tenancy.ID, f messaging.ListFilter) ([]messaging.Message, string, error)
}

// PublicAPIDeps are the dependencies of the public API v1. When they are set, NewRouter mounts
// the /v1 endpoints.
type PublicAPIDeps struct {
	Messages MessageService
	Auth     Authenticator
}

// maxMessageRequestBytes bounds the POST /v1/messages body: 1,600 characters of up to 4 bytes each
// (6 bytes when escaped as \uXXXX), plus the other fields.
const maxMessageRequestBytes = 16 << 10

// Contract patterns (contracts/openapi/public-api-v1.yaml).
var (
	phoneNumberPattern     = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)
	clientReferencePattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,64}$`)
	messageIDPattern       = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	idempotencyKeyPattern  = regexp.MustCompile(`^(?:[\x21\x23-\x5B\x5D-\x7E]{1,255}|"[\x21\x23-\x5B\x5D-\x7E]{1,253}")$`)
)

// mountMessagesAPI registers the implemented /v1 endpoints on r: every route requires a valid API
// key with the operation's permission, and every request is validated against the contract here
// (the generated layer only binds parameters; decision §9 D6).
func mountMessagesAPI(r chi.Router, deps PublicAPIDeps, logger *slog.Logger) {
	r.Group(func(r chi.Router) {
		r.Use(requireAPIKey(deps.Auth, logger))
		publicapi.HandlerWithOptions(&messagesAPI{svc: deps.Messages, logger: logger}, publicapi.ChiServerOptions{
			BaseRouter:       r,
			ErrorHandlerFunc: parameterErrorHandler,
		})
	})
}

// messagesAPI implements the generated publicapi.ServerInterface.
type messagesAPI struct {
	svc    MessageService
	logger *slog.Logger
}

var _ publicapi.ServerInterface = (*messagesAPI)(nil)

// CreateMessage implements POST /v1/messages.
func (a *messagesAPI) CreateMessage(w http.ResponseWriter, r *http.Request, params publicapi.CreateMessageParams) {
	p, _ := principalFrom(r.Context())
	key, ok := normalizeIdempotencyKey(params.IdempotencyKey)
	if !ok {
		writePublicProblem(w, r, http.StatusBadRequest, "idempotency-key-invalid", "Idempotency-Key header is invalid",
			"The Idempotency-Key must be 1 to 255 visible ASCII characters, optionally in double quotes.", nil)
		return
	}
	if mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mediaType != "application/json" {
		writePublicProblem(w, r, http.StatusUnsupportedMediaType, "unsupported-media-type", "Unsupported media type",
			"Send the request body as application/json.", nil)
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxMessageRequestBytes))
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		writePublicProblem(w, r, http.StatusRequestEntityTooLarge, "payload-too-large", "Request body too large",
			"The request body exceeds the allowed size.", nil)
		return
	case err != nil:
		writePublicProblem(w, r, http.StatusBadRequest, problemMalformedRequest,
			"The request body is not valid JSON", "The request body could not be read.", nil)
		return
	}
	req, fieldErrors, malformed := parseMessageRequest(raw)
	if malformed {
		writePublicProblem(w, r, http.StatusBadRequest, problemMalformedRequest,
			"The request body is not valid JSON", "The request body could not be parsed as JSON.", nil)
		return
	}
	if len(fieldErrors) > 0 {
		writePublicProblem(w, r, http.StatusBadRequest, problemValidationError, "The request is invalid",
			"One or more fields are invalid.", fieldErrors)
		return
	}
	req.IdempotencyKey = key

	res, err := a.svc.Accept(r.Context(), p.TenantID, req)
	switch {
	case errors.Is(err, messaging.ErrIdempotencyKeyReused):
		writePublicProblem(w, r, http.StatusUnprocessableEntity, "idempotency-key-reused", "Idempotency-Key reused with a different request",
			"This Idempotency-Key was already used for a different request body.", nil)
	case errors.Is(err, messaging.ErrIdempotencyInProgress):
		w.Header().Set("Retry-After", "1")
		writePublicProblem(w, r, http.StatusConflict, "idempotency-request-in-progress", "A request with this Idempotency-Key is in progress",
			"Retry after the indicated delay with the same key and body.", nil)
	case errors.Is(err, messaging.ErrSenderIDNotApproved):
		writePublicProblem(w, r, http.StatusUnprocessableEntity, "sender-id-not-approved", "Sender ID not approved",
			"The sender_id is not an approved sender ID for this account.", nil)
	case errors.Is(err, messaging.ErrDestinationNotSupported):
		writePublicProblem(w, r, http.StatusUnprocessableEntity, "destination-not-supported", "Destination not supported",
			"The platform does not serve this destination. Only Philippine mobile numbers are supported.", nil)
	case errors.Is(err, messaging.ErrMessageTooLong):
		writePublicProblem(w, r, http.StatusUnprocessableEntity, "message-too-long", "Message too long",
			"The message needs more segments than are allowed.", nil)
	case err != nil:
		a.serverError(w, r, "accept message", err)
	default:
		if res.Replayed {
			w.Header().Set("Idempotent-Replayed", "true")
		}
		w.Header().Set("Location", "/v1/messages/"+res.Message.ID.String())
		writeJSON(w, http.StatusAccepted, toPublicMessage(res.Message))
	}
}

// GetMessage implements GET /v1/messages/{message_id}.
func (a *messagesAPI) GetMessage(w http.ResponseWriter, r *http.Request, messageID publicapi.MessageId, _ publicapi.GetMessageParams) {
	p, _ := principalFrom(r.Context())
	if !messageIDPattern.MatchString(messageID) {
		parameterProblem(w, r, "message_id", "invalid_format", "Must be a message id returned at acceptance.")
		return
	}
	id, err := messaging.ParseMessageID(messageID)
	if err != nil {
		notFound(w, r) // well-formed but not one of ours: indistinguishable from another account's
		return
	}
	m, err := a.svc.Get(r.Context(), p.TenantID, id)
	switch {
	case errors.Is(err, messaging.ErrNotFound):
		notFound(w, r)
	case err != nil:
		a.serverError(w, r, "get message", err)
	default:
		writeJSON(w, http.StatusOK, toPublicMessage(m))
	}
}

// ListMessages implements GET /v1/messages.
func (a *messagesAPI) ListMessages(w http.ResponseWriter, r *http.Request, params publicapi.ListMessagesParams) {
	p, _ := principalFrom(r.Context())
	f := messaging.ListFilter{Limit: 50, AcceptedAfter: params.AcceptedAfter, AcceptedBefore: params.AcceptedBefore}
	var errs []publicapi.FieldError
	if params.Limit != nil {
		if *params.Limit < 1 || *params.Limit > 100 {
			errs = append(errs, paramError("limit", "out_of_range", "Must be from 1 to 100."))
		}
		f.Limit = *params.Limit
	}
	if params.Status != nil {
		if !params.Status.Valid() {
			errs = append(errs, paramError("status", "invalid_value", "Must be one of the documented message statuses."))
		}
		s := messaging.Status(*params.Status)
		f.Status = &s
	}
	if params.ClientReference != nil {
		if !clientReferencePattern.MatchString(*params.ClientReference) {
			errs = append(errs, paramError("client_reference", "invalid_format", "Must be 1 to 64 characters from A-Z, a-z, 0-9, '.', '_', ':' and '-'."))
		}
		f.ClientReference = params.ClientReference
	}
	if params.Cursor != nil {
		if n := len(*params.Cursor); n < 1 || n > 512 {
			errs = append(errs, paramError("cursor", "invalid_cursor", "The cursor is invalid or expired."))
		}
		f.Cursor = *params.Cursor
	}
	if len(errs) > 0 {
		writePublicProblem(w, r, http.StatusBadRequest, problemValidationError, "The request is invalid", "One or more parameters are invalid.", errs)
		return
	}
	messages, next, err := a.svc.List(r.Context(), p.TenantID, f)
	switch {
	case errors.Is(err, messaging.ErrInvalidCursor):
		parameterProblem(w, r, "cursor", "invalid_cursor", "The cursor is invalid or expired.")
	case err != nil:
		a.serverError(w, r, "list messages", err)
	default:
		page := publicapi.MessageList{Data: make([]publicapi.Message, len(messages))}
		for i, m := range messages {
			page.Data[i] = toPublicMessage(m)
		}
		if next != "" {
			page.NextCursor = &next
		}
		writeJSON(w, http.StatusOK, page)
	}
}

func (a *messagesAPI) serverError(w http.ResponseWriter, r *http.Request, op string, err error) {
	a.logger.ErrorContext(r.Context(), op+" failed",
		slog.String("request_id", middleware.RequestIDFromContext(r.Context())), slog.String("error", err.Error()))
	internalError(w, r)
}

func toPublicMessage(m messaging.Message) publicapi.Message {
	status := m.State.Status()
	out := publicapi.Message{
		Id:              m.ID.String(),
		ClientReference: m.ClientReference,
		To:              m.Destination,
		SenderId:        m.SenderID,
		Status:          publicapi.MessageStatus(status),
		StatusCategory:  publicapi.StatusCategory(status.Category()),
		Encoding:        publicapi.MessageEncoding(m.Encoding),
		SegmentCount:    m.SegmentCount,
		AcceptedAt:      m.AcceptedAt.UTC(),
		StatusUpdatedAt: m.StatusUpdatedAt.UTC(),
	}
	if m.Reason != nil {
		out.StatusReason = &publicapi.StatusReason{Code: string(*m.Reason), Description: m.Reason.Description()}
	}
	return out
}

// normalizeIdempotencyKey validates the header against the contract pattern and removes the
// quotes of the structured-field string form, so both forms name the same key.
func normalizeIdempotencyKey(v string) (string, bool) {
	if !idempotencyKeyPattern.MatchString(v) {
		return "", false
	}
	if strings.HasPrefix(v, `"`) {
		v = v[1 : len(v)-1]
	}
	return v, true
}

// parseMessageRequest validates a MessageCreateRequest body against the contract: a JSON object
// with exactly the declared properties (additionalProperties: false), the required ones present,
// string values only (no null), and each value's pattern and length. malformed means the body is
// not valid JSON.
func parseMessageRequest(raw []byte) (req messaging.AcceptRequest, errs []publicapi.FieldError, malformed bool) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	var top any
	if err := dec.Decode(&top); err != nil {
		return req, nil, true
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return req, nil, true // trailing data after the JSON value
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return req, []publicapi.FieldError{bodyError("", "invalid_type", "The body must be a JSON object.")}, false
	}

	str := func(name string, required bool) (string, bool) {
		v, present := fields[name]
		if !present {
			if required {
				errs = append(errs, bodyError("/"+name, "required", "This field is required."))
			}
			return "", false
		}
		var s string
		if bytes.Equal(bytes.TrimSpace(v), []byte("null")) || json.Unmarshal(v, &s) != nil {
			errs = append(errs, bodyError("/"+name, "invalid_type", "Must be a string."))
			return "", false
		}
		return s, true
	}

	if to, ok := str("to", true); ok {
		if !phoneNumberPattern.MatchString(to) {
			errs = append(errs, bodyError("/to", "invalid_phone_number", "Must be an E.164 phone number, for example +639000000001."))
		}
		req.Destination = to
	}
	if sender, ok := str("sender_id", true); ok {
		switch n := utf8.RuneCountInString(sender); {
		case n < 1:
			errs = append(errs, bodyError("/sender_id", "too_short", "Must not be empty."))
		case n > 32:
			errs = append(errs, bodyError("/sender_id", "too_long", "Must be at most 32 characters."))
		}
		req.SenderID = sender
	}
	if body, ok := str("body", true); ok {
		switch n := utf8.RuneCountInString(body); {
		case n < 1:
			errs = append(errs, bodyError("/body", "too_short", "Must not be empty."))
		case n > messaging.MaxBodyCharacters:
			errs = append(errs, bodyError("/body", "too_long", "Must be at most 1600 characters."))
		}
		req.Body = body
	}
	if ref, ok := str("client_reference", false); ok {
		if !clientReferencePattern.MatchString(ref) {
			errs = append(errs, bodyError("/client_reference", "invalid_format", "Must be 1 to 64 characters from A-Z, a-z, 0-9, '.', '_', ':' and '-'."))
		}
		req.ClientReference = &ref
	}
	unknown := make([]string, 0)
	for name := range fields {
		switch name {
		case "to", "sender_id", "body", "client_reference":
		default:
			unknown = append(unknown, name)
		}
	}
	slices.Sort(unknown)
	for _, name := range unknown {
		errs = append(errs, bodyError("/"+jsonPointerEscape(name), "unknown_field", "This field is not allowed."))
	}
	return req, errs, false
}

func bodyError(pointer, code, detail string) publicapi.FieldError {
	return publicapi.FieldError{Pointer: &pointer, Code: code, Detail: detail}
}

func paramError(name, code, detail string) publicapi.FieldError {
	return publicapi.FieldError{Parameter: &name, Code: code, Detail: detail}
}

func parameterProblem(w http.ResponseWriter, r *http.Request, name, code, detail string) {
	writePublicProblem(w, r, http.StatusBadRequest, problemValidationError, "The request is invalid",
		"One or more parameters are invalid.", []publicapi.FieldError{paramError(name, code, detail)})
}

func jsonPointerEscape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1")
}
