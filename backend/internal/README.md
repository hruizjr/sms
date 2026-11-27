# backend/internal/

Private application modules. Code here can only be imported from within `backend/`.

## Module rules

1. **One folder per business capability** (a "module"), not per technical layer. Inside a module,
   separate transport (HTTP handlers), application services, and persistence.
2. **A module owns its database tables.** Other modules must call the owning module's exported Go API;
   they must not query its tables directly.
3. **No grab-bag packages** such as `utils`, `common`, `helpers`, or `misc`.
4. **Shared technical plumbing** lives in one focused package per concern directly under `internal/`
   and must not contain business rules. There is no `platform/` umbrella (owner decision, Phase 3
   Group 2). Current packages: `config`, `database` (with `database/dbroles` for the role definitions,
   `database/dbtest` for integration tests, `database/rlsguard` for the RLS catalog guard and
   `database/sqlcgen` for sqlc-generated queries, never edited by hand), `migration`, `httpserver`,
   `middleware`, `observability`, and `publicapi` (code generated from the public API contract only; never
   edited by hand — see `backend/README.md`).
5. Dependencies point inward: handlers → services → repositories. Domain types never import HTTP or SQL packages.

## Business modules

| Module | Capability | Status |
| ------ | ---------- | ------ |
| `tenancy/` | Tenant registry: create, get, list, activate, suspend, reactivate (platform-owned `tenants` table; no HTTP endpoints) | Phase 4 Group 1 |

## Candidate business modules (Proposed — not approved, not created)

| Module           | Capability                                                                   |
| ---------------- | ---------------------------------------------------------------------------- |
| `identity/`      | Users, authentication, sessions, MFA, roles and permissions                  |
| `tenancy/`       | (registry created in Phase 4 Group 1) onboarding status, account settings     |
| `audit/`         | Append-only audit trail of sensitive actions                                 |
| `billing/`       | Pricing, credit ledger, reservations, settlement, reconciliation             |
| `senderids/`     | Sender ID requests, approval workflow, per-telco registration status         |
| `contacts/`      | Contacts, groups, imports                                                    |
| `compliance/`    | Consent records, opt-outs, suppression lists, content rules                  |
| `campaigns/`     | Campaign definition, scheduling, approval                                    |
| `messaging/`     | Message acceptance, segmentation, idempotency, routing, state machine        |
| `providers/`     | The provider interface plus one adapter per telco and a **simulator** adapter |
| `deliveryreports/` | Delivery receipt ingestion and correlation                                 |
| `webhooks/`      | Outbound customer notifications with signing and retries                     |
| `apikeys/`       | Public API credentials                                                       |

The final module list is an architecture-phase decision.
