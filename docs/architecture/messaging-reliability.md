# Messaging Reliability: Event Delivery, Message Lifecycle, and Delivery Receipts

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — pending owner review (supports ADR-0020, ADR-0021); ADR-0021 accepted with limited scope 2026-11-25 (customer-visible statuses only) | Project owner | 2026-11-25 |

**This document is authoritative** for internal event-delivery guarantees, the transactional outbox,
message lifecycle milestones, provider submission outcomes, and delivery receipt (DLR) handling, as
defined in [`AGENTS.md` §1.1](../../AGENTS.md#11-document-authority-and-conflicts). `AGENTS.md` §11 summarizes these rules and links here; other documents must reference
this one rather than restate it. Items tied to Proposed ADRs or open discovery items remain proposals.
It does not grant permissions, widen task scope, or override restrictions in `AGENTS.md`. If it genuinely conflicts with another document, stop the affected work, report the conflict, and request owner resolution ([`AGENTS.md` §1.1](../../AGENTS.md#11-document-authority-and-conflicts)).

Nothing in this document describes the behaviour of a specific telco. Provider behaviour must come from
official telco documentation (see the [telco discovery checklist](../integrations/telco-discovery-checklist.md)).
ADR-0021 was accepted by the owner on 2026-11-25 **with a limited scope**: the eight customer-visible
statuses and the initial reason-code vocabulary in the
[public API contract §3](../../contracts/openapi/README.md#3-status-reference-customer-visible-statuses).
The **internal** state names below remain **proposed** and illustrative; they are the model those customer
statuses map from. The billing treatment of `UNKNOWN` and other uncertain messages (D-096) and webhook
behaviour (D-099) remain separate open decisions.

---

## 1. Guarantee vocabulary

| Term | Meaning in this project |
| ---- | ----------------------- |
| **At-most-once** | Something happens zero or one times. Duplicates are impossible, but loss is possible. |
| **At-least-once** | Something happens one or more times. Loss is prevented (within the system's durability limits), but duplicates are possible. |
| **Idempotent processing** | Handling the same input again produces no additional business effect. |
| **Exactly-once delivery** | Something is transmitted exactly one time. **Not achievable across independent systems** (our database ↔ broker ↔ telco ↔ handset) when any link can time out or crash. We do not promise it. |

**Baseline for this platform:** internal events are delivered **at least once**, and every consumer is
**idempotent**, so that duplicate deliveries do not produce duplicate business effects **inside our own
database**. A component may use different semantics only with an ADR that explains the technical
justification and its limits.

---

## 2. Internal events and the transactional outbox

### 2.1 What the outbox guarantees

The transactional outbox pattern writes an *outbox record* (the event to be published) in the **same
database transaction** as the business change it describes. A separate *publisher* later reads committed
outbox records and publishes them.

| # | Guarantee |
| - | --------- |
| G1 | The business change and its outbox record are committed **atomically**: both or neither. |
| G2 | If the business transaction **rolls back**, its outbox record does not exist, so the event is **never published**. |
| G3 | Once committed, the outbox record is **durable** and will eventually be published as long as the publisher eventually runs and the destination eventually accepts it. |

### 2.2 What the outbox does NOT guarantee

| # | Non-guarantee | Why |
| - | ------------- | --- |
| N1 | **Single publication.** The same event may be published more than once. | The publisher can crash after publishing but before marking the record as published; a publish acknowledgment can time out even though the destination received the event; retries after errors republish. |
| N2 | **Immediate publication.** | Publishing happens after commit, with some delay, possibly long during outages. |
| N3 | **Global ordering.** | Multiple publishers, retries, and partitions can reorder events unless an ordering scheme is explicitly designed (§2.5). |

### 2.3 Publisher requirements

- Every event receives a **stable event ID** when its outbox record is written. Republishing reuses the
  same ID — this is what lets consumers detect duplicates.
- A record is marked as published **only after** the destination has acknowledged it. A missing or
  ambiguous acknowledgment is treated as "not published" and retried — accepting duplicates (N1) to avoid loss.
- Retries use backoff; publisher lag (age of the oldest unpublished record) is a monitored metric with an alert.
- Multiple publisher instances must not cause unbounded duplicate publishing (e.g. claim records with row
  locks or leases). Some duplication remains possible and is acceptable.
- Published outbox records are cleaned up on a schedule; retention is a design parameter, not unbounded.

### 2.4 Consumer requirements (duplicate detection and handling)

Every consumer MUST be idempotent. The consumer's design document or code comment MUST state which
mechanism it uses:

| Mechanism | How it works | Use when |
| --------- | ------------ | -------- |
| **Processed-event record** | Insert `(consumer_name, event_id)` into a table with a unique constraint **in the same transaction** as the consumer's own database changes. A unique-constraint conflict means "already processed" → skip. | General default for consumers that change our database. |
| **Conditional state transition** | Apply a change only if the current state allows it (e.g. `UPDATE ... WHERE status = 'SUBMITTED'`); zero rows affected means already applied or superseded. | State machines (message status). |
| **Unique business key** | The derived record has a natural unique key (e.g. one ledger entry per `(message_id, entry_type)`); a duplicate insert fails harmlessly. | Ledger entries, receipts, billing effects. |

Rules:

- Dedupe records must be retained for at least the **maximum redelivery window** of the event source.
  That window is a design parameter to be set when the queue/broker is chosen (ADR-0006); it is not set here.
- **Side effects outside our database** (sending a webhook, submitting to a telco, sending an email)
  cannot be committed atomically with a dedupe record. Each such effect needs its own documented policy:
  - **Customer webhooks** are documented to customers as **at-least-once**, carrying the event ID so
    customers can deduplicate.
  - **Telco submissions** follow §4 (ambiguous outcomes); they are never blindly repeated.
- A consumer that fails mid-processing must leave no partial committed effects (one transaction), so
  redelivery can safely retry it.
- Poison events (that always fail) go to a dead-letter store after bounded attempts, with the error, and
  are visible to operators.

### 2.5 Ordering

- **No global ordering is guaranteed.** Designs SHOULD make correctness independent of order wherever
  possible (e.g. monotonic state machines, §5.5).
- Where business correctness depends on order, the component MUST document:
  1. the ordering scope (e.g. "per message", "per ledger account");
  2. the mechanism (e.g. events for one aggregate share a partition/ordering key, or carry a per-aggregate
     version number and consumers reject stale versions);
  3. what happens when an out-of-order event arrives (reject, buffer, or apply precedence rules).
- Known ordering-sensitive areas (to be designed in the architecture phase):

| Area | Ordering need | Proposed approach |
| ---- | ------------- | ----------------- |
| Message status changes | Per message | Precedence rules (§5.5), not arrival order |
| Ledger entries | Per account | Enforced by database transactions and constraints, not event order |
| Campaign lifecycle (scheduled → running → paused → completed) | Per campaign | Conditional state transitions |
| Configuration changes consumed by the data plane | Per configuration item | Versioned configuration; consumers ignore older versions |

---

## 3. Message lifecycle milestones

Each milestone proves something different. **None of the earlier milestones proves delivery to a handset.**

| # | Milestone | What it proves | What it does NOT prove |
| - | --------- | -------------- | ---------------------- |
| M1 | **Accepted by our API** — request validated, message persisted, credit reserved, message ID returned | We own the message and will attempt to process it | That it has been queued, sent, or delivered |
| M2 | **Accepted by our internal queue** — dispatch event committed (outbox) | Dispatch will be attempted (at least once) | That a telco has seen it |
| M3 | **Submission attempted** — request sent to a provider | We tried | That the provider received or accepted it |
| M4 | **Provider acknowledgment** — provider explicitly accepted (e.g. a success response with the provider's message ID) or explicitly rejected | The provider took responsibility for attempting delivery (or refused) | That the handset received it |
| M5 | **Final delivery information** — a receipt reporting a final state, *if the provider supplies one* | The provider's report of the outcome | Absolute truth: receipts can be late, missing, duplicated, conflicting, or wrong |
| M6 | **Customer-visible status** — what the portal, API, and webhooks show | Our best current classification | It is a mapping of internal states; it may be coarser and may change if late information arrives |

### 3.1 Proposed internal states (ADR-0021)

| State | Kind | Meaning |
| ----- | ---- | ------- |
| `ACCEPTED` | Transitional | M1 reached |
| `QUEUED` | Transitional | M2 reached |
| `SUBMITTING` | Transitional | M3 in progress; an attempt record exists |
| `SUBMIT_UNCERTAIN` | **Uncertain** | Submission outcome unknown (timeout, crash after sending, ambiguous response) — see §4 |
| `SUBMITTED` | Transitional | M4: provider accepted |
| `REJECTED` | Final | Rejected by us (validation, suppression, policy) or explicitly by the provider before acceptance |
| `FAILED` | Final | Permanent failure or retries exhausted **before** provider acceptance |
| `DELIVERED` | Final (provider-reported) | Receipt reports delivery |
| `UNDELIVERED` | Final (provider-reported) | Receipt reports a final non-delivery state (including provider-reported expiry) |
| `UNKNOWN` | **Uncertain** | No final information within the configured window — **not** evidence of non-delivery (§5.3) |

"Final (provider-reported)" states can still be flagged by reconciliation if conflicting information
arrives (§5.6). `UNKNOWN` and `SUBMIT_UNCERTAIN` remain eligible for update by late information.

---

## 4. Provider submission and ambiguous outcomes

Internal idempotency (ADR-0018, [API idempotency](api-idempotency.md)) prevents **duplicate operations
inside our platform** — one message record, one reservation per request. It does **not**, by itself,
prevent **duplicate submissions to a provider**. Those can arise from:

- a submission timeout followed by a retry (the first attempt may have succeeded);
- a crash after sending but before recording the provider's acknowledgment;
- redelivery of the dispatch event (at-least-once, §2) after a crash.

Requirements:

1. Before sending, the dispatcher records an **attempt** (`SUBMITTING`, with an attempt ID). A redelivered
   dispatch event that finds an attempt in progress or completed does **not** submit again.
2. On restart, messages left in `SUBMITTING` without a recorded acknowledgment move to `SUBMIT_UNCERTAIN`.
   They are **not automatically resubmitted**.
3. Every provider response is classified as success, retryable, permanent, or **ambiguous**. A timeout
   with no response is ambiguous. Classification tables live in each provider adapter, sourced from
   official telco documentation.
4. Only **retryable** outcomes are retried automatically (backoff with jitter, attempt limit, maximum message age).
5. Resolving `SUBMIT_UNCERTAIN` follows a **per-provider, per-traffic-type policy** that is an owner
   decision (discovery register D-095). Options include:
   - never resubmit (risk: message lost);
   - resubmit only after confirming non-acceptance via a provider status-query capability, if one exists (D-115);
   - wait for a receipt referencing the attempt, then decide;
   - resubmit for selected traffic types where a duplicate is judged less harmful than loss.
6. Provider-side duplicate detection, if a provider offers it, is a **provider capability** that must be
   confirmed in writing (telco checklist §3, §14). Until confirmed, assume none.
7. The credit reservation for an uncertain message is held until the uncertainty is resolved or the
   billing policy decides otherwise (D-096).

**No document, code comment, API description, or customer communication may claim exactly-once SMS
delivery** unless a provider-specific, documented guarantee exists and an ADR records it.

---

## 5. Delivery receipt (DLR) handling

### 5.1 Adapter boundary

- Provider adapters parse provider-specific receipt formats and map them to a **normalized receipt**:
  normalized status, provider status/error codes (kept as opaque values), provider message ID, provider
  timestamps, segment information if available, and a reference to the stored raw receipt.
- **Provider-specific formats and status mappings stay inside adapters.** Core messaging logic never
  branches on a telco name or interprets raw provider codes.
- Each adapter's status-mapping table is documented with its source (telco document and version).

### 5.2 Auditable history

- Every received receipt is stored **as received** (append-only), before or independently of whether it
  can be correlated. Some receipt formats may include a fragment of message text; raw receipts are
  therefore personal data subject to access control and retention rules.
- Every message status transition is recorded in an **append-only status history** with: previous state,
  new state, source (API, dispatcher, provider acknowledgment, receipt, reconciliation, operator),
  reference to the triggering receipt or event, and both our receive time and any provider-reported time.
- Operator corrections are new history entries with actor and reason, never edits.

### 5.3 Missing receipts and the `UNKNOWN` state

- A message with no final receipt within a **configured window** moves to `UNKNOWN`.
- `UNKNOWN` **does not mean undelivered.** Not all providers or routes may return receipts at all.
- The window is **configurable per provider** (and possibly per traffic type). **There is no universal
  default.** Initial values are set from each provider's documentation and observed behaviour (D-116).
- Customer-facing labels were decided by the owner on 2026-11-25 (D-096, D-099): `UNKNOWN` is shown as
  `unknown` and `SUBMIT_UNCERTAIN` as `submission_uncertain` (public API contract §3). Billing treatment
  of `UNKNOWN` remains a pending owner decision (D-096).

### 5.4 Late receipts

- A receipt arriving after a message moved to `UNKNOWN` (or `SUBMIT_UNCERTAIN`) is still correlated and
  applied according to precedence rules, and is eligible for reconciliation.
- If billing already acted on the earlier classification (e.g. a release or refund), the late receipt
  triggers a **new, explicit** ledger entry under the billing rules — never an edit of the earlier entry.
  Whether late receipts can re-charge a customer after a refund is a business decision (D-096).

### 5.5 Duplicate and out-of-order receipts

- **Duplicates:** a receipt identical to one already processed (identity key defined per adapter) is
  stored for audit but causes no further state change.
- **Side effects are keyed by `(message_id, effect_type)`** — settlement, release, refund, customer
  webhook event — so duplicate receipts can never cause duplicate billing, duplicate refunds, or repeated
  business side effects.
- **Out-of-order:** state changes follow **precedence**, not arrival order:
  - a final state is never replaced by a transitional state (e.g. a late "accepted/enroute"-type receipt
    after `DELIVERED` is recorded in history but does not change the state);
  - `UNKNOWN` / `SUBMIT_UNCERTAIN` may be replaced by a provider-reported final state;
  - a receipt that arrives **before** our record of the provider acknowledgment is held as unmatched and
    re-correlated when the acknowledgment is recorded.

### 5.6 Conflicting final receipts

If a provider reports two different final states for the same message (or segment), the system MUST NOT
silently overwrite. It records both, keeps the current state, and flags the message for reconciliation.
The resolution policy is an owner decision (D-098).

### 5.7 Multi-segment messages

Receipts may be per segment or per message depending on the provider (D-104). How segment-level
outcomes roll up to a message-level status, and how partially delivered messages are billed, are pending
decisions (D-097).

### 5.8 Unmatched receipts

Receipts that cannot be correlated (unknown provider message ID, ID reuse, parsing problems) are retained
and surfaced for reconciliation. Whether provider message IDs are unique and how long before reuse must be
confirmed per provider (D-114).

---

## 6. Representative test scenarios (documentation, not code)

These scenarios are required test cases for the relevant components. See also
[testing standards](../standards/testing-standards.md).

### 6.1 Outbox and event delivery

| # | Scenario | Expected result |
| - | -------- | --------------- |
| OB-1 | Business transaction rolls back after the outbox insert | No outbox record; no event published |
| OB-2 | Publisher crashes before publishing a committed record | Event is published after restart |
| OB-3 | Publisher crashes after publishing but before marking the record published | Event is published again with the **same event ID** |
| OB-4 | Destination acknowledgment times out | Record remains unpublished and is retried |
| OB-5 | Consumer receives the same event twice (sequentially) | Business effect applied once; second delivery skipped and counted in a metric |
| OB-6 | Two consumer instances receive the same event concurrently | Exactly one applies the effect (unique constraint/conditional update); the other skips without error |
| OB-7 | Consumer crashes mid-processing | No partial effects committed; redelivery applies the effect once |
| OB-8 | Events for one message arrive out of order | Final state follows precedence rules, not arrival order |
| OB-9 | Poison event | Moves to dead-letter store after bounded attempts; operator-visible |

### 6.2 Provider submission

| # | Scenario | Expected result |
| - | -------- | --------------- |
| PS-1 | Provider returns success | `SUBMITTED`; provider message ID stored |
| PS-2 | Provider returns a retryable error | Retried with backoff within attempt and age limits |
| PS-3 | Provider returns a permanent error | `FAILED`; reservation handled per billing rules; no retry |
| PS-4 | Submission times out with no response | `SUBMIT_UNCERTAIN`; **not** automatically resubmitted |
| PS-5 | Dispatcher crashes after sending, before recording the acknowledgment | On restart: `SUBMIT_UNCERTAIN`; no automatic resubmission |
| PS-6 | Dispatch event redelivered while an attempt is in progress | No second submission |
| PS-7 | Message exceeds maximum age before submission | Not submitted; final state and billing per rules |

### 6.3 Delivery receipts

| # | Scenario | Expected result |
| - | -------- | --------------- |
| DR-1 | Final receipt arrives within the window | Final state; history entry; billing effect once |
| DR-2 | No receipt within the window | `UNKNOWN`; no assumption of non-delivery |
| DR-3 | Late receipt after `UNKNOWN` | State updated per precedence; history kept; billing adjusted by new entries only |
| DR-4 | Duplicate receipt | Stored; no state change; no duplicate billing, refund, or webhook |
| DR-5 | Transitional receipt arrives after a final receipt | History records it; state unchanged |
| DR-6 | Receipt arrives before the provider acknowledgment is recorded | Held unmatched; correlated once the acknowledgment is recorded |
| DR-7 | Receipt with unknown provider message ID | Stored as unmatched; surfaced for reconciliation |
| DR-8 | Two conflicting final receipts | Both recorded; state not silently overwritten; reconciliation flag raised |
| DR-9 | Per-segment receipts for a multi-segment message | Roll-up per the decided rule (D-097); test pending decision |
| DR-10 | Malformed receipt | Stored raw; parse failure recorded; no crash; alert if rate exceeds threshold |

---

## 7. Pending decisions referenced by this document

| ID | Decision |
| -- | -------- |
| D-080 | Charge on submission vs. delivery |
| D-095 | Policy for ambiguous provider submissions, per provider and traffic type |
| D-096 | Billing and customer treatment of `UNKNOWN`/uncertain messages; effect of late receipts after refund |
| D-097 | Multi-segment receipt roll-up and partial-delivery billing |
| D-098 | Conflicting final receipts resolution |
| D-099 | Customer-visible status set and webhook semantics |
| D-104, D-114, D-115, D-116 | Provider receipt behaviour, message ID uniqueness, status-query/dedupe capabilities, receipt timing |
