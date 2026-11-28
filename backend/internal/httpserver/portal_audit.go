package httpserver

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hruizjr/sms/backend/internal/apikey"
	"github.com/hruizjr/sms/backend/internal/audit"
	"github.com/hruizjr/sms/backend/internal/portal"
	"github.com/hruizjr/sms/backend/internal/portalapi"
)

// ListAuditEvents implements GET /portal/v1/audit-events: the organization's security audit log,
// read-only, for owners (audit_log:read). The tenant comes from the session; row-level security limits
// the query to it as well.
func (a *portalAPI) ListAuditEvents(w http.ResponseWriter, r *http.Request, params portalapi.ListAuditEventsParams) {
	p, ok := a.requireAudited(w, r, portal.PermissionAuditLogRead, audit.ActionAuditLogList)
	if !ok {
		return
	}
	f := audit.Filter{Limit: 25, OccurredFrom: params.OccurredFrom, OccurredBefore: params.OccurredBefore}
	var errs []portalapi.FieldError
	if params.Limit != nil {
		if *params.Limit < 1 || *params.Limit > 100 {
			errs = append(errs, portalParamError("limit", "out_of_range", "Must be from 1 to 100."))
		}
		f.Limit = *params.Limit
	}
	if params.Action != nil {
		action := audit.Action(*params.Action)
		if !params.Action.Valid() {
			errs = append(errs, portalParamError("action", "invalid_value", "Must be one of the documented audit actions."))
		}
		f.Action = &action
	}
	if params.Outcome != nil {
		outcome := audit.Outcome(*params.Outcome)
		if !outcome.Valid() {
			errs = append(errs, portalParamError("outcome", "invalid_value", "Must be success, failure or denied."))
		}
		f.Outcome = &outcome
	}
	if f.OccurredFrom != nil && f.OccurredBefore != nil && !f.OccurredBefore.After(*f.OccurredFrom) {
		errs = append(errs, portalParamError("occurred_before", "invalid_range", "Must be later than occurred_from."))
	}
	if params.Cursor != nil {
		if n := len(*params.Cursor); n < 1 || n > 512 {
			errs = append(errs, portalParamError("cursor", "invalid_cursor", "The cursor is invalid or expired."))
		}
		f.Cursor = *params.Cursor
	}
	if len(errs) > 0 {
		writePortalProblem(w, r, http.StatusBadRequest, problemValidationError, "The request is invalid", "One or more parameters are invalid.", errs)
		return
	}
	entries, next, err := a.audit.List(r.Context(), p.TenantID, f)
	switch {
	case errors.Is(err, audit.ErrInvalidCursor):
		writePortalProblem(w, r, http.StatusBadRequest, problemValidationError, "The request is invalid", "One or more parameters are invalid.",
			[]portalapi.FieldError{portalParamError("cursor", "invalid_cursor", "The cursor is invalid or expired.")})
	case err != nil:
		a.serverError(w, r, "list audit events", err)
	default:
		page := portalapi.AuditEventList{Data: make([]portalapi.AuditEvent, len(entries))}
		for i, e := range entries {
			page.Data[i] = toPortalAuditEvent(e)
		}
		if next != "" {
			page.NextCursor = &next
		}
		writeJSON(w, http.StatusOK, page)
	}
}

// toPortalAuditEvent maps an audit entry to the contract. Only the documented detail fields are
// copied; platform-only fields never appear in a tenant's events.
func toPortalAuditEvent(e audit.Entry) portalapi.AuditEvent {
	out := portalapi.AuditEvent{
		Id:         e.ID,
		OccurredAt: e.OccurredAt.UTC(),
		Action:     portalapi.AuditAction(e.Action),
		Outcome:    portalapi.AuditOutcome(e.Outcome),
		Actor:      toPortalAuditParty(e.Actor),
		RequestId:  optionalString(e.RequestID),
		UserAgent:  optionalString(e.UserAgent),
	}
	if e.SourceIP.IsValid() {
		ip := e.SourceIP.String()
		out.SourceIp = &ip
	}
	if e.Target != nil {
		t := toPortalAuditParty(*e.Target)
		out.Target = &t
	}
	d := e.Details
	out.Details = portalapi.AuditDetails{ApiKeyName: optionalString(d.APIKeyName), Via: optionalString(d.Via)}
	if d.Reason != "" {
		reason := string(d.Reason)
		out.Details.Reason = &reason
	}
	if len(d.APIKeyScopes) > 0 {
		scopes := make([]portalapi.ApiKeyScope, len(d.APIKeyScopes))
		for i, sc := range d.APIKeyScopes {
			scopes[i] = portalapi.ApiKeyScope(sc)
		}
		out.Details.ApiKeyScopes = &scopes
	}
	if d.Role != "" {
		role := portalapi.Role(d.Role)
		out.Details.Role = &role
	}
	if d.RequiredPermission != "" {
		perm := portalapi.Permission(d.RequiredPermission)
		out.Details.RequiredPermission = &perm
	}
	if d.LockedUntil != nil {
		until := d.LockedUntil.UTC()
		out.Details.LockedUntil = &until
	}
	return out
}

func toPortalAuditParty(p audit.Party) portalapi.AuditParty {
	out := portalapi.AuditParty{Type: portalapi.AuditPartyType(p.Type), Email: optionalString(p.Email)}
	if p.ID != ([16]byte{}) {
		var id string
		if p.Type == string(audit.TargetAPIKey) {
			id = apikey.Key{ID: p.ID}.IDString() // the form the API key list uses
		} else {
			id = pgtype.UUID{Bytes: p.ID, Valid: true}.String()
		}
		out.Id = &id
	}
	return out
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
