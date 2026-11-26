# Traffic Evidence Worksheet

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Template — **no traffic data collected yet** (EX-6 deferred); **provisional design targets** recorded in §7 by owner decision (2026-11-26, EX-2) — not evidence | Project owner | 2026-11-26 |

Use this worksheet to collect evidence that tests the reported **~5 billion per month** figure (D-004) and
replaces it with verified demand. The questions and sizing formulas are defined in
[traffic capacity discovery](../requirements/traffic-capacity-discovery.md) (§3 questions, §4 formulas);
this worksheet captures the answers.

> **Rule:** no traffic model, infrastructure sizing, capacity plan, cost estimate, or financial model may
> use the 5-billion figure until its unit and commercial basis are established here with evidence.

## 1. What could "5 billion" mean?

| Interpretation | What it would imply | Evidence that would identify it |
| -------------- | ------------------- | ------------------------------- |
| SMS **API requests** per month | One request may carry many recipients, so message count could be much higher | The originator's model counts requests, not recipients |
| Individual **message recipients** (messages) | Segments ≥ messages; each multipart message multiplies telco submissions | Volume reports per recipient/message |
| **Billable segments** | Messages ≤ segments | Telco or provider invoices counted in segments |
| **Submission attempts** (including retries) | Overstates unique messages | Model includes retry assumptions |
| **Delivered** messages | Submitted volume is higher (some fail) | Reports of delivered counts |
| **Aggregate** across many customers or providers | Only part may come to us | List of customers/providers behind the figure |
| **Recipients reachable** (a contact base size) | Not a traffic figure at all | Figure equals a contact-list size |
| An **annual** figure stated as monthly | 12× lower monthly load | Source document shows a year |

## 2. Source of the figure

| Field | Answer | Evidence ID |
| ----- | ------ | ----------- |
| Who produced the figure | — | — |
| When | — | — |
| Method (bottom-up from customers? market estimate?) | — | — |
| Unit (from §1) | — | — |
| Committed portion (signed) | — | — |
| Estimated portion (customer stated) | — | — |
| Aspirational portion (market potential) | — | — |

## 3. Per-customer data

Use **pseudonyms** (CUST-A, CUST-B, …) in this file; keep the mapping to real names in the owner's secure
store. Record the unit for every number.

| Customer | Evidence ID | Status (committed / estimate / aspiration) | Use-case mix (OTP / transactional / promotional / other %) | Month 1 volume | Month 6 volume | Month 12 volume | Unit | Busiest-hour share or peak/sec | Largest campaign size and required completion time | Avg segments per message | UCS-2 share | Network split (Smart / Globe / DITO / unknown) | International? | MO / replies needed? | Receipt needs (speed, history) |
| -------- | ----------- | ------------------------------------------ | ---------------------------------------------------------- | -------------- | -------------- | --------------- | ---- | ------------------------------ | -------------------------------------------------- | ------------------------ | ----------- | ---------------------------------------------- | -------------- | -------------------- | ------------------------------ |
| — | — | — | — | — | — | — | — | — | — | — | — | — | — | — | *No data yet.* |

## 4. Aggregation (fill only from §3 rows with evidence)

| Measure | Committed | Estimated | Aspirational | Notes |
| ------- | --------- | --------- | ------------ | ----- |
| Messages per month at month 1 | — | — | — | |
| Messages per month at month 12 | — | — | — | |
| Segments per month at month 12 | — | — | — | messages × avg segments |
| Peak messages/second (combined, worst case) | — | — | — | not a simple sum: check whether peaks coincide |
| Delivery receipts per month | — | — | — | depends on telco receipt behaviour (D-104) |
| Share by network | — | — | — | compare with telco capacity (D-101) |

Sizing calculations, once inputs exist, use the formulas in
[traffic capacity discovery §4](../requirements/traffic-capacity-discovery.md#4-derived-values-to-be-computed-once-data-exists).

## 5. Context, evidence, and verified demand

This worksheet keeps three kinds of information strictly apart:

| Kind | What it is | Can it validate the 5-billion figure? |
| ---- | ---------- | ------------------------------------- |
| **Context** | Market background (e.g. national subscriber counts, industry reports, news) | **No.** Context may suggest questions to ask; it never confirms or rejects a traffic figure. |
| **Evidence** | Customer records, historical volume reports from current providers, signed contracts or letters of intent stating volumes, source-backed projections with a documented method | Yes, within the scope and date of each document, once logged in the [evidence register](evidence-register.md) |
| **Verified demand** | The total built only from evidence rows in §3, split into committed, estimated, and aspirational (§4) | This is the only basis for capacity planning, telco negotiations, and cost estimates |

Why subscriber and SIM counts are not used:

- Subscriber and SIM counts are **not evidence of demand** for our platform.
- Total mobile subscribers cannot establish the volume that any particular aggregator can acquire or route.
- The 5-billion monthly figure must be validated through **customer-level historical traffic, documented
  commitments, and source-backed projections** recorded in §2–§4.
- **No traffic, infrastructure, or financial model may rely on the figure** until its unit and commercial
  basis are established.

## 6. Status

| Question | Status |
| -------- | ------ |
| Unit of the 5-billion figure established? | **No** |
| Committed volume established? | **No** |
| Launch-customer volumes recorded? | **No** |
| Peak throughput established? | **No** (provisional design targets only, §7) |

## 7. Provisional design targets (owner decision 2026-11-26, EX-2)

On 2026-11-26 the owner decided to proceed with **provisional platform design targets** (decisions TR-1 to TR-9
and gate PG-10; [decision log §16](../engineering/decision-log.md#16-owner-decisions--provisional-traffic-targets-phase-2-ex-2-2026-11-26)).
This section is the authoritative record of those targets.

> **What these targets are not.** They are **not evidence, not a demand forecast and not telco capacity.** No
> customer data, commitment or traffic history stands behind them. Every distribution below is **synthetic or
> provisional**. §2 to §6 stay empty: **no traffic evidence has been collected** (EX-6 deferred). D-004 and D-060
> stay **open and unused**: the unverified monthly figure is not used by, compared with, or checked against
> these targets.

They are **platform capacity targets**: the workload our platform must accept, persist, queue, dispatch to the
simulator and process receipts for. They come from the approved MVP scope: single-message sending only through
the API and portal (MVP-07, MVP-09), with no bulk sending, campaigns or scheduling (MVP-08, MVP-12, MVP-13) and no
inbound messages (MVP-15), so peaks come from many independent single-message requests, not from campaign bursts.

### 7.1 Targets

| # | Target | Pilot | Initial (first 6 months of production) | Growth (12 months) | Basis and confidence |
| - | ------ | ----- | -------------------------------------- | ------------------ | -------------------- |
| TR-1 | Client tenants | 1–3 | up to 10 | up to 50 | Staff-controlled onboarding (MVP-01); at least one validated pilot customer (PG-8). Concentration: one tenant may generate up to **50 %** of traffic (sizes per-tenant limits, MVP-30). Pilot: medium (follows from owner decisions); others: low |
| TR-2 | Messages per day | 10,000 | 100,000 | 1,000,000 | Order-of-magnitude steps across a wide design range; no evidence. Low |
| TR-2 | Messages per month (30 days) | ~300,000 | ~3,000,000 | ~30,000,000 | As above. Low |
| TR-3 | Calculated peak submission rate | ~1.25 msg/s | ~12.5 msg/s | ~125 msg/s | Busiest hour carries 15 % of a day's messages; short bursts run at 3× the busiest-hour average (`0.15 × daily ÷ 3600 × 3`). Low |
| TR-3 | **Design peak submission rate**, sustained 15 minutes | **10 msg/s** | **50 msg/s** | **250 msg/s** | Headroom over the calculated rate: **8× pilot, 4× initial, 2× growth**. Low for the numbers; medium for the shape (no campaign bursts) |
| TR-3 | **Stress test** (tests only), sustained 15 minutes | — | — | **500 msg/s** | 2× the growth design rate; evidence for ADR-0006's reconsideration criteria without adopting new infrastructure |
| TR-4 | Average segments per message | 1.3 | 1.3 | 1.3 | Provisional: OTPs and alerts mostly single-segment. Low |
| TR-4 | UCS-2 share | 10 % | 10 % | 10 % | Provisional. Low |
| TR-4 | Peak segment rate | 1.5 × submission rate | 1.5 × | 1.5 × | Provisional, conservative. Segments are counted by the platform's own calculation (MVP-10); a telco may count differently (D-117, `TODO(telco-confirm)`). Low |

### 7.2 Test workloads

Load and simulator tests define **three separate workloads**. A test states which it drives and reports each
one separately; a target met for one is not evidence for another.

| Workload | What is counted | Pilot | Initial | Growth | Stress |
| -------- | --------------- | ----- | ------- | ------ | ------ |
| **Message submissions** | Single-message requests accepted by the API or portal (one recipient each) and dispatched to the simulator | 10/s | 50/s | 250/s | 500/s |
| **Segments** | Segments submitted to the simulator (submission rate × 1.5 at peak) | 15/s | 75/s | 375/s | 750/s |
| **Receipt events** | Simulated delivery-receipt events processed: at most one per segment, plus up to 10 % duplicates (submission segments × 1.1) | 16.5/s | 82.5/s | 412.5/s | 825/s |

Receipt-event rates are a **synthetic upper bound**. Real receipt behaviour (per segment or per message, timing,
duplicates) is unknown (D-104) and is not assumed. Delayed, out-of-order and missing receipts are exercised as the
simulator failure scenarios (S3), not as extra rate. Inbound (MO) events: **zero** in the MVP (MVP-15).

### 7.3 Workload distributions (synthetic or provisional)

| Dimension | Distribution used in tests | Status |
| --------- | -------------------------- | ------ |
| Use case | OTP 60 %, transactional alerts 35 %, other single messages 5 %; no campaigns | Provisional (D-061 open) |
| Channel | API 90 %, portal 10 % | Provisional (portal sends are manual, one recipient each) |
| Time of day | 80 % between 07:00 and 22:00 Asia/Manila; busiest hour 15 % of the day | Provisional |
| Destination network | Synthetic split across two simulated providers (for example 50/50), labelled synthetic | **Synthetic.** No network-split target; the real split and number portability are unknown (D-070) |
| Destination country | Philippines only | Provisional (assumption D-043) |
| Provider throughput limits in tests | Synthetic limits for the simulator's throttling scenario, labelled synthetic | **Synthetic** (S2); imply nothing about any telco |

### 7.4 Platform capacity versus telco throughput

| | Platform capacity target (this section) | Telco throughput limit |
| -- | --------------------------------------- | ---------------------- |
| What it is | The rate the platform accepts, persists, queues and processes | The rate a telco accepts, per account, connection, sender ID or traffic type |
| Source | Owner decision (provisional) | Signed agreement and official documentation only (D-101; telco checklist §7, §8) |
| Today | §7.1 and §7.2 | **Unknown**; nothing is assumed |
| In production | Dispatch runs at the **lower** of the platform rate and each telco's contracted limit; excess is queued with back-pressure (AGENTS.md §11.3, §11.4), never silently dropped | Enforced centrally per provider; never exceeded |

Meeting a platform target says nothing about the throughput any telco will provide.

### 7.5 Permitted uses

| Allowed | Not allowed |
| ------- | ----------- |
| Load-test and simulator scenario design; evaluating the PostgreSQL-backed queue and partitioning (ADR-0006, ADR-0008 criteria); configured worker counts and back-pressure settings; placeholder alert thresholds (MVP-28); sizing development and test environments | Telco capacity requests or negotiations; customer commitments or SLAs; financial models, pricing or cost estimates; production infrastructure sizing before the PG-10 re-baseline; any claim of verified demand |

### 7.6 Revision triggers and gate

The targets are re-baselined when any of these happens: a pilot customer provides volume information; measured
traffic exceeds 50 % of the current scenario's target; the unit and basis of D-004/D-060 are established with
evidence; bulk sending, campaigns or scheduling are approved for a release (MVP-08, MVP-12, MVP-13); contracted
telco throughput becomes known (D-101).

**Gate PG-10:** before general production rollout, the targets are **re-baselined** against validated
pilot-customer volume information (PG-8) and contracted telco limits (PG-1). **Production infrastructure sizing
requires this re-baseline.**

Still dependent: launch customers and their volumes (D-061, D-074); telco throughput and segment-counting basis
(D-101, D-117); an OTP latency objective (a separate service-level decision, needed before ADR-0006's queue-latency
criterion can be applied); storage sizing (record size and retention periods, D-073, D-124).
