# frontend/

The web applications, built with **React + TypeScript (strict) + Vite + Tailwind CSS v4** in a pnpm
workspace (ADR-0013). Phase 3 Group 5 added the foundation (application shell, the shared UI package with design
tokens, `Button` and `Card`, tooling and CI;
[decision log §10](../docs/engineering/decision-log.md#10-owner-decisions--frontend-foundation-phase-3-group-5-2026-11-25)).
Phase 4 Group 3 added the first working **client portal**
([decision log §20](../docs/engineering/decision-log.md#20-owner-decisions--first-working-client-portal-phase-4-group-3-2026-11-27)):

- **Sign in** with email and password (server-side session cookie; nothing stored in the browser);
- **Messages**: the organization's messages with recipient, sender ID, message ID, submission time and status;
- **Message details** with the customer-visible status history, updated automatically while in progress;
- **Send SMS**: one recipient, an approved sender ID, a character count and the platform's segment estimate, a
  review step, and one idempotency key per intended send (a repeated click or a retry cannot duplicate it).

Phase 4 Group 4 redesigned the portal and added **API keys** for organization owners
([decision log §21](../docs/engineering/decision-log.md#21-phase-4-group-4--enterprise-client-portal-and-api-key-management-2026-11-27)):
a sidebar (menu button below 1024 px), one page header and status badge style, responsive tables, and an API keys
page to list, create (name, permissions and the owner's current password) and revoke keys. A new key is shown **once**, held in memory only, with a
copy button and a `curl` example that uses a placeholder, never the key.

Phase 4 Group 5 refined the portal into an enterprise console without changing its features or API use
([decision log §22](../docs/engineering/decision-log.md#22-phase-4-group-5--enterprise-client-portal-ui-2026-11-28)):
a dark sidebar (a modal menu panel below 1024 px), breadcrumbs, a toolbar and footer inside the messages table
(table from 768 px, the whole row opens the message), a status panel, label–value list, copyable message ID and
status timeline on message details, a step indicator and message preview in the compose review, and refined tokens
with contrast tests.

Phase 4 Group 6 ([decision log §23](../docs/engineering/decision-log.md#23-phase-4-group-6--security-auditability-and-operational-controls-2026-11-28))
adds the **audit log** page (`/audit-log`, owners only, "Security" in the navigation): the organization's security
events newest first, filters by event, outcome and Philippine-time date range, "Load more", and a read-only details
dialog (request ID, IP address, browser). The API keys page shows when each key was **last used** ("Never used"
when it never was; recorded at most once a minute). A sign-in refused by the per-source limits (`429`), or a busy service
(`503`, also on API key creation), explains how long to wait, from `Retry-After`. Each event's outcome badge shows the
stored outcome exactly (an account lock is stored as **Enforced**; no badge relabels another value). Sender and
Viewer see no audit log link; the server refuses them in any case. A sender ID approval by the administration tool
appears as "Sender ID approved" with the sender ID (decision log §23.2).

Not here yet, each needing separate owner approval: an admin portal; billing or cost display; user or sender ID
management; bulk sending; router, data-fetching, form or headless UI libraries.

## Structure

```
frontend/
├── package.json            Workspace scripts, shared tools, exact Node.js and pnpm versions ("engines")
├── pnpm-workspace.yaml     Workspace packages, 7-day package age policy, exact dependency versions (catalog)
├── pnpm-lock.yaml          The only lockfile for the frontend (separate from contracts/)
├── .npmrc                  engine-strict, save-exact
├── eslint.config.js        ESLint (type-aware typescript-eslint, React hooks, project rules)
├── .prettierrc.json        Prettier, with Tailwind class sorting
├── tsconfig.base.json      Shared strict compiler options
├── apps/
│   └── client-portal/      The client portal (admin-portal added later, ADR-0013)
│       ├── index.html
│       ├── vite.config.ts  Vite, React, Tailwind, Vitest; production Content Security Policy; dev proxy
│       ├── config/         Build configuration code (Content Security Policy, development proxy) and its tests
│       ├── scripts/        verify-build.mjs: checks the production build
│       └── src/
│           ├── main.tsx
│           ├── app/                App (routes), error boundary
│           ├── components/         App shell (sidebar, menu panel), page header (breadcrumbs), alerts, empty states,
│           │                       skeletons, form fields, description list, copy button
│           ├── features/auth/      Session context (sign-in, sign-out, central 401 handling), sign-in page
│           ├── features/messages/  Messages list, message details, compose, status presentation
│           ├── features/api-keys/  API keys page, create and revoke dialogs, one-time key display (owners)
│           ├── features/audit-log/ Security audit log page, filters and event details dialog (owners)
│           ├── lib/api/            The typed portal API client; schema.gen.ts GENERATED from the contract
│           ├── lib/router.tsx      Minimal History-API router (no router library, G3-6)
│           ├── styles/index.css    Tailwind entry point; imports the shared theme
│           └── test/               Test setup (network guard), fake API transport, render helper
└── packages/
    └── ui/                 @sms/ui: shared design tokens and primitives (owner decision F8)
        └── src/
            ├── theme.css   Design tokens (Tailwind @theme); default colour palette switched off
            ├── Badge, Button, Card, Dialog, Icon (+ tests)
            ├── tokens.test.ts      WCAG AA contrast checks for the tokens
            └── testing/            Test support: contrast, network guard, styling rules
```

Shared tokens and primitives live in `packages/ui`. Components that only one application uses stay in that
application (`src/components/`). `@sms/ui` is consumed as TypeScript source; it has no build step.

**API types are generated** from [`contracts/openapi/portal-api-v1.yaml`](../contracts/openapi/portal-api-v1.yaml)
with `openapi-typescript` (types only) into `apps/client-portal/src/lib/api/schema.gen.ts` (never edit it). After a
contract change run `pnpm --filter @sms/client-portal generate:api`; `pnpm check:api` (and CI) fails if the
committed file is stale. The browser never uses the public `/v1` API or API keys.

## Prerequisites

- **Node.js 26.13.1** and **pnpm 10.34.9**, exactly. `engine-strict` refuses other versions.
- Install pnpm with `npm install --global pnpm@10.34.9`.

## Commands

Run from `frontend/`:

| Command                             | What it does                                                                            |
| ----------------------------------- | --------------------------------------------------------------------------------------- |
| `pnpm install --frozen-lockfile`    | Install exactly what the lockfile records (what CI does)                                |
| `pnpm dev`                          | Development server for the client portal at http://localhost:5173                       |
| `pnpm build`                        | Production build into `apps/client-portal/dist/`, then build verification               |
| `pnpm preview`                      | Serve the production build at http://localhost:4173                                     |
| `pnpm test`                         | Unit and component tests (Vitest, jsdom, Testing Library) in every package              |
| `pnpm typecheck`                    | TypeScript type check of every package                                                  |
| `pnpm lint`                         | ESLint, failing on any warning                                                          |
| `pnpm format` / `pnpm format:check` | Format with Prettier / check formatting                                                 |
| `pnpm audit`                        | Dependency vulnerability audit; fails on moderate or higher                             |
| `pnpm check:api`                    | Fails if the generated portal API types differ from the contract                        |
| `pnpm check`                        | API types, formatting, lint, type check, tests, build: everything CI runs but the audit |

The commands work the same on Linux, macOS and Windows. CI runs them on Linux; Windows compatibility is
validated by the owner's local Windows testing (owner decision, 2026-11-27).

## Dependencies and the package age policy

- Exact versions are defined once, in the `catalog:` section of `pnpm-workspace.yaml`. Each `package.json`
  refers to them as `"catalog:"`.
- Adding a dependency needs owner approval (`AGENTS.md` §2 rule 5).
- **Minimum release age: 7 days** (`minimumReleaseAge: 10080` in `pnpm-workspace.yaml`). It applies to
  direct and transitive packages. Behaviour verified for pnpm 10.34.9 in Group 5:
  - when pnpm resolves versions, it refuses any version published less than 7 days ago
    (`ERR_PNPM_NO_MATURE_MATCHING_VERSION`), and never falls back to a newer one;
  - when the registry metadata has no publication time, it fails (`ERR_PNPM_MISSING_TIME`).
- `minimumReleaseAgeStrict` and `minimumReleaseAgeIgnoreMissingTime` are also set, as the owner requested.
  They are pnpm 11 settings that pnpm 10.34.9 does not read; its behaviour already matches them.
- **Limitation:** pnpm 10 applies the age check when it _resolves_ versions: adding, updating, or creating
  the lockfile. Installing from an existing lockfile (`--frozen-lockfile`, as CI does) does not re-check
  the versions already recorded there. The lockfile must therefore always be produced by pnpm with this
  configuration, and every lockfile change is reviewed. In Group 5:
  - every one of the 275 package versions in the committed lockfile was checked against the registry and
    found to be at least 7 days old;
  - a from-scratch resolution reproduced the lockfile byte for byte.

## Design tokens and components

- Colours are **semantic tokens** only: `bg-surface`, `text-text-muted`, `border-border-strong`,
  `bg-primary`, `text-danger`, and so on. Tailwind's default palette is switched off, so classes like
  `bg-blue-500` produce nothing.
- A test rejects default palette colour names and arbitrary values (`bg-[#…]`) in source files.
- ESLint rejects inline `style` attributes, unless a disable comment on that line gives the reason.
- `tokens.test.ts` checks WCAG 2.2 AA contrast:
  - 4.5:1 for every text and background pair the components use;
  - 3:1 for the focus indicator and component boundaries.
- Every focusable element gets the same visible focus outline: 2 px in the `focus` colour, offset onto the
  background.
- Light theme only. The colours and the "Client Portal" label are neutral placeholders until branding is
  decided. Fonts are the operating system's own.
- Responsive layout is mobile-first and checked at 360, 768 and 1280 px.

## Frontend security rules

**Network calls only through the typed portal API client, to the page's own origin; no browser persistence.**
The session is an `HttpOnly` cookie managed by the server; the CSRF token is kept in memory only and sent in
`X-CSRF-Token` on every state-changing request. Three layers enforce the rules:

1. **ESLint** rejects:
   - `fetch` everywhere except `apps/client-portal/src/lib/api/client.ts` (owner decision G3-6);
   - `XMLHttpRequest`, `WebSocket`, `EventSource` and `navigator.sendBeacon`;
   - `localStorage`, `sessionStorage`, `indexedDB`, `caches`, `navigator.serviceWorker` and `document.cookie`;
   - `dangerouslySetInnerHTML`.
2. **Tests** replace the browser network APIs with functions that throw, so any test that reaches the real
   network fails (`@sms/ui/testing`, `installNetworkGuard`). Pages are tested through an injected fake
   transport (`src/test/fake-api.ts`).
3. **The production build** contains a Content Security Policy in a `<meta>` element:
   - `connect-src 'self'` lets the page call only its own origin (the portal API); every other host is refused;
   - only same-origin scripts and styles are allowed, with nothing inline.

   `scripts/verify-build.mjs` fails the build if the policy is missing or misplaced, if `index.html` has
   inline code, or if source maps are produced.

   The development server is not covered by this policy, because hot reload needs a WebSocket and inline
   scripts. When hosting is decided, the policy (plus `frame-ancestors`) should also be sent as an HTTP
   header.

The build reads no environment variables. Anything exposed to the Vite client bundle (`VITE_*`) is public,
so secrets never go there (`AGENTS.md` §8.10).

## Error boundary scope

`AppErrorBoundary` shows a "Something went wrong" screen with a reload button when a component below it
fails while rendering, in a lifecycle method or in a constructor. It does **not** catch errors in event
handlers or asynchronous code (promises, timers). It also does not catch errors in the boundary itself or
in the application shell outside it. Error details are never shown to the user.

## Running with the local API (development proxy)

Start the backend stack from the repository root with
`docker compose -f infra/local/compose.yaml up -d --build --wait` ([`infra/local/README.md`](../infra/local/README.md)),
then run `pnpm dev` here and open http://localhost:5173.

To sign in you need a portal user: create one with `smsctl portal-user create`
([`infra/local/README.md` §4](../infra/local/README.md#4-use-the-client-portal-in-the-browser-phase-4-group-3)).
Open the portal as **http://localhost:5173** (the session cookie is `Secure`; browsers accept that over plain
HTTP for `localhost` only).

The Vite development server forwards **`/healthz`, `/readyz` and `/portal`** to the API at
`http://127.0.0.1:8080` (`apps/client-portal/config/dev-proxy.ts`; owner decisions L5 and G3-6):

- **One origin:** the portal and the API share one origin in development, so the session cookie is
  first-party and the API needs no CORS.
- **The public API is not proxied:** no `/v1` path. Any other path is answered by Vite itself.
- **The proxy applies to `pnpm dev` only**, not to `pnpm preview`.
- **The target is fixed at port 8080**, the default `API_PORT`. Making it configurable would need Node.js
  type definitions, which are not added.

The "Local stack check" CI job starts the stack and `pnpm dev`, then checks:

- `/readyz` and `/healthz` answer through the proxy;
- `/v1/messages` is not proxied.

## Routing

No router library (owner decision G3-6): `src/lib/router.tsx` (about 50 lines on the History API) provides the
current path, navigation and links. Routes: `/messages` (also `/`), `/messages/new`, `/messages/{id}`; without a
session every path shows the sign-in page. Hosting must serve `index.html` for every client-side path (Vite's
development server already does).

Follow [`AGENTS.md`](../AGENTS.md) §8 (Frontend standards).
