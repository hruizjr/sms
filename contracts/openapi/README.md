# Public API v1 — contract guide

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft contract — Phase 3 Group 3; **not implemented**; provisional items pending owner decisions (§7) | Project owner | 2026-11-25 |

The machine-readable source of truth is [`public-api-v1.yaml`](public-api-v1.yaml) (OpenAPI 3.1,
ADR-0015). This guide explains the semantics behind it for reviewers and future implementers. The
specification and this guide are checked against each other by the contract tests
(`contracts/tests/`), so the tables in §3 and §5 must stay in this exact format.

This contract defines **our own** client-facing API. It is independent of any SMS gateway, protocol,
network, or library: no network protocol terms, network message IDs, raw network status codes, or
gateway-specific behaviour appear in it. Network behaviour is translated into our status model inside
provider adapters ([messaging reliability §5.1](../../docs/architecture/messaging-reliability.md#51-adapter-boundary)).

## 1. Scope

| In this contract | Not in this contract (and why) |
| ---------------- | ------------------------------ |
| `POST /v1/messages` — accept one message to one recipient | **Bulk submission** — all-or-nothing vs per-recipient semantics is open (D-094e) |
| `GET /v1/messages/{message_id}` — current status of one message | **Webhooks / status callbacks** — deferred in the candidate MVP (product brief §5); principles in §6 |
| `GET /v1/messages` — list with cursor pagination and filters | **Cost estimate / prices / balances** — monetary representation is open (ADR-0019) |
| API key authentication, permissions, problem details, idempotency | **Portal (browser) API**, sender ID management, API key management, scheduling, campaigns, inbound messages — later phases |
| | Health/readiness endpoints — operational, not client-facing (`backend/README.md`) |

## 2. Acceptance is not submission, and not delivery

The API reports milestones from [messaging reliability §3](../../docs/architecture/messaging-reliability.md#3-message-lifecycle-milestones)
separately. A client must never treat an earlier milestone as proof of a later one.

| Client sees | Milestone | Proves | Does **not** prove |
| ----------- | --------- | ------ | ------------------ |
| `202 Accepted` + `status: accepted` | M1 (and later M2, M3 internally) | The platform validated, stored, and owns the message | That any network has seen it |
| `status: provider_accepted` | M4 | A network acknowledged and accepted the message | That the handset received it |
| `status: delivered` / `undelivered` | M5 | The network's final report | Absolute truth; reports can be late, duplicated, conflicting, or wrong |
| `status: submission_uncertain` | M3 without M4 | Hand-off was attempted; the answer was lost | Either success or failure |
| `status: unknown` | M4 without M5 in the window | No final report arrived in time | Non-delivery |

**No exactly-once promise.** The idempotency key prevents duplicate *operations inside our platform*
(one message record, one credit reservation per key). It cannot prevent a duplicate SMS when a network
times out after accepting a message; that case is reported as `submission_uncertain` and is **not**
resent automatically ([messaging reliability §4](../../docs/architecture/messaging-reliability.md#4-provider-submission-and-ambiguous-outcomes)).
Resolving it is per-provider policy (D-095).

## 3. Status reference (customer-visible statuses)

Provisional set (D-099; internal states from ADR-0021, both Proposed). Every internal state maps to
exactly one customer status. Clients should branch on `status_category` for robustness, since new status
values may be added within `/v1`.

| Status | Category | Internal states (ADR-0021) | Meaning for the client |
| ------ | -------- | -------------------------- | ---------------------- |
| `accepted` | `in_progress` | ACCEPTED, QUEUED, SUBMITTING | Accepted by the platform; no network has acknowledged it yet. |
| `provider_accepted` | `in_progress` | SUBMITTED | A network acknowledged and accepted it; awaiting a delivery report. |
| `submission_uncertain` | `uncertain` | SUBMIT_UNCERTAIN | Hand-off outcome unknown; it **may already be delivered**. Not resent automatically. Resending it yourself may create a duplicate SMS. |
| `delivered` | `final` | DELIVERED | The network reported delivery. |
| `undelivered` | `final` | UNDELIVERED | The network reported a final non-delivery, including expiry. |
| `unknown` | `uncertain` | UNKNOWN | No final report within the reporting window. **Not** proof of non-delivery; a late report can still change it. |
| `rejected` | `final` | REJECTED | Refused by the platform after acceptance (e.g. opt-out found before hand-off) or explicitly refused by a network. |
| `failed` | `final` | FAILED | Could not be handed off: permanent hand-off error, retries exhausted, or too old to send. |

Rules that follow from [messaging reliability §5](../../docs/architecture/messaging-reliability.md#5-delivery-receipt-dlr-handling):

- Statuses change by **precedence, not arrival order**. A `final` status is never replaced by an
  `in_progress` one. `submission_uncertain` and `unknown` can be replaced by a later network report.
- Duplicate network reports do not change status twice and never cause duplicate charges.
- If a network sends conflicting final reports, the current status is kept and flagged for
  reconciliation; the resolution policy is open (D-098).
- How per-segment reports roll up into one message status is open (D-097).
- `status_reason.code` gives the platform-defined reason (initial set in the specification); raw network
  codes are never exposed. Mapping network codes to these reasons happens in provider adapters from
  network documentation.

## 4. Idempotency (proposed, ADR-0018)

Summary of [API idempotency](../../docs/architecture/api-idempotency.md) as applied here:

- `Idempotency-Key` is **required** on `POST /v1/messages` (D-094a); scope is account + operation (D-094b).
- Fingerprint = the operation plus the parsed JSON body; whitespace and key order do not matter.
- Same key + same body → original response replayed (`202`, same body and message `id`,
  `Idempotent-Replayed: true`). The replay shows the **acceptance-time** state; use `GET` for the current status.
- Same key + different body → `422 idempotency-key-reused`. Same key while in progress → `409` + `Retry-After`.
- Not recorded (safe to fix and retry with the same key): `400`, `401`, `403`, `402` (D-094c), `413`,
  `415`, `422` other than key reuse, `429`, and `5xx` where nothing was committed.
- Retention: not yet decided (D-094d); recommendation is at least 24 hours. It will be published in
  the specification once decided.
- `client_reference` is a separate, optional label for the client's own correlation. It is **not**
  deduplicated.

**Client retry rule:** after a timeout, connection error, `409`, `429`, `500`, or `503`, retry with the
same key and the same body. Never generate a new key for a retry of the same intended message.

## 5. Problem types

All errors are RFC 9457 `application/problem+json` documents with `type`, `title`, `status`, optional
`detail`, `request_id` and (validation only) `errors`. Generic HTTP errors use `type: about:blank`,
as the Group 1 skeleton already does. Problem-type URIs are
`https://api.example.invalid/problems/<slug>`; the host is a placeholder until the public domain is
decided (D-145).

| Slug | Status | When |
| ---- | ------ | ---- |
| `malformed-request` | 400 | Body is not valid JSON |
| `validation-error` | 400 | A field, parameter, or cursor is invalid; see `errors` |
| `idempotency-key-missing` | 400 | `Idempotency-Key` header missing |
| `idempotency-key-invalid` | 400 | `Idempotency-Key` value malformed |
| `unauthenticated` | 401 | Missing, malformed, unknown, revoked, or expired API key |
| `insufficient-credit` | 402 | Not enough available credit to accept the message (provisional) |
| `forbidden` | 403 | Key lacks the required permission, or the account may not use the operation |
| `idempotency-request-in-progress` | 409 | Same key still being processed |
| `payload-too-large` | 413 | Body exceeds the size limit |
| `unsupported-media-type` | 415 | Body is not `application/json` |
| `idempotency-key-reused` | 422 | Same key, different body |
| `sender-id-not-approved` | 422 | `sender_id` is not approved for the account |
| `destination-not-supported` | 422 | Destination not served (initially non-Philippine numbers, D-043) |
| `message-too-long` | 422 | Body needs more segments than allowed (D-144) |
| `recipient-opted-out` | 422 | Recipient is on a suppression list |
| `rate-limited` | 429 | Rate limit exceeded; see `Retry-After` |

`404`, `405`, `500`, and `503` use `about:blank`. A message of another account always returns `404`.

## 6. Webhooks (not in this contract)

Webhooks are deferred in the candidate MVP. When they are added (as a separate, owner-approved
change under `contracts/webhooks/`), they must follow the rules already set elsewhere: delivered
**at least once** with a stable event ID for deduplication ([messaging reliability §2.4](../../docs/architecture/messaging-reliability.md#24-consumer-requirements-duplicate-detection-and-handling)),
signed with HMAC and a timestamp, sent only to URLs validated against SSRF
([security baseline §7](../../docs/security/security-baseline.md#7-application-and-api-security)),
carrying the same status values as §3, and possibly arriving out of order (clients use
`status_updated_at` and `GET` as the authority).

## 7. Decisions for the owner

Every value below is written into the specification as **provisional** and marked so. Changing any of
them before `v1` is published is cheap; after publication most are breaking changes.

| ID | Decision | Provisional choice in the spec | Alternatives |
| -- | -------- | ------------------------------ | ------------ |
| D-094a | Idempotency key required or optional | Required | Optional (lower integration friction; a later switch to required is breaking) |
| D-094b | Key scope | Account (tenant) + operation | Per API key + operation; per account across operations |
| D-094c | Record business rejections (insufficient credit) | Not recorded; retry after top-up succeeds | Record and replay the rejection |
| D-094d | Key retention | Not stated; ≥ 24 h recommended | 24 h / 72 h / 7 days |
| D-094e | Bulk semantics | Bulk endpoint omitted | All-or-nothing; per-recipient results |
| D-096 / D-099 | Show `SUBMIT_UNCERTAIN` to customers | Separate status `submission_uncertain` | Fold into `unknown`; show as `accepted` (hides risk; not recommended) |
| D-099 | Customer-visible status set and names | 8 statuses + `status_category` (§3) | Expose internal states 1:1; coarser set (e.g. pending/delivered/failed — loses the uncertain distinction) |
| D-099 | Initial `status_reason` codes | 10 platform-defined codes (spec `StatusReason`) | Fewer, broader codes; add codes when adapters are built |
| D-143 | API key presentation and permission names | `Authorization: Bearer <key>`; `messages:send`, `messages:read` | `X-API-Key` header; other permission model (security baseline S-2/S-3) |
| D-144 | Maximum message length | `body` ≤ 1,600 characters (structural); max segments undecided → `422 message-too-long` | A fixed segment limit (e.g. 6 or 10), once network limits (D-105) are known |
| D-145 | Public API domain and problem-type URI base | Placeholder `https://api.example.invalid` | Final API and documentation domains |
| — | Insufficient credit status code | `402` | `422` or `409` with the same problem type |

Unchanged and still open: D-095 (ambiguous submission policy), D-097 (segment roll-up), D-098
(conflicting finals), ADR-0019 (money; no amounts in this contract).

## 8. Validation

From `contracts/` (Node.js 26.13.1 and pnpm 10.34.9, pinned in `package.json`):

```bash
pnpm install --frozen-lockfile
pnpm run check   # redocly lint + bundle + contract tests
```

- `pnpm run lint` — OpenAPI validity, style rules in `redocly.yaml`, and every example validated
  against its schema.
- `pnpm run bundle` — writes the single-file bundle to `build/` (git-ignored).
- `pnpm run test` — project rules (`tests/public-api-v1.test.mjs`): `/v1` paths, authentication and
  permission on every operation, problem details on every error, `snake_case`, required
  `Idempotency-Key`, `202` for acceptance, no message text or network identifiers in responses, the
  status table above matches the specification, no gateway or protocol terms, and no exactly-once claim.

CI runs the same commands (`.github/workflows/contracts-ci.yml`, job **OpenAPI contract check**).
TypeScript type generation (`openapi-typescript`, approved) is added with its first consumer, the
frontend (Group 4). Go server-side validation or code generation is not chosen (ADR-0015) and needs
owner approval.
