package audit

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNormalizeUserAgent(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"":                                 "",
		"  Mozilla/5.0  ":                  "Mozilla/5.0",
		"curl/8.0\r\nX-Injected: 1":        "curl/8.0X-Injected: 1",
		"tab\tand\x00nul":                  "tabandnul",
		"\xff\xfe invalid UTF-8":           "",
		strings.Repeat("é", 300):           strings.Repeat("é", MaxUserAgentCharacters),
		strings.Repeat("a", 255) + " bcde": strings.Repeat("a", 255),
	}
	for in, want := range tests {
		if got := NormalizeUserAgent(in); got != want {
			t.Errorf("NormalizeUserAgent(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEventValidation(t *testing.T) {
	t.Parallel()
	now := time.Now()
	user := [16]byte{1}
	valid := Event{OccurredAt: now, Action: ActionSignIn, Outcome: OutcomeSuccess, Actor: PortalUser(user),
		Target: &Target{Type: TargetPortalSession, ID: user}}
	if err := valid.validate(); err != nil {
		t.Fatalf("valid event: %v", err)
	}
	invalid := map[string]Event{
		"platform action":        {OccurredAt: now, Action: ActionSignInRateLimited, Outcome: OutcomeDenied, Actor: Anonymous},
		"unknown action":         {OccurredAt: now, Action: "portal.password_reset", Outcome: OutcomeSuccess, Actor: Anonymous},
		"unknown outcome":        {OccurredAt: now, Action: ActionSignIn, Outcome: "maybe", Actor: Anonymous},
		"no time":                {Action: ActionSignIn, Outcome: OutcomeSuccess, Actor: Anonymous},
		"user without ID":        {OccurredAt: now, Action: ActionSignIn, Outcome: OutcomeSuccess, Actor: Actor{Type: ActorPortalUser}},
		"anonymous with ID":      {OccurredAt: now, Action: ActionSignIn, Outcome: OutcomeFailure, Actor: Actor{Type: ActorAnonymous, ID: user}},
		"unknown actor":          {OccurredAt: now, Action: ActionSignIn, Outcome: OutcomeFailure, Actor: Actor{Type: "api_key", ID: user}},
		"target without ID":      {OccurredAt: now, Action: ActionSignIn, Outcome: OutcomeFailure, Actor: Anonymous, Target: &Target{Type: TargetPortalUser}},
		"unknown target type":    {OccurredAt: now, Action: ActionSignIn, Outcome: OutcomeFailure, Actor: Anonymous, Target: &Target{Type: "message", ID: user}},
		"zero-valued everything": {},
	}
	for name, ev := range invalid {
		if err := ev.validate(); err == nil {
			t.Errorf("%s: validate() succeeded", name)
		}
	}
}

// Details has a fixed set of fields, none of which can hold a credential.
func TestDetailsFields(t *testing.T) {
	t.Parallel()
	until := time.Date(2026, 11, 28, 9, 15, 0, 0, time.UTC)
	b, err := json.Marshal(Details{
		Reason: ReasonTooManyAttempts, APIKeyName: "n", APIKeyScopes: []string{"messages:send"}, Role: "viewer",
		RequiredPermission: "api_keys:manage", LockedUntil: &until, Via: "smsctl", MaxAttempts: 5, WindowSeconds: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(b, &fields); err != nil {
		t.Fatal(err)
	}
	want := []string{"api_key_name", "api_key_scopes", "locked_until", "max_attempts", "reason", "required_permission", "role", "via", "window_seconds"}
	if len(fields) != len(want) {
		t.Errorf("Details fields = %v, want %v", fields, want)
	}
	for _, name := range want {
		if _, ok := fields[name]; !ok {
			t.Errorf("Details has no %q field", name)
		}
	}
	if b, _ := json.Marshal(Details{}); string(b) != "{}" {
		t.Errorf("empty Details = %s, want {}", b)
	}
}
