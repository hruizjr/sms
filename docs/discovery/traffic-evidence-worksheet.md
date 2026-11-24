# Traffic Evidence Worksheet

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Template — **no traffic data collected yet** | Project owner | 2026-11-24 |

Use this worksheet to collect evidence that tests the reported **~5 billion per month** figure (D-004) and
replaces it with verified demand. The questions and sizing formulas are defined in
[traffic capacity discovery](../requirements/traffic-capacity-discovery.md) (§3 questions, §4 formulas);
this worksheet captures the answers.

> **Rule:** no infrastructure sizing, capacity plan, or cost estimate may use the 5-billion figure until
> its unit and basis are established here with evidence.

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

## 5. Sanity check (context only, not evidence)

News reports found in a web search (secondary; not verified against NTC publications) put the number of
registered SIMs at roughly **114 million** when mandatory SIM registration closed in 2023, against about
**168 million** SIMs before registration (e.g. https://newsinfo.inquirer.net/?p=1810114, seen in search
results on 2026-11-24). If "5 billion" meant messages per month, that would be on the order of **40–45 SMS
per registered SIM per month** from A2P traffic routed through one aggregator. That is not impossible, but
it is a strong claim that needs strong evidence. Obtain current subscriber figures from official sources
before using them.

## 6. Status

| Question | Status |
| -------- | ------ |
| Unit of the 5-billion figure established? | **No** |
| Committed volume established? | **No** |
| Launch-customer volumes recorded? | **No** |
| Peak throughput established? | **No** |
