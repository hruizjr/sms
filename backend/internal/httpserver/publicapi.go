package httpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/hruizjr/sms/backend/internal/middleware"
	"github.com/hruizjr/sms/backend/internal/publicapi"
)

// problemTypeBase is the base of the contract's problem type URIs (contracts/openapi/README.md §5).
// Provisional: the public domain is not decided yet (D-145), so the contract uses this placeholder.
const problemTypeBase = "https://api.example.invalid/problems/"

// Problem types this file can emit. Titles are the ones used in the contract's examples where the
// contract gives one.
const (
	problemValidationError       = "validation-error"
	problemMalformedRequest      = "malformed-request"
	problemIdempotencyKeyMissing = "idempotency-key-missing"
)

// mountPublicAPI registers the generated public API v1 routes (contracts/openapi/public-api-v1.yaml)
// on r, served by impl through the generated strict (typed request/response) handler.
//
// It is used only by the tests of the generated binding layer. The implemented endpoints are mounted
// by mountMessagesAPI (messages.go), which uses the non-strict generated interface so that it can
// validate the raw request body itself.
//
// What the generated layer does: routing, binding of path/query/header parameters to their declared
// types, and JSON body decoding. What it does NOT do (owner decision D6), and what mountMessagesAPI
// adds: authentication and authorization, and full request validation against the contract
// (patterns, lengths, enums, required body fields, additionalProperties: false, rejecting null where
// the contract does not allow it, Content-Type).
//
// The generator's default error handlers write the raw Go error text as text/plain. They are always
// replaced here, so clients receive RFC 9457 problem details with fixed, client-safe text only.
func mountPublicAPI(r chi.Router, impl publicapi.StrictServerInterface, logger *slog.Logger) {
	strict := publicapi.NewStrictHandlerWithOptions(impl, nil, publicapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, _ error) {
			// The only request error the strict layer reports is an undecodable JSON body. The
			// decoder's message is not returned: it can echo request content.
			writePublicProblem(w, r, http.StatusBadRequest, problemMalformedRequest,
				"The request body is not valid JSON", "The request body could not be parsed as JSON.", nil)
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.ErrorContext(r.Context(), "public API handler failed",
				slog.String("request_id", middleware.RequestIDFromContext(r.Context())),
				slog.String("error", err.Error()))
			internalError(w, r)
		},
	})
	publicapi.HandlerWithOptions(strict, publicapi.ChiServerOptions{
		BaseRouter:       r,
		ErrorHandlerFunc: parameterErrorHandler,
	})
}

// parameterErrorHandler turns a generated parameter-binding error into a 400 problem document.
func parameterErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	var missingHeader *publicapi.RequiredHeaderError
	if errors.As(err, &missingHeader) && missingHeader.ParamName == "Idempotency-Key" {
		writePublicProblem(w, r, http.StatusBadRequest, problemIdempotencyKeyMissing,
			"Idempotency-Key header is required",
			"Send an Idempotency-Key header that is unique for this operation.", nil)
		return
	}

	name, code, detail := "", "invalid_value", "The value could not be parsed as the declared type."
	var (
		invalidFormat *publicapi.InvalidParamFormatError
		tooMany       *publicapi.TooManyValuesForParamError
		requiredParam *publicapi.RequiredParamError
		unmarshaling  *publicapi.UnmarshalingParamError
	)
	switch {
	case errors.As(err, &invalidFormat):
		name = invalidFormat.ParamName
	case errors.As(err, &tooMany):
		name, code, detail = tooMany.ParamName, "multiple_values", "The parameter must be given at most once."
	case errors.As(err, &requiredParam):
		name, code, detail = requiredParam.ParamName, "required", "The parameter is required."
	case errors.As(err, &missingHeader):
		name, code, detail = missingHeader.ParamName, "required", "The header is required."
	case errors.As(err, &unmarshaling):
		name = unmarshaling.ParamName
	}

	var fieldErrors []publicapi.FieldError
	if name != "" {
		fieldErrors = []publicapi.FieldError{{Parameter: &name, Code: code, Detail: detail}}
	}
	writePublicProblem(w, r, http.StatusBadRequest, problemValidationError,
		"The request is invalid", "One or more parameters are invalid.", fieldErrors)
}

// writePublicProblem writes a problem document of the public API, using the generated contract
// type. title and detail must be fixed, client-safe text.
func writePublicProblem(w http.ResponseWriter, r *http.Request, status int, slug, title, detail string, fieldErrors []publicapi.FieldError) {
	body := publicapi.Problem{
		Type:   problemTypeBase + slug,
		Title:  title,
		Status: status,
		Detail: &detail,
	}
	if requestID := middleware.RequestIDFromContext(r.Context()); requestID != "" {
		body.RequestId = &requestID
	}
	if len(fieldErrors) > 0 {
		body.Errors = &fieldErrors
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body) // see writeProblem for why the error is not actionable
}
