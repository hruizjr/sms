# Financial Integrity: Ledger Rules, Monetary Precision, and Rounding

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — rules proposed; representation and scale **open** (ADR-0011, ADR-0019) | Project owner | 2026-11-24 |

**This document is authoritative** for how money and credits are represented, calculated, rounded, and
recorded. `AGENTS.md` summarizes the rules and links here.

---

## 1. Non-negotiable rules

| # | Rule |
| - | ---- |
| F1 | **Never use IEEE floating-point types** (`float32`, `float64`, PostgreSQL `real`/`double precision`, JavaScript arithmetic on `Number`) for monetary amounts, rates, or credit calculations. |
| F2 | **All ledger movements use exact, deterministic arithmetic.** The same inputs always produce the same result on every machine. |
| F3 | **Currency and monetary scale are explicitly documented** for every amount and rate field (in the schema, the API contract, and code types). No field may hold an amount of unstated scale. |
| F4 | **Rate precision is validated against actual commercial pricing** (telco wholesale rates and our retail price lists) before the representation is finalized. |
| F5 | **Rounding rules are explicit, deterministic, and auditable**: where rounding happens, which mode, and what inputs were used are recorded or derivable. |
| F6 | **Charges are based on billable SMS segments**, not API requests. One request may create many messages (recipients); each message may have several segments. |
| F7 | **Reservations, releases, settlements, refunds, and reversals preserve ledger integrity**: each is its own entry, idempotent, and linked to its cause. |
| F8 | **Historical ledger entries are never rewritten or deleted.** Corrections are new compensating entries that reference the original. |

## 2. What is still unknown (and why we do not choose a representation yet)

| Question | Discovery item |
| -------- | -------------- |
| Is the customer-facing unit pesos or "SMS credits"? | D-081 |
| Currency (assumed PHP, not confirmed for all cases, e.g. foreign clients or USD-denominated wholesale terms) | D-092 |
| How many decimal places do telco wholesale rates and our retail rates actually use? | D-092, telco checklist §15 |
| Where is rounding applied (per segment, per message, per campaign, per invoice) and with which mode? | D-093 |
| Are amounts recorded gross or net of taxes? | D-127 |

**Do not assume that whole pesos or centavos are sufficient.** A per-segment rate might be quoted with more
precision than a centavo. Illustration (hypothetical numbers, not real rates): if a rate had four decimal
places and each segment's charge were rounded to the nearest centavo, each segment could be off by up to
half a centavo — up to ₱5,000 across one million segments. Small per-unit precision choices become material
at volume.

## 3. Representation options (decision deferred to ADR-0019)

| Option | Description | Advantages | Disadvantages |
| ------ | ----------- | ---------- | ------------- |
| **A. Integer in centavos** (scale 2) | `bigint` / Go `int64` counting centavos | Simple; fast; familiar | **Insufficient** if any rate or intermediate value needs more than two decimal places |
| **B. Integer in a finer fixed unit** (e.g. 10⁻⁴ or 10⁻⁶ peso) | `bigint` counting a documented sub-unit | Exact; fast; native Go and SQL types; large headroom (an `int64` at 10⁻⁶ scale still holds amounts in the trillions of pesos) | Scale must be chosen to cover every rate and intermediate value; changing it later requires a data migration; every conversion to display units must be explicit |
| **C. Exact decimal** | PostgreSQL `numeric(p, s)` with an exact decimal type in Go | Flexible precision; reads naturally; scale can differ by field | Go's standard library has **no decimal type** — needs a third-party library (dependency approval) or careful use of `math/big`; slower than integers; driver mapping needs care |
| **D. Hybrid** | High-precision decimal (C) for **rates**; scaled integer (B) for **ledger amounts**, with documented rounding at the point a charge is computed | Rates keep full precision; ledger stays fast and simple | Two representations; the conversion and rounding point must be rigorously defined and tested |

Evaluation criteria for the decision: maximum precision of real rates; rounding point and mode; volume and
performance (ledger writes on the hot path); dependency footprint; overflow headroom; reporting and
reconciliation with telco invoices; clarity for future developers.

**No option is selected.** The decision is made when actual telco rates and the commercial pricing model are
available (Phase 2).

### 3.1 API and frontend representation

- JavaScript `Number` is an IEEE double: it cannot represent most decimal fractions exactly, and integers
  above 2⁵³ − 1 lose precision. **Monetary amounts in API payloads should be transmitted as strings**
  (decimal strings or integer-unit strings, per ADR-0019) — proposed in ADR-0015/ADR-0019.
- The frontend only **formats** monetary values for display; it never performs monetary arithmetic. Cost
  estimates are calculated by the server.

## 4. Rounding (to be decided — D-093)

The decision must specify:

1. **Where** rounding occurs: per segment, per message, per request/campaign, per ledger entry, or per invoice.
2. **Mode**: e.g. half-up, half-even (banker's rounding), always up, always down — and who bears the
   difference.
3. **Residue handling**: whether accumulated rounding differences are tracked.
4. **Auditability**: each charge entry stores or references the inputs needed to recompute it — rate
   (and rate version), billable segment count, scale, rounding mode.

Rounding is implemented in **one shared function** used by cost estimates, reservations, and settlements,
so the estimate the customer sees and the actual charge cannot diverge through different arithmetic.

## 5. Segment-based billing

- Billable segments are computed by **one shared segmentation function** (encoding detection, GSM-7
  extended characters, UCS-2, concatenation overhead) used for estimate, reservation, and charge.
- The telco's own segment count may differ (e.g. different concatenation handling). Differences are
  detected by **reconciliation** with telco records, not by silently changing past charges.
- Segmentation rules per provider must come from telco documentation (telco checklist §10).

## 6. Ledger operations (ADR-0011)

| Entry | Effect | Idempotency key (proposed) |
| ----- | ------ | -------------------------- |
| `top_up` | Increases available balance | Payment/top-up reference |
| `reservation` | Moves amount from available to reserved | `(message_id, reservation)` |
| `reservation_release` | Returns unused reserved amount | `(message_id, release)` |
| `charge` (settlement) | Converts reserved amount to charged | `(message_id, charge)` |
| `refund` | Returns a charged amount | `(message_id, refund)` or refund reference |
| `reversal` | Compensates a specific earlier entry (error correction) | Reference to the reversed entry |
| `adjustment` | Admin correction with mandatory reason, actor, and audit record | Adjustment request ID |

Rules:

- A settlement that differs from its reservation produces explicit entries (charge the actual amount,
  release the remainder) — never an edit of the reservation.
- Unique constraints on idempotency keys make duplicate entries impossible even under duplicate events or
  receipts ([messaging reliability §5.5](messaging-reliability.md#55-duplicate-and-out-of-order-receipts)).
- Balance invariants (e.g. available balance never below the permitted limit) are enforced by the
  database inside the same transaction as the entry.
- Any cached balance is verifiable against the sum of ledger entries; a periodic job checks this and alerts.
- Charges reference the **rate version** in effect; rate changes create new versions, never edits.
- Wholesale (telco) cost tracking for margin and reconciliation is a separate concern from the customer
  ledger; its design is part of the architecture phase.

## 7. Representative test scenarios (documentation, not code)

| # | Scenario | Expected result |
| - | -------- | --------------- |
| FI-1 | Many concurrent reservations against one balance | No overspend; balance equals ledger sum |
| FI-2 | Duplicate settlement event | One `charge` entry |
| FI-3 | Duplicate final receipt after settlement | No second charge or refund |
| FI-4 | Late receipt after release/refund | New compensating entries per billing rules; earlier entries unchanged |
| FI-5 | Rounding at scale boundaries (values just above/below half) | Matches the documented mode exactly |
| FI-6 | Estimate vs. actual charge for the same message | Identical when inputs are identical |
| FI-7 | UCS-2 vs GSM-7 content of the same length | Different segment counts; charge follows segments |
| FI-8 | Admin adjustment without reason | Rejected |
| FI-9 | Attempted update or delete of a ledger entry by the application role | Denied by the database |
| FI-10 | Rate change mid-campaign | Each charge references the rate version in effect per the documented rule |
