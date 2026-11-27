# Public API v1 — contract guide

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft contract — Phase 3 Group 3; **implemented in Phase 4 Group 2** against the simulator provider only (no rate limits, credit checks or suppression lists yet); owner decisions of 2026-11-25 applied; remaining provisional items in §7 | Project owner | 2026-11-27 |

The machine-readable source of truth is [`public-api-v1.yaml`](public-api-v1.yaml) (OpenAPI 3.1,
ADR-0015). This guide explains the semantics behind it for reviewers and future implementers. The
specification and this guide are checked against each other by the contract tests
(`contracts/tests/`), so the tables in §3 and §5 must stay in this exact format.

This contract defines **our own** client-facing API. It is independent of any SMS gateway, protocol,
provider, network, or library: no protocol terms, provider message IDs, raw provider status codes, or
gateway-specific behaviour appear in it. Provider behaviour is translated into our status model inside
provider adapters ([messaging reliability §5.1](../../docs/architecture/messaging-reliability.md#51-adapter-boundary)).

## 1. Scope

| In this contract | Not in this contract (and why) |
| ---------------- | ------------------------------ |
| `POST /v1/messages` — accept one message to one recipient | **Bulk sending** (many recipients in one request, through the API or the portal) — deferred to a later release by owner decision (D-094e, MVP-08, MVP-09; 2026-11-25) |
| `GET /v1/messages/{message_id}` — current status of one message | **Webhooks / status callbacks** — deferred in the candidate MVP (product brief §5); principles in §6 |
| `GET /v1/messages` — list with cursor pagination and filters | **Cost estimate / prices / balances** — monetary representation is open (ADR-0019) |
| API key authentication, permissions, problem details, idempotency | **Portal (browser) API**, sender ID management, API key management, scheduling, campaigns, inbound messages — later phases |
| | Health/readiness endpoints — operational, not client-facing (`backend/README.md`) |

## 2. Acceptance is not submission, and not delivery

**Provider** means the platform's **immediate upstream SMS provider**: the party the platform hands a
message to. Under the operating model (D-160: direct aggregation only, pending partner ratification), that is a
mobile network operator with which the business integrates directly. Even so, a provider's acknowledgment is
**not necessarily** acceptance by the recipient's mobile network (for example, for a ported or off-network
number), and delivery outcomes are the provider's **reports**, which the platform
relays but cannot independently verify.

The API reports milestones from [messaging reliability §3](../../docs/architecture/messaging-reliability.md#3-message-lifecycle-milestones)
separately. A client must never treat an earlier milestone as proof of a later one.

| Client sees | Milestone | Proves | Does **not** prove |
| ----------- | --------- | ------ | ------------------ |
| `202 Accepted` + `status: accepted` | M1 (and later M2, M3 internally) | The platform validated, stored, and owns the message | That any provider or network has seen it |
| `status: provider_accepted` | M4 | Our immediate upstream SMS provider acknowledged and accepted the message | That the recipient's mobile network accepted it, or that the handset received it |
| `status: delivered` / `undelivered` | M5 | The provider's final report (a provider-reported outcome) | An independently verified fact; reports can be late, duplicated, conflicting, or wrong |
| `status: submission_uncertain` | M3 without M4 | Hand-off to the provider was attempted; the provider's answer was not received | Either success or failure |
| `status: unknown` | M4 without M5 in the window | No final report from the provider arrived in time | Non-delivery |

**No exactly-once promise.** The idempotency key prevents duplicate *operations inside our platform*
(one message record, one credit reservation per key). It cannot prevent a duplicate SMS when a provider
times out after accepting a message; that case is reported as `submission_uncertain` and is **not**
resent automatically ([messaging reliability §4](../../docs/architecture/messaging-reliability.md#4-provider-submission-and-ambiguous-outcomes)).
Resolving it is per-provider policy (D-095).

## 3. Status reference (customer-visible statuses)

Owner-approved set (D-099, 2026-11-25), including `submission_uncertain` as a separate status
(D-096, customer-visible part); new values may be added later within `/v1`. The internal states come
from messaging reliability §3.1. ADR-0021 was accepted on 2026-11-25, **limited to** these eight
statuses and the initial reason codes; it does not settle the billing treatment of `unknown` and other
uncertain messages (D-096) or webhook behaviour (D-099), which remain separate open decisions. Every
internal state maps to exactly one customer status. Clients should branch on `status_category` for robustness, since new status
values may be added within `/v1`.

| Status | Category | Internal states (ADR-0021) | Meaning for the client |
| ------ | -------- | -------------------------- | ---------------------- |
| `accepted` | `in_progress` | ACCEPTED, QUEUED, SUBMITTING | Accepted by the platform; no provider has acknowledged it yet. |
| `provider_accepted` | `in_progress` | SUBMITTED | Our immediate upstream SMS provider acknowledged and accepted it. Not necessarily accepted by the recipient's mobile network; awaiting a delivery report. |
| `submission_uncertain` | `uncertain` | SUBMIT_UNCERTAIN | Outcome of the hand-off to the provider unknown; it **may already be delivered**. Not resent automatically. Resending it yourself may create a duplicate SMS. |
| `delivered` | `final` | DELIVERED | Provider-reported delivery outcome: the provider reported delivery. Not independently verified. |
| `undelivered` | `final` | UNDELIVERED | Provider-reported delivery outcome: the provider reported a final non-delivery, including expiry. Not independently verified. |
| `unknown` | `uncertain` | UNKNOWN | No final report from the provider within the reporting window. **Not** proof of non-delivery; a late report can still change it. |
| `rejected` | `final` | REJECTED | Refused by the platform after acceptance (e.g. opt-out found before hand-off) or explicitly refused by the provider. |
| `failed` | `final` | FAILED | Could not be handed off to the provider: permanent hand-off error, retries exhausted, or too old to send. |

Rules that follow from [messaging reliability §5](../../docs/architecture/messaging-reliability.md#5-delivery-receipt-dlr-handling):

- Statuses change by **precedence, not arrival order**. A `final` status is never replaced by an
  `in_progress` one. `submission_uncertain` and `unknown` can be replaced by a later provider report.
- Duplicate provider reports do not change status twice and never cause duplicate charges.
- If a provider sends conflicting final reports, the current status is kept and flagged for
  reconciliation; the resolution policy is open (D-098).
- How per-segment reports roll up into one message status is open (D-097).
- `status_reason.code` gives the platform-defined reason (initial set in the specification); raw provider
  codes are never exposed. Mapping provider codes to these reasons happens in provider adapters from
  provider documentation.

## 4. Idempotency (ADR-0018)

Summary of [API idempotency](../../docs/architecture/api-idempotency.md) as applied here. ADR-0018 was
accepted by the owner on 2026-11-25, with decisions D-094a–e and the clarifications below.

- `Idempotency-Key` is **required** on `POST /v1/messages` (D-094a); scope is account + operation (D-094b).
- **Comparison:** the operation plus the **normalized request body** (the parsed JSON body; whitespace
  and key order do not matter). `X-Request-ID` and other tracing or transport headers are **not** part
  of the comparison.
- Outcomes for the same account, operation, and key, checked in this order:

  | # | Situation | Response |
  | - | --------- | -------- |
  | 1 | Different normalized body (original still processing **or** completed) | `422 idempotency-key-reused`; nothing processed |
  | 2 | Identical body, original still processing | `409 idempotency-request-in-progress` + `Retry-After` |
  | 3 | Identical body, original completed | Original `202` response and body replayed, `Idempotent-Replayed: true` |

  The replay shows the **acceptance-time** state; use `GET` for the current status.
- Not recorded (safe to fix and retry with the same key): `400`, `401`, `403`, `402` (D-094c, D-146), `413`,
  `415`, `422` other than key reuse, `429`, and `5xx` where nothing was committed.
- **Retention: seven days** (D-094d) after the original request was accepted.
- **After retention expires** the key is forgotten: a request reusing it is processed as a **new**
  request (no fingerprint comparison, no `422`, no replay) and can create a second message, charge, and
  SMS. Clients must not retry with a key older than seven days; they can look up earlier messages with
  `GET /v1/messages?client_reference=...`.
- Bulk requests: none; bulk sending is deferred to a later release (D-094e).
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
| `insufficient-credit` | 402 | Not enough available credit to accept the message (D-146) |
| `forbidden` | 403 | Key lacks the required permission, or the account may not use the operation |
| `idempotency-request-in-progress` | 409 | Same key and identical body; the original is still being processed |
| `payload-too-large` | 413 | Body exceeds the size limit |
| `unsupported-media-type` | 415 | Body is not `application/json` |
| `idempotency-key-reused` | 422 | Same key, different normalized body — even while the original is still being processed |
| `sender-id-not-approved` | 422 | `sender_id` is not approved for the account |
| `destination-not-supported` | 422 | Destination not served (initially non-Philippine numbers, D-043) |
| `message-too-long` | 422 | Body needs more segments than product/provider policy allows (D-144) |
| `recipient-opted-out` | 422 | Recipient is on a suppression list |
| `rate-limited` | 429 | Rate limit exceeded; see `Retry-After` |

`404`, `405`, `500`, and `503` use `about:blank`. A message of another account always returns `404`.

**Request identifier.** Every operation accepts an optional `X-Request-ID` request header and every
response returns `X-Request-ID` (also `request_id` in problem details). This matches the Group 1
middleware (`backend/internal/middleware/requestid.go`): a supplied value of 1-128 characters from
`[A-Za-z0-9._-]` is reused; any other value is ignored (not rejected) and a new identifier is
generated.

## 6. Webhooks (not in this contract)

Webhooks are deferred in the candidate MVP. When they are added (as a separate, owner-approved
change under `contracts/webhooks/`), they must follow the rules already set elsewhere: delivered
**at least once** with a stable event ID for deduplication ([messaging reliability §2.4](../../docs/architecture/messaging-reliability.md#24-consumer-requirements-duplicate-detection-and-handling)),
signed with HMAC and a timestamp, sent only to URLs validated against SSRF
([security baseline §7](../../docs/security/security-baseline.md#7-application-and-api-security)),
carrying the same status values as §3, and possibly arriving out of order (clients use
`status_updated_at` and `GET` as the authority).

## 7. Owner decisions and remaining provisional items

**Decided by the owner on 2026-11-25** (recorded in the
[discovery register](../../docs/requirements/discovery-register.md) and the
[owner discovery workbook](../../docs/discovery/owner-discovery-workbook.md)), together with the
acceptance of ADR-0018 and ADR-0021 in the [decision log](../../docs/engineering/decision-log.md#8-owner-decisions--public-api-contract-2026-11-25).

| ID | Decision | Applied in the specification |
| -- | -------- | ---------------------------- |
| D-094a | Idempotency key required | `Idempotency-Key` required on `POST /v1/messages` |
| D-094b | Key scope: account + operation | `IdempotencyKey` description |
| D-094c | Insufficient-credit rejections are not recorded | `IdempotencyKey`, `InsufficientCredit` |
| D-094d | Retention: seven days | `IdempotencyKey` (retention and expiry) |
| D-094e | Bulk sending through the API **and** the portal deferred to a later release (MVP-08, MVP-09) | No bulk endpoint |
| D-096 (customer-visible part) | `submission_uncertain` is a separate status | `MessageStatus`, §3 |
| D-099 (status part) | The eight statuses and ten initial reason codes, extensible | `MessageStatus`, `StatusCategory`, `StatusReason`, §3 |
| D-143 | Bearer API key; permissions `messages:send`, `messages:read` | `api_key` security scheme, `x-required-permission` |
| D-144 | Maximum API message length 1,600 characters; maximum segment count deferred to product/provider policy | `MessageCreateRequest.body` |
| D-145 | API domain stays a placeholder for now | `servers`, problem-type URIs (`https://api.example.invalid`) |
| D-146 | HTTP `402` for insufficient credit | `InsufficientCredit` response |
| D-147 | Explicit `sender_id` required; no account-level default | `MessageCreateRequest.sender_id` (required), `SenderId` |
| ADR-0018 | Accepted, including the outcome order (different body → `422` even while processing; identical body → `409` while processing, replay when completed) and body-only comparison | `IdempotencyKey`, `IdempotencyRequestInProgress`, `UnprocessableMessage`, §4 |
| ADR-0021 | Accepted, limited to the eight statuses and the initial reason-code vocabulary | `MessageStatus`, `StatusReason`, §3 |

**Still provisional or open** (not changed by these decisions):

| Item | Current state in the specification |
| ---- | ---------------------------------- |
| Replay header name `Idempotent-Replayed` | Provisional; name marked "TBD" in [API idempotency §3.5](../../docs/architecture/api-idempotency.md#35-outcomes) and not part of the D-094 decisions |
| D-145 final domain | Placeholder until decided |
| D-096 billing treatment of `unknown` and other uncertain messages | **Open**, separate decision; not settled by ADR-0021; no prices in this contract |
| D-099 webhook behaviour | **Open**, separate decision; not settled by ADR-0021; no webhooks in this contract |
| Maximum segments per message (D-144, D-105) | Product/provider policy; `422 message-too-long` when exceeded |
| Sender ID format and length per network | `TODO(telco-confirm)` (telco checklist §9) |
| D-095, D-097, D-098, ADR-0019 | Open; not modelled in this contract |

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
  `Idempotency-Key`, `202` for acceptance, no message text or provider identifiers in responses, the
  status table above matches the specification, no gateway or protocol terms, and no exactly-once claim.

CI runs the same commands (`.github/workflows/contracts-ci.yml`, job **OpenAPI contract check**).
TypeScript type generation (`openapi-typescript`, approved) is added with its first consumer, the
frontend (Group 5; numbered Group 4 before the owner's renumbering). **Go code** is generated from this
specification by `oapi-codegen` into `backend/internal/publicapi` (Phase 3 Group 4; owner decisions in
[decision log §9](../../docs/engineering/decision-log.md#9-owner-decisions--go-contract-integration-phase-3-group-4-2026-11-25);
how to regenerate: [`backend/README.md`](../../backend/README.md#openapi-code-generation-public-api-v1)).
A change to this file therefore also needs `go generate ./internal/publicapi/...` from `backend/`; the
backend CI fails otherwise. Go request **validation** against this specification is not chosen yet
(deferred to the first endpoint implementation, decision D6).
