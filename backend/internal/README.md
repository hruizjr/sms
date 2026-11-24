# backend/internal/

Private application modules. Code here can only be imported from within `backend/`.

## Module rules

1. **One folder per business capability** (a "module"), not per technical layer. Inside a module,
   separate transport (HTTP handlers), application services, and persistence.
2. **A module owns its database tables.** Other modules must call the owning module's exported Go API;
   they must not query its tables directly.
3. **No grab-bag packages** such as `utils`, `common`, `helpers`, or `misc`.
4. **Shared technical plumbing** (configuration loading, database pool, logging, HTTP server, metrics)
   lives under `platform/` and must not contain business rules.
5. Dependencies point inward: handlers → services → repositories. Domain types never import HTTP or SQL packages.

## Candidate modules (Proposed — not approved, not created)

| Module           | Capability                                                                   |
| ---------------- | ---------------------------------------------------------------------------- |
| `platform/`      | Config, logging, DB pool, HTTP server, metrics, clock, ID generation          |
| `identity/`      | Users, authentication, sessions, MFA, roles and permissions                  |
| `tenancy/`       | Client accounts (tenants), onboarding status, account settings               |
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
