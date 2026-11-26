# frontend/

The web applications, built with **React + TypeScript (strict) + Vite + Tailwind CSS v4** in a pnpm
workspace (ADR-0013). Phase 3 Group 5 added the foundation:

- the **client portal** application shell (one page; no features);
- the shared UI package, with design tokens, `Button` and `Card`;
- tooling and CI.

The owner's decisions for this group are in
[decision log §10](../docs/engineering/decision-log.md#10-owner-decisions--frontend-foundation-phase-3-group-5-2026-11-25).

Not here yet, each needing separate owner approval:

- an admin portal;
- a router, and libraries for data fetching, forms, or headless UI primitives;
- API calls or an API client;
- authentication;
- any portal feature.

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
│           ├── app/                App, error boundary
│           ├── components/layout/  App shell, skip link (app-specific components)
│           ├── features/home/      The only page
│           ├── styles/index.css    Tailwind entry point; imports the shared theme
│           └── test/setup.ts       Test setup (jest-dom matchers, network guard)
└── packages/
    └── ui/                 @sms/ui: shared design tokens and primitives (owner decision F8)
        └── src/
            ├── theme.css   Design tokens (Tailwind @theme); default colour palette switched off
            ├── Button.tsx, Card.tsx (+ tests)
            ├── tokens.test.ts      WCAG AA contrast checks for the tokens
            └── testing/            Test support: contrast, network guard, styling rules
```

Shared tokens and primitives live in `packages/ui`. Components that only one application uses stay in that
application (`src/components/`). `@sms/ui` is consumed as TypeScript source; it has no build step.

There is **no `src/lib/api/` and no generated API types yet** (owner decision F3). Generating TypeScript
types from the OpenAPI contract is deferred until a portal endpoint exists. The public `/v1` API uses API
keys, which must never be used in a browser. The portal will need a separately approved,
session-authenticated portal API contract.

## Prerequisites

- **Node.js 26.13.1** and **pnpm 10.34.9**, exactly. `engine-strict` refuses other versions.
- Install pnpm with `npm install --global pnpm@10.34.9`.

## Commands

Run from `frontend/`:

| Command                             | What it does                                                                          |
| ----------------------------------- | ------------------------------------------------------------------------------------- |
| `pnpm install --frozen-lockfile`    | Install exactly what the lockfile records (what CI does)                              |
| `pnpm dev`                          | Development server for the client portal at http://localhost:5173                     |
| `pnpm build`                        | Production build into `apps/client-portal/dist/`, then build verification             |
| `pnpm preview`                      | Serve the production build at http://localhost:4173                                   |
| `pnpm test`                         | Unit and component tests (Vitest, jsdom, Testing Library) in every package            |
| `pnpm typecheck`                    | TypeScript type check of every package                                                |
| `pnpm lint`                         | ESLint, failing on any warning                                                        |
| `pnpm format` / `pnpm format:check` | Format with Prettier / check formatting                                               |
| `pnpm audit`                        | Dependency vulnerability audit; fails on moderate or higher                           |
| `pnpm check`                        | Formatting check, lint, type check, tests, build: everything CI runs except the audit |

The commands work the same on Linux, macOS and Windows. CI runs them on Linux and Windows.

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

**No network calls and no browser persistence until a session-authenticated portal API contract is
approved.** Three layers enforce this:

1. **ESLint** rejects:
   - `fetch`, `XMLHttpRequest`, `WebSocket`, `EventSource` and `navigator.sendBeacon`;
   - `localStorage`, `sessionStorage`, `indexedDB`, `caches`, `navigator.serviceWorker` and `document.cookie`;
   - `dangerouslySetInnerHTML`.
2. **Tests** replace the browser network APIs with functions that throw, so any test that triggers a
   request fails (`@sms/ui/testing`, `installNetworkGuard`).
3. **The production build** contains a Content Security Policy in a `<meta>` element:
   - `connect-src 'none'` makes the browser refuse network calls from the page;
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

The Vite development server forwards **only `/healthz` and `/readyz`** to the API at
`http://127.0.0.1:8080` (`apps/client-portal/config/dev-proxy.ts`; owner decision L5, Phase 3 Group 6):

- **One origin:** the portal and the API share one origin in development, so the API needs no CORS.
  This also suits the future server-managed session cookies.
- **Nothing else is proxied:** no `/v1` or other API path. Any other path is answered by Vite itself.
- **No frontend code calls these endpoints.** The ESLint network rules, the test network guard and the
  production `connect-src 'none'` policy are unchanged.
- **The proxy applies to `pnpm dev` only**, not to `pnpm preview`.
- **The target is fixed at port 8080**, the default `API_PORT`. Making it configurable would need Node.js
  type definitions, which are not added.

The "Local stack check" CI job starts the stack and `pnpm dev`, then checks:

- `/readyz` and `/healthz` answer through the proxy;
- `/v1/messages` is not proxied.

## Routing

There is no router (ADR-0013 defers routing to Phase 4). The portal has one view at `/`. Navigation is
added when a second real page exists. Hosting will then need to serve `index.html` for every client-side
path.

Follow [`AGENTS.md`](../AGENTS.md) §8 (Frontend standards).
