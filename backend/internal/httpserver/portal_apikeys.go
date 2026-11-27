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

// maxAPIKeyRequestBytes bounds the create request (a name of at most 100 characters and two scopes).
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

// CreateApiKey implements POST /portal/v1/api-keys with the same key service as smsctl. The full key
// is in this response only; it is never stored or logged.
func (a *portalAPI) CreateApiKey(w http.ResponseWriter, r *http.Request, _ portalapi.CreateApiKeyParams) {
	p, ok := a.require(w, r, portal.PermissionAPIKeysManage)
	if !ok {
		return
	}
	raw, ok := readJSONBody(w, r, maxAPIKeyRequestBytes)
	if !ok {
		return
	}
	name, scopes, errs, malformed := parseAPIKeyCreateRequest(raw)
	if malformed {
		malformedJSON(w, r)
		return
	}
	if len(errs) > 0 {
		writePortalProblem(w, r, http.StatusBadRequest, problemValidationError, "The request is invalid", "One or more fields are invalid.", errs)
		return
	}
	key, token, err := a.keys.Create(r.Context(), p.TenantID, name, scopes)
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

// parseAPIKeyCreateRequest validates {"name": string, "scopes": [scope, ...]} (additionalProperties:
// false). malformed reports a body that is not JSON.
func parseAPIKeyCreateRequest(raw []byte) (name string, scopes []apikey.Scope, errs []portalapi.FieldError, malformed bool) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	var top any
	if err := dec.Decode(&top); err != nil {
		return "", nil, nil, true
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return "", nil, nil, true
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return "", nil, []portalapi.FieldError{portalBodyError("", "invalid_type", "The body must be a JSON object.")}, false
	}
	for field := range fields {
		if field != "name" && field != "scopes" {
			errs = append(errs, portalBodyError("/"+jsonPointerEscape(field), "unknown_field", "This field is not allowed."))
		}
	}

	switch v, ok := fields["name"]; {
	case !ok:
		errs = append(errs, portalBodyError("/name", "required", "This field is required."))
	case bytes.Equal(bytes.TrimSpace(v), []byte("null")) || json.Unmarshal(v, &name) != nil:
		errs = append(errs, portalBodyError("/name", "invalid_type", "Must be a string."))
	default:
		normalized, err := apikey.NormalizeLabel(name)
		if err != nil {
			errs = append(errs, invalidAPIKeyName())
		}
		name = normalized
	}

	var list []string
	switch v, ok := fields["scopes"]; {
	case !ok:
		errs = append(errs, portalBodyError("/scopes", "required", "This field is required."))
	case bytes.Equal(bytes.TrimSpace(v), []byte("null")) || json.Unmarshal(v, &list) != nil:
		errs = append(errs, portalBodyError("/scopes", "invalid_type", "Must be an array of strings."))
	case len(list) == 0:
		errs = append(errs, portalBodyError("/scopes", "required", "Choose at least one permission."))
	default:
		for _, s := range list {
			sc := apikey.Scope(s)
			switch {
			case !sc.Valid():
				errs = append(errs, portalBodyError("/scopes", "invalid_value", "Choose messages:send, messages:read or both."))
				return name, nil, errs, false
			case slices.Contains(scopes, sc):
				errs = append(errs, portalBodyError("/scopes", "duplicate", "Each permission may be listed once."))
				return name, nil, errs, false
			}
			scopes = append(scopes, sc)
		}
	}
	return name, scopes, errs, false
}
