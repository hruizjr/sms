# Traffic Evidence Worksheet

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Template — **no traffic data collected yet** | Project owner | 2026-11-25 (§5 revised) |

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
| Peak throughput established? | **No** |
