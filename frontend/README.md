# frontend/

The web application(s), built with **React + TypeScript (strict) + Vite + Tailwind CSS**.
**Not yet scaffolded** — created in Phase 3, Group 4.

## Structure (ADR-0013 — accepted 2026-11-25)

The product has two audiences: customer businesses (**Client Portal**) and our own staff (**Super Admin
Portal**, with powerful permissions). The owner decided:

- a pnpm workspace with **one application initially**, `client-portal`, and a shared `packages/ui` package;
- an `admin-portal` application is added later, when admin functionality begins, so admin code can live on
  a separate origin;
- router, data-fetching, and form libraries are deferred to Phase 4; the headless UI library decision
  (D-140) is deferred.

```
frontend/
├── apps/
│   └── client-portal/     (admin-portal added later)
└── packages/
    └── ui/                Shared design tokens and components
```

Toolchain: React, TypeScript (strict), Vite, Tailwind CSS v4, pnpm 10 (exact version pinned), Node.js 26 LTS.

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

## Component strategy (ADR-0013 — accepted 2026-11-25)

Tailwind CSS is the only styling system. Complex interactive widgets (dialogs, menus, dropdowns, popovers,
comboboxes) may use a maintained, accessible **headless** primitives library — selection is an owner
decision (D-140), and no package is added until approved. No pre-styled UI framework.

Follow [`AGENTS.md`](../AGENTS.md) §8 (Frontend standards).
