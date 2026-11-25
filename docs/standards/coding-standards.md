# Coding Standards

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — pending owner review | Project owner | 2026-11-24 |

This document expands the rules in [`AGENTS.md`](../../AGENTS.md) §7–§10 with rationale and examples.
The MUST / MUST NOT rules stated in `AGENTS.md` are project-wide non-negotiable rules; this document is
authoritative for the language conventions and examples that elaborate them, as defined in [`AGENTS.md` §1.1](../../AGENTS.md#11-document-authority-and-conflicts).
It does not grant permissions, widen task scope, or override restrictions in `AGENTS.md`. If it genuinely conflicts with another document, stop the affected work, report the conflict, and request owner resolution ([`AGENTS.md` §1.1](../../AGENTS.md#11-document-authority-and-conflicts)).

Examples are illustrative only — they are not application code and do not define real APIs.

---

## General (all languages)

- **Names describe business meaning.** `reserveCredits`, not `doUpdate2`. Use the glossary's terms
  (tenant, segment, delivery receipt, sender ID) consistently in code, APIs, and UI.
- **Small units.** Functions do one thing. Files stay focused. If a function needs a comment to explain
  *what* it does, consider renaming or splitting it; use comments to explain *why*.
- **No dead code, no commented-out code.** Git history keeps old versions.
- **TODOs carry context:** `TODO(<task-id or telco-confirm>): <what and why>`. Untracked TODOs are not allowed.
- **Formatting is automated** (gofmt, Prettier). Do not hand-format or argue about style the tools decide.
- **No magic numbers.** Name constants; make operational limits configurable.

---

## Go

### Package layout

```
backend/
├── cmd/api/main.go                 // thin: config → wiring → run → shutdown
└── internal/
    ├── config/  database/  migration/  httpserver/  middleware/  observability/
    │                                // technical plumbing: one package per concern, no business rules
    └── billing/                     // a business module (future)
        ├── billing.go               // exported service API + domain types
        ├── ledger.go                // domain logic
        ├── repository.go            // persistence (SQL), unexported where possible
        ├── http.go                  // HTTP handlers for this module (transport)
        └── billing_test.go
```

Technical packages sit directly under `internal/` next to business modules; there is no
`internal/platform/` umbrella (owner decision, Phase 3 Group 2). As a module grows, sub-packages are
acceptable (`billing/ledger`), but avoid deep nesting.

### Interfaces at the consumer

```go
// In package messaging (the consumer), not in package billing.
type creditReserver interface {
	Reserve(ctx context.Context, tenantID TenantID, messageID MessageID, amount Credits) error
}
```

Small interfaces defined where they are used make testing easy and avoid import cycles.

### Error handling

```go
// Sentinel/typed errors for expected conditions.
var ErrInsufficientCredit = errors.New("insufficient credit")

func (s *Service) Reserve(ctx context.Context, ...) error {
	if err := s.repo.reserve(ctx, ...); err != nil {
		return fmt.Errorf("reserve credit for message %s: %w", messageID, err)
	}
	return nil
}

// At the HTTP boundary only:
switch {
case errors.Is(err, billing.ErrInsufficientCredit):
	writeProblem(w, http.StatusPaymentRequired, "insufficient-credit", ...)
default:
	logger.ErrorContext(ctx, "reserve failed", "error", err) // log once, here
	writeProblem(w, http.StatusInternalServerError, "internal-error", ...)
}
```

- Error strings: lowercase, no trailing punctuation, describe the failed operation.
- Never compare error strings; use `errors.Is`/`errors.As`.
- Do not expose internal errors to API clients.

### Context

- First parameter, named `ctx`.
- Set deadlines on outbound calls: `ctx, cancel := context.WithTimeout(ctx, cfg.DBTimeout); defer cancel()`.
- When work must continue after the request ends (e.g. finishing an audit write), use
  `context.WithoutCancel(ctx)` deliberately with its own timeout, and comment why.

### Goroutine lifecycle (pattern)

```go
// Run blocks until ctx is cancelled and all workers have exited.
func (d *Dispatcher) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx) // golang.org/x/sync — dependency to be approved
	for i := 0; i < d.cfg.Workers; i++ { // bounded
		g.Go(func() error { return d.worker(ctx) })
	}
	return g.Wait()
}

func (d *Dispatcher) worker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case job, ok := <-d.jobs:
			if !ok {
				return nil
			}
			d.handle(ctx, job) // handle recovers panics and records failures
		}
	}
}
```

Rules: bounded worker count from config; every loop observes `ctx.Done()`; the owner waits for exit;
panics inside workers are recovered and converted to errors/metrics.

### Graceful shutdown order

1. Receive SIGTERM → stop accepting new requests/work (close listeners, stop consuming).
2. Allow in-flight work to complete within a configured deadline.
3. Work not completed must remain safely re-processable (never acknowledged before it is done).
4. Close resources in reverse order: providers/sessions → broker clients → database pool → logger flush.

### Configuration

```go
type Config struct {
	HTTPAddr     string        `env:"HTTP_ADDR"`      // illustrative tags; loader library TBD
	DatabaseURL  string        `env:"DATABASE_URL"`   // secret: no default
	DBTimeout    time.Duration `env:"DB_TIMEOUT"`
}
func (c Config) Validate() error { /* fail fast with clear messages */ }
```

Whether to use a small config library or `os.Getenv` with hand-written parsing is a scaffolding decision.

### Database access

- Repository methods take `ctx` and, for tenant data, an explicit `tenantID`.
- Transaction helper:

```go
func (db *DB) InTx(ctx context.Context, fn func(tx Tx) error) error // commits if fn returns nil
```

- Lock rows you will update based on their current value (`SELECT ... FOR UPDATE`), or use a single
  conditional `UPDATE ... WHERE balance >= $1 RETURNING ...`.
- Batch inserts for high-volume paths; avoid per-row round trips in loops.

### Money

- No `float32`/`float64` anywhere in monetary paths, including intermediate calculations and tests.
- The monetary type and scale are **not yet decided** (ADR-0019). Until accepted, do not implement monetary
  types; when accepted, all amounts use one shared type with explicit currency and scale, and one shared
  rounding function. Rules: [financial integrity](../architecture/financial-integrity.md).

### Logging

```go
logger.InfoContext(ctx, "message submitted",
	"message_id", msg.ID, "tenant_id", msg.TenantID, "provider", p.Name(),
	"recipient", mask.Phone(msg.To)) // never the full number, never the body
```

### Testing style

```go
func TestSegmentCount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		text string
		want int
	}{
		{"empty", "", 1},           // expected values must be justified by the encoding rules
		{"gsm7 single", strings.Repeat("a", 160), 1},
		{"gsm7 two", strings.Repeat("a", 161), 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := SegmentCount(tt.text); got != tt.want {
				t.Errorf("SegmentCount() = %d, want %d", got, tt.want)
			}
		})
	}
}
```

(Whether an empty message is allowed at all is a product rule; the example only shows the style.)

### Tooling (proposed, confirmed at scaffolding)

`gofmt`/`goimports`, `go vet`, `golangci-lint` (with a committed config), `govulncheck`, `go test -race`.

---

## Frontend

### Folder structure (per app)

```
src/
├── app/                 App.tsx, router, providers, global error boundary
├── features/
│   └── campaigns/
│       ├── components/  CampaignForm.tsx, CampaignList.tsx
│       ├── hooks/       useCampaigns.ts
│       ├── api.ts       calls via lib/api only
│       ├── schemas.ts   validation schemas
│       └── *.test.tsx
├── components/
│   ├── ui/              Button.tsx, Input.tsx, Table.tsx, Dialog.tsx, Badge.tsx ...
│   └── layout/          AppShell.tsx, PageHeader.tsx
├── lib/
│   ├── api/             client.ts, generated types
│   └── format/          phone, money, date formatting
└── styles/
    └── index.css        Tailwind entry + design tokens
```

### TypeScript configuration (minimum)

```jsonc
{
  "compilerOptions": {
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "noFallthroughCasesInSwitch": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true
  }
}
```

### Components

```tsx
type ButtonProps = {
  variant?: 'primary' | 'secondary' | 'danger';
  size?: 'sm' | 'md';
  loading?: boolean;
} & React.ButtonHTMLAttributes<HTMLButtonElement>;

export function Button({ variant = 'primary', size = 'md', loading = false, ...props }: ButtonProps) {
  // variants defined in one map, using semantic token classes only
}
```

- Named exports; one component per file; file name = component name.
- Props are explicit types; avoid spreading unknown props onto DOM elements without typing.
- Keep components presentational where possible; put data fetching and logic in hooks.
- Complex interactive widgets (dialogs, menus, popovers, comboboxes) follow the component strategy in
  ADR-0013: a headless accessible primitives library may be used once the owner approves one (D-140);
  styling is always Tailwind. Until then, do not add such a library.

### Tailwind CSS and design tokens

Tokens are defined **once** as CSS variables in the Tailwind entry file and referenced through
semantic utility names. Illustrative shape (exact syntax depends on the Tailwind version pinned at
scaffolding; Tailwind v4 defines theme tokens in CSS):

```css
@import "tailwindcss";

@theme {
  --color-surface: ...;        /* page and card backgrounds */
  --color-surface-muted: ...;
  --color-text: ...;
  --color-text-muted: ...;
  --color-primary: ...;
  --color-danger: ...;
  --color-success: ...;
  --color-warning: ...;
  --radius-card: ...;
}
```

Rules:
- Components use semantic classes (`bg-surface text-text-muted border-danger`). Never raw hex values or
  arbitrary values (`bg-[#0f172a]`) in components.
- Status colours (delivered, failed, pending) map to semantic tokens and are always paired with text/icon.
- Spacing and typography use Tailwind's scale; extensions go through tokens.
- Class lists that become long are a sign a reusable component is needed.
- Automatic class sorting (e.g. Prettier's Tailwind plugin) is proposed at scaffolding.

### API integration

- One API client wraps `fetch`: base URL, credentials, JSON parsing, problem-details error mapping,
  request IDs.
- Types are generated from `contracts/openapi/` — do not hand-write response types.
- Feature `api.ts` files expose typed functions; hooks wrap them with the data-fetching library.

### Forms

- Schema-based validation; the same schema drives field errors and submit enabling.
- Server validation errors (problem details with field errors) are mapped back to fields.
- Disable submit while pending; show progress; prevent duplicate submissions.
- Destructive or costly actions (send campaign, adjust credits) require explicit confirmation showing the
  consequence (recipients, segments, estimated cost).

### States

Every data-driven view implements: **loading** (skeleton), **empty** (explanation + primary action),
**error** (friendly message + retry), **success**. Partial failures (e.g. import with some invalid rows)
show what succeeded and what failed.

### Formatting conventions

- Phone numbers displayed in a readable local format; stored/sent as E.164.
- Money: display amounts provided by the server; never compute with JavaScript numbers. The API's
  monetary format (string with documented scale) is defined by ADR-0019 (open); formatting for display
  (e.g. with `Intl.NumberFormat('en-PH', { style: 'currency', currency: 'PHP' })`) must not lose
  precision — convert from the exact string representation, not via floating point.
- Dates: display in `Asia/Manila` unless the user chooses otherwise; show timezone in reports/exports.

### Accessibility checklist (per component/page)

- [ ] Semantic elements (`button`, `nav`, `main`, `table`, `label`).
- [ ] Keyboard: reachable, operable, logical order, visible focus.
- [ ] Labels and error associations for every input.
- [ ] Contrast meets WCAG 2.2 AA.
- [ ] Dialogs: focus trap, `Esc` closes, focus returns to trigger.
- [ ] Status changes announced to assistive technology where relevant.

---

## SQL

- Keywords uppercase or lowercase consistently (choose at scaffolding; formatter enforced).
- One migration per logical change; include comments explaining non-obvious constraints.
- Name every constraint and index explicitly (see `AGENTS.md` §10.2).
- No `SELECT *` in application queries.
