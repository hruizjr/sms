# Questionnaire — Intermediary Aggregator / Connectivity Provider

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| **Not applicable — third-party SMS aggregator/intermediary connectivity is not an acceptable operating model under D-160.** (2026-11-26, EX-9 review; owner-approved, pending partner ratification; [decision log §17](../../engineering/decision-log.md#17-owner-decisions--business-decisions-review-phase-2-ex-9-2026-11-26)). Kept for the record; not deleted | Project owner | 2026-11-26 |

**When to use:** if direct telco agreements are slow or unavailable, an existing aggregator or connectivity
provider might carry our traffic (interim or permanent). This changes our margin, control, regulatory
position, and technical design, so it is an **owner decision** (D-160), not a default.

Many technical questions match the [telco common core](telco-core.md) Part 2 (TC-20 to TC-31); ask those
too, because an intermediary's interface and limits may differ from the telcos'.

| # | Question | Register IDs | P |
| - | -------- | ------------ | - |
| IM-01 | Which networks do you reach, and is each route **direct** to the telco or through another party? | D-118, D-107, D-069 | P0 |
| IM-02 | Are you authorized by each telco to carry third-party aggregator traffic? Can you show evidence? | D-113, D-118 | P0 |
| IM-03 | Pricing, billing basis (submitted vs. delivered), minimum commitments, and rate precision. | D-108, D-041, D-117 | P0 |
| IM-04 | Throughput available to us, and how it relates to your own telco allocations. | D-101 | P0 |
| IM-05 | Delivery receipt pass-through: do we receive the telco's original receipt data or your normalized version? | D-104, D-114 | P1 |
| IM-06 | Sender ID registration: do you register on our behalf with each telco? | D-103 | P1 |
| IM-07 | Contract terms: exclusivity, termination notice, data ownership, and our ability to move to direct connections later. | D-160 | P0 |
| IM-08 | Data protection: where is message data processed and stored; what role do you take under the Data Privacy Act? | D-121, D-129 | P1 |

**Answers log:**

| Question | Evidence ID | Date | Summary (non-confidential) |
| -------- | ----------- | ---- | -------------------------- |
| — | — | — | *Not yet asked.* |
