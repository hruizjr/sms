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
	user := [16]byte{1}
	valid := Event{Action: ActionSignIn, Outcome: OutcomeSuccess, Actor: PortalUser(user),
		Target: &Target{Type: TargetPortalSession, ID: user}}
	if err := valid.validate(); err != nil {
		t.Fatalf("valid event: %v", err)
	}
	lock := Event{Action: ActionAccountLocked, Outcome: OutcomeEnforced, Actor: System, Target: &Target{Type: TargetPortalUser, ID: user}}
	if err := lock.validate(); err != nil {
		t.Fatalf("account lock (enforced): %v", err)
	}
	deduplicated := Event{Action: ActionAPIKeyList, Outcome: OutcomeDenied, Actor: PortalUser(user),
		DedupKey: DeniedKey(PortalUser(user), ActionAPIKeyList, time.Now())}
	if err := deduplicated.validate(); err != nil {
		t.Fatalf("deduplicated refusal: %v", err)
	}
	invalid := map[string]Event{
		"platform action":        {Action: ActionSignInRateLimited, Outcome: OutcomeDenied, Actor: Anonymous},
		"unknown action":         {Action: "portal.password_reset", Outcome: OutcomeSuccess, Actor: Anonymous},
		"unknown outcome":        {Action: ActionSignIn, Outcome: "maybe", Actor: Anonymous},
		"success for a lock":     {Action: ActionAccountLocked, Outcome: OutcomeSuccess, Actor: System, Target: &Target{Type: TargetPortalUser, ID: user}},
		"enforced for a sign-in": {Action: ActionSignIn, Outcome: OutcomeEnforced, Actor: Anonymous},
		"bad dedup key":          {Action: ActionAPIKeyList, Outcome: OutcomeDenied, Actor: PortalUser(user), DedupKey: "Denied; DROP"},
		"long dedup key":         {Action: ActionAPIKeyList, Outcome: OutcomeDenied, Actor: PortalUser(user), DedupKey: strings.Repeat("a", 161)},
		"user without ID":        {Action: ActionSignIn, Outcome: OutcomeSuccess, Actor: Actor{Type: ActorPortalUser}},
		"anonymous with ID":      {Action: ActionSignIn, Outcome: OutcomeFailure, Actor: Actor{Type: ActorAnonymous, ID: user}},
		"unknown actor":          {Action: ActionSignIn, Outcome: OutcomeFailure, Actor: Actor{Type: "api_key", ID: user}},
		"target without ID":      {Action: ActionSignIn, Outcome: OutcomeFailure, Actor: Anonymous, Target: &Target{Type: TargetPortalUser}},
		"unknown target type":    {Action: ActionSignIn, Outcome: OutcomeFailure, Actor: Anonymous, Target: &Target{Type: "message", ID: user}},
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
		RequiredPermission: "api_keys:manage", LockedUntil: &until, Via: "smsctl", DedupIntervalSeconds: 300,
		Limit: 5, WindowSeconds: 60, RatePerSecond: 10, SourcePrefixLength: 48,
	})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(b, &fields); err != nil {
		t.Fatal(err)
	}
	want := []string{"api_key_name", "api_key_scopes", "dedup_interval_seconds", "limit", "locked_until", "rate_per_second", "reason",
		"required_permission", "role", "source_prefix_length", "via", "window_seconds"}
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

func TestDedupKeys(t *testing.T) {
	t.Parallel()
	user := PortalUser([16]byte{0xab, 1})
	base := time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)
	k := DeniedKey(user, ActionAPIKeyList, base.Add(10*time.Second))
	if !dedupKeyAllowed(k) {
		t.Fatalf("DeniedKey %q has an invalid form", k)
	}
	if k != DeniedKey(user, ActionAPIKeyList, base.Add(4*time.Minute+59*time.Second)) {
		t.Error("two refusals in the same five-minute interval have different keys")
	}
	for name, other := range map[string]string{
		"next interval": DeniedKey(user, ActionAPIKeyList, base.Add(5*time.Minute)),
		"other action":  DeniedKey(user, ActionAuditLogList, base.Add(10*time.Second)),
		"other actor":   DeniedKey(PortalUser([16]byte{0xab, 2}), ActionAPIKeyList, base.Add(10*time.Second)),
	} {
		if other == k {
			t.Errorf("%s: same key %q", name, k)
		}
	}
	manila := time.FixedZone("PHT", 8*3600)
	if DeniedKey(user, ActionAPIKeyList, base.Add(10*time.Second).In(manila)) != k {
		t.Error("the key depends on the time zone")
	}
	until := base.Add(15 * time.Minute)
	lk := LockedAttemptKey(ActionSignIn, [16]byte{1}, until)
	if !dedupKeyAllowed(lk) || lk == LockedAttemptKey(ActionSignIn, [16]byte{1}, until.Add(time.Microsecond)) ||
		lk == LockedAttemptKey(ActionAPIKeyCreate, [16]byte{1}, until) || lk != LockedAttemptKey(ActionSignIn, [16]byte{1}, until.In(manila)) {
		t.Errorf("LockedAttemptKey %q: not one key per account, action and lock", lk)
	}
}
