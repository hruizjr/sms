package httpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"slices"

	"github.com/hruizjr/sms/backend/internal/apikey"
	"github.com/hruizjr/sms/backend/internal/middleware"
	"github.com/hruizjr/sms/backend/internal/portal"
	"github.com/hruizjr/sms/backend/internal/portalapi"
)

// maxAPIKeyRequestBytes bounds the create request (a name of at most 100 characters, two scopes and
// a password of at most 1 KiB).
const maxAPIKeyRequestBytes = 4 << 10

// ListApiKeys implements GET /portal/v1/api-keys.
func (a *portalAPI) ListApiKeys(w http.ResponseWriter, r *http.Request, _ portalapi.ListApiKeysParams) {
	p, ok := a.require(w, r, portal.PermissionAPIKeysManage)
	if !ok {
		return
	}
	keys, err := a.keys.List(r.Context(), p.TenantID)
	if err != nil {
		a.serverError(w, r, "list api keys", err)
		return
	}
	list := portalapi.ApiKeyList{Data: make([]portalapi.ApiKey, len(keys))}
	for i, k := range keys {
		list.Data[i] = toPortalAPIKey(k)
	}
	writeJSON(w, http.StatusOK, list)
}

// CreateApiKey implements POST /portal/v1/api-keys with the same key service as smsctl. The owner
// re-enters their current password first (owner decision G4-7): without a successful
// re-authentication no key is created and nothing secret is returned. The full key is in the success
// response only; neither it nor the password is ever stored or logged.
func (a *portalAPI) CreateApiKey(w http.ResponseWriter, r *http.Request, _ portalapi.CreateApiKeyParams) {
	p, ok := a.require(w, r, portal.PermissionAPIKeysManage)
	if !ok {
		return
	}
	raw, ok := readJSONBody(w, r, maxAPIKeyRequestBytes)
	if !ok {
		return
	}
	req, errs, malformed := parseAPIKeyCreateRequest(raw)
	if malformed {
		malformedJSON(w, r)
		return
	}
	if len(errs) > 0 {
		writePortalProblem(w, r, http.StatusBadRequest, problemValidationError, "The request is invalid", "One or more fields are invalid.", errs)
		return
	}
	switch err := a.auth.Reauthenticate(r.Context(), p, req.currentPassword); {
	case errors.Is(err, portal.ErrReauthenticationFailed):
		a.logger.InfoContext(r.Context(), "api key creation refused: re-authentication failed",
			slog.String("request_id", middleware.RequestIDFromContext(r.Context())),
			slog.String("tenant_id", p.TenantID.String()), slog.String("user_id", userIDString(p.UserID)))
		writePortalProblem(w, r, http.StatusForbidden, "reauthentication-failed", "Password confirmation failed",
			"The current password is incorrect, or the account is temporarily locked after too many attempts.", nil)
		return
	case err != nil:
		a.serverError(w, r, "api key re-authentication", err)
		return
	}
	key, token, err := a.keys.Create(r.Context(), p.TenantID, req.name, req.scopes)
	switch {
	case errors.Is(err, apikey.ErrInvalidLabel):
		writePortalProblem(w, r, http.StatusBadRequest, problemValidationError, "The request is invalid", "One or more fields are invalid.",
			[]portalapi.FieldError{invalidAPIKeyName()})
		return
	case errors.Is(err, apikey.ErrInvalidScopes):
		writePortalProblem(w, r, http.StatusBadRequest, problemValidationError, "The request is invalid", "One or more fields are invalid.",
			[]portalapi.FieldError{portalBodyError("/scopes", "invalid_value", "Choose messages:send, messages:read or both.")})
		return
	case err != nil:
		a.serverError(w, r, "create api key", err)
		return
	}
	a.logger.InfoContext(r.Context(), "api key created",
		slog.String("request_id", middleware.RequestIDFromContext(r.Context())),
		slog.String("tenant_id", p.TenantID.String()), slog.String("user_id", userIDString(p.UserID)),
		slog.String("api_key_id", key.IDString()))
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, portalapi.ApiKeyCreated{ApiKey: toPortalAPIKey(key), Key: token})
}

// RevokeApiKey implements DELETE /portal/v1/api-keys/{api_key_id}. Revoking a key that is already
// revoked returns it unchanged; a key of another tenant is not found (row-level security hides it).
func (a *portalAPI) RevokeApiKey(w http.ResponseWriter, r *http.Request, apiKeyID portalapi.ApiKeyId, _ portalapi.RevokeApiKeyParams) {
	p, ok := a.require(w, r, portal.PermissionAPIKeysManage)
	if !ok {
		return
	}
	id, err := apikey.ParseKeyID(apiKeyID)
	if err != nil {
		notFound(w, r)
		return
	}
	key, err := a.keys.Revoke(r.Context(), p.TenantID, id)
	if errors.Is(err, apikey.ErrNotFound) {
		// Not found, or already revoked: answer with the revoked key if it exists.
		keys, listErr := a.keys.List(r.Context(), p.TenantID)
		if listErr != nil {
			a.serverError(w, r, "revoke api key", listErr)
			return
		}
		i := slices.IndexFunc(keys, func(k apikey.Key) bool { return k.ID == id && k.RevokedAt != nil })
		if i < 0 {
			notFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, toPortalAPIKey(keys[i]))
		return
	}
	if err != nil {
		a.serverError(w, r, "revoke api key", err)
		return
	}
	a.logger.InfoContext(r.Context(), "api key revoked",
		slog.String("request_id", middleware.RequestIDFromContext(r.Context())),
		slog.String("tenant_id", p.TenantID.String()), slog.String("user_id", userIDString(p.UserID)),
		slog.String("api_key_id", key.IDString()))
	writeJSON(w, http.StatusOK, toPortalAPIKey(key))
}

func toPortalAPIKey(k apikey.Key) portalapi.ApiKey {
	out := portalapi.ApiKey{
		Id:        k.IDString(),
		Name:      k.Label,
		Prefix:    k.Prefix(),
		Scopes:    make([]portalapi.ApiKeyScope, len(k.Scopes)),
		Status:    "active",
		CreatedAt: k.CreatedAt.UTC(),
	}
	for i, sc := range k.Scopes {
		out.Scopes[i] = portalapi.ApiKeyScope(sc)
	}
	if k.RevokedAt != nil {
		revoked := k.RevokedAt.UTC()
		out.Status, out.RevokedAt = "revoked", &revoked
	}
	return out
}

func invalidAPIKeyName() portalapi.FieldError {
	return portalBodyError("/name", "invalid_value", "Must be 1 to 100 characters, without control characters.")
}

// apiKeyCreateRequest is a validated POST /portal/v1/api-keys body.
type apiKeyCreateRequest struct {
	name            string
	scopes          []apikey.Scope
	currentPassword string
}

// parseAPIKeyCreateRequest validates {"name": string, "scopes": [scope, ...], "current_password":
// string} (additionalProperties: false). malformed reports a body that is not JSON. Error details
// never echo the password.
func parseAPIKeyCreateRequest(raw []byte) (req apiKeyCreateRequest, errs []portalapi.FieldError, malformed bool) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	var top any
	if err := dec.Decode(&top); err != nil {
		return req, nil, true
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return req, nil, true
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return req, []portalapi.FieldError{portalBodyError("", "invalid_type", "The body must be a JSON object.")}, false
	}
	for field := range fields {
		if field != "name" && field != "scopes" && field != "current_password" {
			errs = append(errs, portalBodyError("/"+jsonPointerEscape(field), "unknown_field", "This field is not allowed."))
		}
	}
	isNull := func(v json.RawMessage) bool { return bytes.Equal(bytes.TrimSpace(v), []byte("null")) }

	switch v, ok := fields["name"]; {
	case !ok:
		errs = append(errs, portalBodyError("/name", "required", "This field is required."))
	case isNull(v) || json.Unmarshal(v, &req.name) != nil:
		errs = append(errs, portalBodyError("/name", "invalid_type", "Must be a string."))
	default:
		normalized, err := apikey.NormalizeLabel(req.name)
		if err != nil {
			errs = append(errs, invalidAPIKeyName())
		}
		req.name = normalized
	}

	switch v, ok := fields["current_password"]; {
	case !ok:
		errs = append(errs, portalBodyError("/current_password", "required", "Enter your current password to confirm."))
	case isNull(v) || json.Unmarshal(v, &req.currentPassword) != nil:
		errs = append(errs, portalBodyError("/current_password", "invalid_type", "Must be a string."))
	case req.currentPassword == "":
		errs = append(errs, portalBodyError("/current_password", "required", "Enter your current password to confirm."))
	case len(req.currentPassword) > portal.MaxPasswordBytes:
		errs = append(errs, portalBodyError("/current_password", "too_long", "Must be at most 1024 bytes."))
	}

	var list []string
	switch v, ok := fields["scopes"]; {
	case !ok:
		errs = append(errs, portalBodyError("/scopes", "required", "This field is required."))
	case isNull(v) || json.Unmarshal(v, &list) != nil:
		errs = append(errs, portalBodyError("/scopes", "invalid_type", "Must be an array of strings."))
	case len(list) == 0:
		errs = append(errs, portalBodyError("/scopes", "required", "Choose at least one permission."))
	default:
		for _, s := range list {
			sc := apikey.Scope(s)
			switch {
			case !sc.Valid():
				return req, append(errs, portalBodyError("/scopes", "invalid_value", "Choose messages:send, messages:read or both.")), false
			case slices.Contains(req.scopes, sc):
				return req, append(errs, portalBodyError("/scopes", "duplicate", "Each permission may be listed once.")), false
			}
			req.scopes = append(req.scopes, sc)
		}
	}
	return req, errs, false
}
