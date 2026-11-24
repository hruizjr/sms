# API Idempotency Contract (Proposed)

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — decision-ready proposal for ADR-0018; owner decisions pending (D-094) | Project owner | 2026-11-24 |

**This document is authoritative** for how idempotency keys work on our APIs. It is a **proposal**: each
section states the recommended rule and, where the owner must decide, the options. Nothing here is
implemented.

---

## 1. Purpose and limits

A client that sends "send this SMS" and gets no response (network drop, timeout) cannot tell whether we
received it. If it simply retries, it might create a second message — a second charge and a second SMS to
a real person. An **idempotency key** lets the client retry safely: we recognise the retry and return the
original result instead of acting again.

**What the key protects:** duplicate **operations inside our platform** — at most one accepted operation
(one set of message records, one credit reservation) per key, per scope, within the retention period.

**What the key does NOT protect:**

- **Duplicate submissions to a telco.** Those depend on provider behaviour and our handling of ambiguous
  outcomes — see [messaging reliability §4](messaging-reliability.md#4-provider-submission-and-ambiguous-outcomes).
  An idempotency key never implies exactly-once SMS delivery.
- **Client-side duplicates with different keys.** If a client sends the same content twice with two
  different keys, that is two operations by design. Optional content-based duplicate detection (same
  recipient and text within a time window) would be a separate product feature and owner decision.

## 2. Reference

The header name and status codes below align with the IETF Internet-Draft *The Idempotency-Key HTTP Header
Field* (draft-ietf-httpapi-idempotency-key-header, revision -09 consulted on 2026-11-24). It is a **draft,
not a published standard**; re-check its status when the contract is finalized. Where this proposal goes
beyond the draft, that is our own design choice.

## 3. Proposed contract

### 3.1 Which endpoints

| Rule | Recommendation | Decision needed |
| ---- | -------------- | --------------- |
| Endpoints covered | All operations that create messages, campaigns, or financial effects (message submission, bulk submission, campaign launch, and admin credit adjustments) | Confirm list during API design |
| Required or optional on message submission | **Required** in `/v1` | **Yes (D-094a).** Required from day one avoids a breaking change later; optional lowers integration friction |
| Missing key where required | `400 Bad Request`, problem type `idempotency-key-missing` (the draft recommends 400) | — |

### 3.2 Key format

- Header `Idempotency-Key`. The draft defines its value as a Structured Field String
  (i.e. a quoted string). Exact parsing rules are fixed when the OpenAPI contract is written.
- Recommended content: a randomly generated UUID. Maximum length to be set in the contract (e.g. a few
  hundred characters at most); keys outside the allowed format are rejected with `400`.
- Keys are opaque to us; we never derive meaning from them.

### 3.3 Scope of a key

| Option | Behaviour | Advantages | Disadvantages |
| ------ | --------- | ---------- | ------------- |
| A. Per **tenant + operation** | Same key from any of the tenant's API credentials for the same operation is the same request | A client that rotates or load-balances across several API keys still gets protection; matches "one business intent" | Two independent systems of the same tenant could collide if they generate poor keys (mitigated by requiring random keys and fingerprint checks) |
| B. Per **API credential + operation** | Each API key has its own key space | Strong separation between a tenant's systems | A retry sent with a rotated credential creates a duplicate operation |
| C. Per tenant, across all operations | One key space per tenant | Simple | A key reused across different operation types blocks unrelated requests |

**Recommendation: A (tenant + operation).** Portal-originated actions use the same contract: the frontend
generates one key per user intent (e.g. when the "Confirm send" dialog opens) and reuses it on retry.
**Decision: D-094b.**

### 3.4 Request fingerprint

- At first use, we store a **fingerprint** (cryptographic hash) of the canonicalized request: operation
  plus the parsed request body (so whitespace or JSON key order does not matter). The exact
  canonicalization is defined with the OpenAPI contract.
- Headers that do not change meaning (e.g. tracing IDs) are excluded.

### 3.5 Outcomes

| Situation | Response |
| --------- | -------- |
| First request with a new key | Processed normally; the outcome is recorded with the key (see §3.7 for which outcomes are recorded) |
| Same key, **same** fingerprint, original **completed** | The **original response is replayed**: same status code and body, including the same message IDs. A response header (name TBD, e.g. `Idempotent-Replayed: true`) indicates a replay. The replay reflects the result **at acceptance time**, not the current delivery status; clients query the message resource for current status |
| Same key, **different** fingerprint | `422 Unprocessable Content`, problem type `idempotency-key-reused`; nothing is processed (the draft recommends 422) |
| Same key while the original is **still in progress** | `409 Conflict`, problem type `idempotency-request-in-progress`, with `Retry-After` (the draft recommends 409) |
| Same key after the retention period expired | Treated as a new request (documented to clients) |

### 3.6 Concurrent duplicates

Two identical requests can arrive at the same moment (e.g. a client retrying aggressively). Proposed
handling:

1. **Single-transaction acceptance (preferred where possible).** The idempotency record (with a unique
   constraint on `tenant_id + operation + key`), the message records, the credit reservation, and the
   outbox event are written in **one database transaction**. The second concurrent request's insert waits
   for the first transaction and then fails the unique constraint; it then reads the stored outcome and
   replays it. A crash before commit leaves no record, so a client retry is processed cleanly.
2. **Long operations (e.g. large bulk submissions).** If acceptance cannot fit in one short transaction,
   the idempotency record is first committed as `in_progress` with a lease; concurrent duplicates receive
   `409`. If the worker dies, the lease expires and the defined recovery path applies (design detail for
   the bulk API).

Either way, concurrency is resolved by a **database unique constraint**, not by application-level checks
or a cache.

### 3.7 Which outcomes are recorded

| Outcome | Recommendation | Reason |
| ------- | -------------- | ------ |
| Success (`2xx`) | Recorded and replayed | Core purpose |
| Validation errors before any processing (`400`, `422` for invalid content) | Not recorded | Nothing happened; the client may fix and retry with the same key |
| Business rejections (e.g. insufficient credit) | **Decision D-094c.** Recommendation: not recorded, so a retry after top-up succeeds | Recording would replay a stale failure |
| Server errors (`5xx`) | Not recorded **only if** the transaction rolled back (nothing committed) | Safe to retry |

### 3.8 Retention

| Option | Trade-off |
| ------ | --------- |
| 24 hours | Small storage; protects normal automatic retries |
| 72 hours | Covers weekend outages of client systems |
| 7 days or more | Covers long-delayed replays; more storage at high volume (one record per request) |

**Recommendation:** at least 24 hours; final value depends on prospective clients' retry behaviour and
verified volume. **Decision: D-094d.** The chosen period is published in the API documentation (as the
draft advises).

### 3.9 Bulk requests

One key covers the whole bulk request. Whether a bulk request is all-or-nothing or returns per-recipient
results (some accepted, some rejected) is an API design decision (**D-094e**); the replay always returns
the original per-recipient result.

### 3.10 Interaction with credits

The credit reservation is created in the same transaction as the idempotency record (§3.6). A replay never
creates a second reservation. Ledger entries carry their own unique keys
([financial integrity](financial-integrity.md)), so even a bug in the idempotency layer cannot create
duplicate ledger entries for the same message.

## 4. Representative test scenarios (documentation, not code)

| # | Scenario | Expected result |
| - | -------- | --------------- |
| IK-1 | Same key + same body, sequential | One operation; second response is a replay with identical IDs; one reservation |
| IK-2 | Same key + different body | `422`; no new message; no reservation |
| IK-3 | Two identical requests concurrently | One operation; the other replays or receives `409`; one reservation |
| IK-4 | Crash before commit, then client retry with same key | Processed once |
| IK-5 | Same key from two different API credentials of the same tenant | Per chosen scope (§3.3) |
| IK-6 | Same key from two different tenants | Independent operations (keys never cross tenants) |
| IK-7 | Retry after the retention period | New operation (documented behaviour) |
| IK-8 | Missing key on a required endpoint | `400` |
| IK-9 | Validation failure, then corrected request with same key | Corrected request processed |
| IK-10 | Insufficient credit, top-up, retry with same key | Per D-094c |
| IK-11 | Replay after delivery status changed | Replay shows acceptance-time response; message resource shows current status |

## 5. Open decisions (recorded in the discovery register)

D-094a required vs optional · D-094b key scope · D-094c recording business rejections · D-094d retention ·
D-094e bulk semantics · D-095 ambiguous provider submissions (external duplicates, see messaging reliability).
