# Telco Technical Discovery Checklist

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — no telco information received yet; technical discovery formally deferred during simulator-only development (owner decision 2026-11-26, EX-3/EX-7). **No question below has been answered** | Project owner | 2026-11-26 |

## Purpose and rules

This checklist lists what we must learn from each telco before designing or building its integration.

- **Every item requires confirmation from the telco**, preferably in official written technical
  documentation. Items marked **[Business]** also need a commercial/contractual answer; items marked
  **[Regulatory]** also need legal/regulatory input.
- Record each answer with its **source** (document title, version, date, or named contact and date).
  Store the telco's documents outside Git if they are confidential; record only their reference here.
- **AI agents must not fill in answers from general knowledge.** Common industry practice is noted only to
  explain *why* a question matters.
- Before asking, clarify whether **PLDT Enterprise and Smart** are separate integrations or a single
  route/contract (they belong to the same corporate group). **Update 2026-11-26 (EX-9 review, D-160, pending
  partner ratification):** the coverage objective is direct integration with **Smart, Globe and DITO**; PLDT
  Enterprise is not a separate mobile network and may be a commercial or enterprise contracting counterpart.
  Only **direct** aggregation is pursued: no third-party aggregator or intermediary may route messages, so the
  answers below decide whether direct integration with each operator is possible. **No question is answered by
  this update.**

## Simulator-only development (owner decision 2026-11-26)

The owner decided on 2026-11-26 to proceed with **simulator-only development and testing** without a telco
specification (Phase 2 EX-3, decisions S1–S8; [decision log §15](../engineering/decision-log.md#15-owner-decisions--simulator-only-development-phase-2-ex-3-2026-11-26)). Telco technical discovery is **formally deferred**
during simulator-only development (EX-7 amended). This changes nothing below:

- **Every question remains unanswered.** No item is ticked and no status in the matrix changes until a telco
  answers in writing with a recorded source.
- **The answers remain production prerequisites.** No real adapter is built without official documentation
  and separate owner approval (gate PG-1), and every real adapter must pass the provider contract suite using
  the provider's official specifications, plus validation in its test environment where available (§17); if
  none is available, the gap is documented and an alternative validation is explicitly approved (gate PG-9).
- **The simulator does not answer these questions.** Its settings are generic and synthetic (S2): it is not
  named after, and does not emulate, Smart, Globe, PLDT Enterprise or DITO, and its capabilities are labelled
  *synthetic* or *unknown*. Results against it are "verified against the simulator" only (S7).
- **Discovery continues** outside simulator-only development (S8), including the PLDT Enterprise / Smart
  question (D-112), direct or intermediary connectivity (D-118), and whether to approach DITO.

## Status matrix

Legend: `—` not asked · `Asked` · `Partial` · `Confirmed` (with evidence) · `N/A`

| # | Topic | Smart | Globe | PLDT Ent. (counterpart) | DITO |
| - | ----- | ----- | ----- | --------- | ---------- |
| 1 | Connectivity protocols | — | — | — | — |
| 2 | SMPP specifications | — | — | — | — |
| 3 | API documentation (non-SMPP) | — | — | — | — |
| 4 | Authentication | — | — | — | — |
| 5 | IP allowlisting | — | — | — | — |
| 6 | VPN / private connectivity | — | — | — | — |
| 7 | Throughput limits | — | — | — | — |
| 8 | Concurrent sessions | — | — | — | — |
| 9 | Sender ID registration | — | — | — | — |
| 10 | Encoding and segmentation | — | — | — | — |
| 11 | Delivery receipts | — | — | — | — |
| 12 | Incoming messages (MO) | — | — | — | — |
| 13 | Error codes | — | — | — | — |
| 14 | Retry policies | — | — | — | — |
| 15 | Billing and reconciliation | — | — | — | — |
| 16 | SLA and support | — | — | — | — |
| 17 | Test environment | — | — | — | — |
| 18 | Production onboarding | — | — | — | — |
| 19 | Aggregator / reseller authorization | — | — | — | — |
| 20 | Content and traffic policies | — | — | — | — |
| 21 | Routing scope and number portability | — | — | — | — |

---

## 1. Connectivity protocols
- [ ] Do you offer direct A2P connectivity to aggregators, or only through intermediaries? (D-118; we pursue
      direct connectivity only — D-160, pending partner ratification)
- [ ] Which protocols are offered for A2P submission: SMPP, HTTP/REST API, other?
- [ ] Which is recommended/required for our expected volume?
- [ ] Are different protocols used for submission, delivery receipts, and incoming messages?
- [ ] Is TLS supported/required on the connection? (Plain SMPP runs over unencrypted TCP; encryption often
      relies on VPN or TLS wrapping — confirm.)

## 2. SMPP specifications (if SMPP is used)
- [ ] SMPP version (e.g. 3.3, 3.4, 5.0) and the telco's own interface specification document.
- [ ] Supported bind types: transmitter, receiver, transceiver.
- [ ] Required values: `system_id`, `password`, `system_type`, `interface_version`, address range.
- [ ] Source and destination address TON/NPI values for alphanumeric sender IDs, short codes, and long codes.
- [ ] Destination number format expected (with/without `63`, leading `0`, `+`).
- [ ] Supported optional parameters (TLVs) and any vendor-specific TLVs.
- [ ] `data_coding` values supported and their mapping (GSM-7 default, UCS-2, others).
- [ ] Concatenation method: UDH, `sar_*` TLVs, or `message_payload`.
- [ ] Window size (maximum unacknowledged requests per bind).
- [ ] `enquire_link` interval expected; idle timeout; behaviour on missed keep-alives.
- [ ] `registered_delivery` usage for requesting DLRs.
- [ ] Validity period and scheduled delivery support/limits.
- [ ] Response timeout expectations for `submit_sm_resp`.
- [ ] Maintenance/reconnection etiquette (backoff requirements, maximum reconnect rate).

## 3. API documentation (if HTTP/other APIs are used)
- [ ] Official API documentation and version.
- [ ] Endpoints for submission, status query, DLR callback, MO callback.
- [ ] Request/response formats, size limits, batch submission support.
- [ ] Idempotency or duplicate-detection support on the telco side.
- [ ] Callback security (signatures, IP ranges, mutual TLS).

## 4. Authentication
- [ ] Credential types (SMPP system ID/password, API keys, certificates, OAuth).
- [ ] Credential issuance, rotation procedure, expiry.
- [ ] Separate credentials per environment (test/production) and per traffic type?
- [ ] Mutual TLS or client certificates required?

## 5. IP allowlisting
- [ ] Must our source IPs be registered? How many? Lead time to change?
- [ ] Do they need static public IPs, and do cloud provider IP ranges qualify?
- [ ] Which telco IPs/hostnames must we allow for callbacks and connections?

## 6. VPN / private connectivity
- [ ] Is an IPsec VPN, private line, or cross-connect required or available?
- [ ] Supported VPN parameters and who provides the equipment/endpoint.
- [ ] Redundancy options (dual tunnels, dual sites).
- [ ] Does this constrain where we can host (e.g. specific data centres)? **[Business]**

## 7. Throughput limits
- [ ] Contracted TPS per account, per bind, per sender ID, per traffic type (OTP vs. bulk).
- [ ] How TPS is measured (per segment or per message; per second or averaged).
- [ ] Behaviour when exceeded (throttling error, queueing, disconnect, penalties).
- [ ] Process, lead time, and cost to increase capacity. **[Business]**
- [ ] Separate limits or priority lanes for OTP/time-critical traffic?

## 8. Concurrent sessions
- [ ] Maximum number of simultaneous binds per account and per bind type.
- [ ] Are binds allowed from multiple source IPs/sites (for redundancy)?
- [ ] What happens if we exceed the bind limit?

## 9. Sender ID registration
- [ ] Is registration of alphanumeric sender IDs ("masking") required? Process, documents, lead time, cost.
- [ ] Does registration with one telco apply to others? (Assume not until confirmed.)
- [ ] Rules on sender ID format (length, characters) and prohibited sender IDs.
- [ ] Behaviour when an unregistered sender ID is used (rejection, replacement, blocking).
- [ ] Availability and process for short codes and long numbers; two-way support. **[Business]**
- [ ] Any regulatory rules on sender identification. **[Regulatory]**

## 10. Message encoding and segmentation
- [ ] Supported character sets; GSM-7 extension table support; national language shift tables.
- [ ] UCS-2 support and handling of characters outside UCS-2 (e.g. some emoji).
- [ ] Maximum number of segments per concatenated message.
- [ ] Who performs segmentation (us vs. telco when using `message_payload`)?
- [ ] How segments are counted for billing and TPS.

## 11. Delivery receipts
- [ ] Are DLRs provided for all traffic? On-net and off-net?
- [ ] DLR format (e.g. text in `short_message` vs. TLVs `receipted_message_id`/`message_state`).
- [ ] Message ID format and consistency between `submit_sm_resp` and DLR (e.g. hex vs. decimal).
- [ ] Are provider message IDs unique across binds and over time? After how long can an ID be reused?
- [ ] Can multiple final receipts arrive for the same message? Can a receipt arrive before `submit_sm_resp`?
- [ ] DLR per segment or per message?
- [ ] Full list of final and intermediate states and their meanings.
- [ ] Typical and maximum DLR delay; does the telco retry DLR delivery if we are disconnected?
- [ ] Are DLRs delivered on the same bind or a separate receiver bind?

## 12. Incoming messages (MO)
- [ ] Is MO supported for our numbers/short codes? Format and delivery channel.
- [ ] Opt-out keyword handling: do telcos manage STOP keywords centrally, or must we? **[Regulatory]**
- [ ] Encoding and concatenation of inbound messages.

## 13. Error codes
- [ ] Complete list of error/status codes (SMPP `command_status` and any vendor-specific codes) with meanings.
- [ ] Which codes are temporary (retryable) vs. permanent?
- [ ] Throttling-specific codes and expected backoff.
- [ ] Codes indicating blocked content, invalid sender ID, invalid number, ported number, or insufficient balance.

## 14. Retry policies
- [ ] Telco recommendations for retrying after temporary errors (timing, maximum attempts).
- [ ] Does the telco retry delivery to unreachable handsets internally? For how long (validity period)?
- [ ] Recommended handling of submission timeouts (no `submit_sm_resp`) to avoid duplicates.
- [ ] Does the telco detect duplicate submissions? On what basis, and is this guaranteed in writing?
- [ ] Is there an API to query the status of a previously submitted message (to resolve ambiguous outcomes)?

## 15. Billing and reconciliation
- [ ] Billing basis: per submitted segment, per delivered segment, per message? **[Business]**
- [ ] Rates; differences by traffic type, destination network, volume tier. **[Business]**
- [ ] Currency and number of decimal places in rates; the telco's own rounding rules on invoices. **[Business]**
- [ ] Prepaid deposit, credit limit, minimum commitment, or settlement terms. **[Business]**
- [ ] Availability and format of CDRs or usage reports; frequency; delivery method.
- [ ] Dispute and reconciliation process; timelines.
- [ ] Invoicing and tax documentation. **[Regulatory]**

## 16. SLA and support
- [ ] Availability commitment for SMSC/API; measurement method.
- [ ] Latency commitments, if any.
- [ ] Planned maintenance windows and notification lead time.
- [ ] 24/7 support contacts, escalation matrix, incident communication channel.
- [ ] Service credits/penalties. **[Business]**

## 17. Test environment
- [ ] Is a test/staging SMSC or sandbox available? How does it differ from production?
- [ ] Test credentials, test numbers, and whether test messages reach real handsets.
- [ ] Ability to simulate errors, throttling, and DLR states.
- [ ] Certification or acceptance test plan required before production.

## 18. Production onboarding
- [ ] Step-by-step onboarding process and typical duration.
- [ ] Documents required (company registration, permits, KYC, use-case descriptions).
- [ ] Technical acceptance testing and sign-off.
- [ ] Ramp-up limits after go-live (initial TPS caps).
- [ ] Ongoing compliance reporting requirements.

## 19. Aggregator and reseller authorization
- [ ] Is our company permitted to aggregate traffic from multiple end customers? Conditions? **[Business] [Regulatory]**
- [ ] Must each end customer or sender ID be declared/registered with the telco?
- [ ] Is reselling to other aggregators permitted? **[Business]**
- [ ] Required NTC or other regulatory registrations the telco expects us to hold. **[Regulatory]**
- [ ] Liability allocation for abusive traffic sent by our customers. **[Business]**

## 20. Content and traffic policies
- [ ] Rules on URLs/links in A2P messages; URL registration or allowlisting.
- [ ] Prohibited content categories; marketing time-of-day restrictions. **[Regulatory]**
- [ ] Traffic classification (OTP/transactional vs. promotional) and differing rules or prices.
- [ ] Spam/fraud monitoring by the telco and its consequences (blocking, suspension).

## 21. Routing scope and number portability
- [ ] Does the telco accept traffic destined for other networks' subscribers (off-net)? Price?
- [ ] If this route is unavailable, may traffic for this network's subscribers be sent via another telco (failover)? (D-119)
- [ ] How are ported numbers handled? Does the telco route correctly, or must we determine the current network?
- [ ] Is an MNP database, HLR lookup, or number-range data available to us? Cost and terms.

---

## Answers log

| Date | Telco | Topic # | Answer summary | Source / evidence | Recorded by |
| ---- | ----- | ------- | -------------- | ----------------- | ----------- |
| — | — | — | *No answers recorded yet.* | — | — |
