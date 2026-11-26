# Traffic and Capacity Discovery

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — awaiting data (no evidence collected); provisional design targets recorded by owner decision 2026-11-26 (§7) | Project owner | 2026-11-26 |

## 1. Purpose

The project owner has reported a projected traffic figure of **approximately 5 billion per month**.
This figure is **not verified** and its **unit is unknown**. This document defines what we need to learn
before any infrastructure sizing, telco capacity negotiation, or cost estimate is made.

> **Rule:** No design, capacity plan, or cost estimate may use the 5-billion figure until the questions in
> §3 are answered and the answers are recorded in §5 with their source.

## 2. Why the unit matters

The same number can mean very different system loads:

| If "5 billion" means... | What the system actually processes |
| ----------------------- | ---------------------------------- |
| SMS messages submitted per month | 5B messages, **plus** extra segments for long messages, **plus** delivery receipts |
| Billable segments per month | Fewer messages than 5B (each message ≥ 1 segment) |
| Potential recipients / mobile numbers | Not a traffic figure at all; actual sends depend on frequency per recipient |
| Combined projected traffic of prospective clients | Upper bound; real traffic arrives gradually and partially |
| Existing committed traffic | Firm requirement — but needs contracts or historical data |
| Delivery attempts (incl. retries) | Overstates unique messages |
| Annual figure mistakenly stated as monthly | 12× smaller load |

### Illustrative arithmetic (NOT a sizing)

To show why the details matter, here is the average rate **if** 5 billion single-segment messages were
spread perfectly evenly over a 30-day month:

```
5,000,000,000 ÷ (30 days × 86,400 s/day) ≈ 1,929 messages per second, around the clock
```

Real traffic is never evenly spread:

- If most traffic happens in a 12-hour business window, the average in that window is ~3,858/s.
- Campaign bursts (e.g. a retailer sending to millions at 10:00) can be many times the average.
- Every message may produce a delivery receipt — roughly doubling events processed.
- A 2-segment message doubles telco submissions.

Each of these multipliers is unknown today. A plausible-sounding guess could be off by more than 10×
in either direction, which is why we do not size anything yet.

As a sanity check, compare any monthly figure against the number of active mobile subscribers in the
Philippines (figure to be obtained from NTC or telco sources) — the implied messages per subscriber per
month should be plausible.

## 3. Questions to answer

### 3.1 Definition and source
1. What exactly does the figure count: messages, segments, recipients, delivery attempts, or something else?
2. Is it per month or per year?
3. Who produced the figure, how, and when? Is there a spreadsheet or source document?
4. How much is **existing committed** traffic (signed customers, letters of intent) vs. **prospective**?
5. Is any of this traffic currently sent through another aggregator? If so, can historical volume
   reports (by month, day, hour) be obtained?

### 3.2 Volume per client
6. Which clients or client segments make up the figure? Volume per client?
7. When would each client start sending (ramp-up schedule)?
8. Which use cases: OTP, transactional alerts, marketing, government advisories? Share of each?

### 3.3 Volume per network
9. Expected split by destination network (Smart, Globe, DITO, others)?
10. Is the split based on data or an assumption (number portability makes prefixes unreliable)?

### 3.4 Average and peak throughput
11. Expected daily volume pattern: what share of traffic falls in the busiest hour?
12. Peak messages per second required for the largest single client or campaign?
13. Latency requirement for OTPs (e.g. "submitted to telco within N seconds")?
14. Seasonal peaks (paydays, holidays, sale events like 11.11/12.12, emergencies, elections)?

### 3.5 Campaign burst patterns
15. Largest single campaign size (recipients)?
16. Required completion time for a large campaign (e.g. 1 million recipients in 30 minutes)?
17. Are campaigns scheduled at round times (many clients at 09:00 or 12:00)?
18. Can campaigns be paced (throttled over time), or must they go out as fast as possible?

### 3.6 Message length and encoding
19. Average message length? Distribution (share of 1, 2, 3+ segment messages)?
20. Share of messages requiring UCS-2 (emoji, special characters, non-Latin scripts)?

### 3.7 Delivery receipts and inbound traffic
21. Do all telcos return DLRs for all messages? Per segment or per message?
22. Expected inbound (MO) volume: replies, opt-outs, keywords?

### 3.8 Retention and reporting
23. How long must message records (metadata and content) be retained? (Regulatory + business.)
24. How long must detailed delivery reports be queryable by clients in the portal?
25. Required reporting granularity (per message, hourly aggregates, daily aggregates)?

### 3.9 Telco capacity (must be compared with demand)
26. Contracted TPS per telco, per account, per bind? Number of binds?
27. Can capacity be increased, how quickly, and at what cost?

## 4. Derived values (to be computed once data exists)

These formulas will be used **after** §3 is answered. All inputs must cite their source.

| Value | Formula |
| ----- | ------- |
| Monthly segments | `monthly_messages × average_segments_per_message` |
| Average TPS (active window) | `monthly_segments ÷ (days × active_hours_per_day × 3600)` |
| Peak TPS | `max(busiest_hour_share × daily_segments ÷ 3600, largest_campaign_segments ÷ required_completion_seconds)` — plus concurrent baseline traffic |
| Required telco capacity per network | `peak_TPS × network_share × safety_margin` (compare with contracted TPS) |
| DLR events per month | `monthly_segments_or_messages × DLR_ratio` (per telco behaviour) |
| Storage per month (messages) | `monthly_messages × bytes_per_message_record` (record size measured from the actual schema) |
| Total retained storage | `storage_per_month × retention_months` (+ indexes, replicas, backups) |

## 5. Collected data

| Question # | Answer | Source / evidence | Date | Confidence |
| ---------- | ------ | ----------------- | ---- | ---------- |
| — | *No data collected yet.* | — | — | — |

## 6. Interim design guidance

Until data is available:

- Design for **horizontal scalability** (stateless processes, idempotent consumers, partitionable data)
  without deploying distributed infrastructure prematurely.
- Make throughput limits **configuration**, not code.
- Build load-testing capability early so that when targets are known they can be verified.
- Treat telco contracted throughput as the most likely real ceiling — the platform does not need to
  accept traffic far faster than it can deliver it (beyond buffering bursts).

## 7. Provisional design targets (owner decision 2026-11-26)

The owner decided on 2026-11-26 to proceed with **provisional platform design targets** (Phase 2 EX-2,
decisions TR-1 to TR-9, gate PG-10; [decision log §16](../engineering/decision-log.md#16-owner-decisions--provisional-traffic-targets-phase-2-ex-2-2026-11-26)).
The targets, test workloads (message submissions, segments and receipt events, defined separately) and
distributions are recorded in the [traffic evidence worksheet §7](../discovery/traffic-evidence-worksheet.md#7-provisional-design-targets-owner-decision-2026-11-26-ex-2).

- They are **platform capacity targets**, not evidence, not a demand forecast and not telco throughput. Every
  distribution in them is synthetic or provisional.
- §5 above stays empty: **no traffic data has been collected** (EX-6 deferred). The questions in §3 remain open.
- The rule in §1 still applies: D-004 and D-060 remain **open and unused**; nothing is sized on, compared with or
  checked against the unverified monthly figure.
- **Gate PG-10:** before general production rollout, the targets are re-baselined against validated
  pilot-customer volume information and contracted telco limits; production infrastructure sizing requires this
  re-baseline.
