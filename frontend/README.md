# frontend/

The web application(s), built with **React + TypeScript (strict) + Vite + Tailwind CSS**.
**Not yet scaffolded** — no `package.json` or source code exists in Phase 1.

## Open decision: one app or two? (ADR-0013 — Proposed)

The product has two very different audiences:

- **Client Portal** — used by customer businesses.
- **Super Admin Portal** — used by our own staff, with powerful permissions (credits, pricing, providers).

| Option | Pros | Cons |
| ------ | ---- | ---- |
| A. One app, role-based routes | Simplest build and deploy | Admin code ships to every customer's browser; one compromise surface |
| B. Two apps sharing a component package | Admin can live on a separate domain behind extra network controls (e.g. IP allowlist/VPN); smaller attack surface | Slightly more build configuration (a workspace) |

**Recommendation (pending approval): Option B**, introduced as a small npm/pnpm workspace:

```
frontend/
├── apps/
│   ├── client-portal/
│   └── admin-portal/
└── packages/
    └── ui/            Shared design tokens and components
```

## Proposed structure inside each app

```
src/
├── app/               Bootstrap, routing, providers, error boundary
├── features/<name>/   Feature-specific components, hooks, API calls, tests
├── components/ui/     Design-system primitives (Button, Input, Table, Dialog, ...)
├── components/layout/ Shells, navigation, page layouts
├── lib/api/           The single typed API client (types generated from contracts/)
├── lib/               Small framework-agnostic helpers
└── styles/            Tailwind entry point and design tokens (CSS variables)
```

Follow [`AGENTS.md`](../AGENTS.md) §8 (Frontend standards).
