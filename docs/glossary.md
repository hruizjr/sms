# Glossary

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft | Project owner | 2026-11-24 |

Plain-language explanations of terms used in this project. Definitions describe general industry
usage; **how each Philippine telco implements a concept must be confirmed with that telco.**

## Messaging and telecommunications

| Term | Meaning |
| ---- | ------- |
| **A2P** (Application-to-Person) | Messages sent by a business system to a person: OTPs, alerts, reminders, marketing. This platform's business. |
| **P2P** (Person-to-Person) | Ordinary texting between people. Telcos often price and police A2P traffic differently from P2P. |
| **Aggregator** | A company that connects many business customers to mobile networks through one platform and one set of telco connections. |
| **MNO** (Mobile Network Operator) | A company operating a mobile network (e.g. Smart, Globe, DITO). |
| **SMSC** (Short Message Service Centre) | The telco system that receives SMS for delivery and forwards them to handsets. Our platform connects to it. |
| **SMPP** (Short Message Peer-to-Peer) | The most common protocol for connecting an aggregator to a telco SMSC. A persistent TCP connection carrying binary commands. Version 3.4 is widely used; telcos may use other versions or extensions. |
| **ESME** | "External Short Message Entity" — SMPP's term for the client side, i.e. us. |
| **Bind** | An authenticated SMPP session. Types: *transmitter* (send only), *receiver* (receive only), *transceiver* (both). Telcos limit how many binds we may hold. |
| **`submit_sm`** | SMPP command to submit one message (or one segment) to the telco. The telco answers with `submit_sm_resp` containing its own message ID. |
| **`deliver_sm`** | SMPP command the telco sends to us, carrying either a **delivery receipt** or an **incoming message**. |
| **`enquire_link`** | SMPP keep-alive "ping" to check a session is still alive. |
| **Window size** | Number of SMPP requests that may be outstanding (sent, awaiting response) at once on one bind. Telco-defined. |
| **TPS** (Transactions / messages Per Second) | Throughput limit, usually contracted per account and/or per bind. |
| **MT** (Mobile-Terminated) | A message delivered *to* a phone. Most of our traffic. |
| **MO** (Mobile-Originated) | A message sent *from* a phone, e.g. a customer replying "STOP". Requires an inbound number/short code. |
| **DLR** (Delivery Receipt / Delivery Report) | Notification from the telco about what happened to a message: delivered, failed, expired, etc. Can arrive seconds to hours later, or never. Formats vary by telco. |
| **Segment** | SMS has a fixed payload size. Long messages are split into segments and reassembled on the phone. Telcos usually charge per segment. |
| **GSM-7** | The default 7-bit SMS alphabet (basic Latin letters, digits, some symbols). Up to **160** characters in one segment, **153** per segment when split. Some characters (e.g. `€ [ ] { } ^ ~ \ \|`) count as two. |
| **UCS-2** | 16-bit encoding used when a message contains characters outside GSM-7 (e.g. emoji, `ñ` in some cases, non-Latin scripts). Up to **70** characters in one segment, **67** per segment when split. A single emoji can more than double cost. |
| **UDH** (User Data Header) | Header bytes inside a segment that tell the phone how to reassemble multi-part messages. It consumes payload space, hence 153/67. |
| **Sender ID / Masking** | What the recipient sees as the sender, e.g. `MYBANK`. In the Philippines often called "SMS masking". Usually must be registered and approved by each telco. |
| **Short code** | A short number (e.g. 4–5 digits) used for A2P services, often for two-way messaging. Allocation is telco/regulator controlled. |
| **Long code** | A regular-length mobile number used as a sender. |
| **OTP** | One-Time Password, typically for login or transaction verification. Time-sensitive and security-sensitive. |
| **MNP** (Mobile Number Portability) | Subscribers in the Philippines can keep their number when switching networks (Mobile Number Portability Act, RA 11202). The number prefix therefore does **not** reliably tell you the recipient's current network. |
| **HLR lookup** | A query to find which network currently serves a number. Availability and cost in the Philippines must be confirmed. |
| **Routing** | Choosing which telco connection carries a message. Depends on the recipient's network, agreements, cost, and availability. |
| **On-net / off-net** | Delivering to a subscriber of the same telco we submitted to (on-net) vs. another telco's subscriber (off-net). Whether a telco accepts off-net A2P traffic is a commercial/technical question. |
| **Validity period** | How long the telco keeps retrying delivery to a phone that is off or out of coverage before giving up. |
| **TON / NPI** | SMPP fields describing number format (Type of Number, Numbering Plan Indicator). Correct values are telco-specified. |
| **CDR** | Call/Charging Detail Record — the telco's billing record of traffic, used for reconciliation. |
| **NTC** | National Telecommunications Commission — the Philippine telecom regulator. |
| **NPC** | National Privacy Commission — enforces the Data Privacy Act of 2012 (RA 10173). |
| **Smishing** | SMS phishing. A major abuse risk for A2P platforms. |

## Architecture and engineering

| Term | Meaning |
| ---- | ------- |
| **Tenant** | One client business using the platform. Tenants must never see each other's data. |
| **Multi-tenancy** | One platform serving many tenants with isolated data. |
| **Control plane** | The parts that *manage* the system: accounts, users, pricing, billing, configuration, admin. |
| **Data plane** | The parts that *move messages*: ingestion, queueing, dispatch, delivery receipts. High volume, latency-sensitive. |
| **Modular monolith** | One codebase/deployable family with strict internal module boundaries; can later be split into services. |
| **Microservices** | Many independently deployed services communicating over the network. Powerful but operationally expensive. |
| **Provider adapter** | Code that translates our internal "send this message" into one telco's protocol, and their responses back. |
| **Message broker / queue** | Infrastructure (e.g. Kafka, Redpanda) that stores messages between producers and consumers durably. |
| **At-least-once** | A processing guarantee: every event is processed, but possibly more than once. Requires idempotency. |
| **Exactly-once** | Every event processed exactly once. Achievable inside some systems, but **not end-to-end for SMS**, because a telco timeout leaves us unsure whether the message was accepted. |
| **Idempotency** | Doing the same operation twice has the same effect as doing it once. |
| **Idempotency key** | A client-supplied unique value identifying a request so retries are recognised as duplicates. |
| **Transactional outbox** | Writing a database change and the event about it in one transaction, then publishing the event separately. Events for rolled-back changes are never published and committed events are not lost, but an event **may be published more than once**, so consumers must be idempotent. |
| **Back-pressure** | Slowing down intake when downstream is overloaded, rather than failing or buffering without limit. |
| **Circuit breaker** | Automatically stopping calls to a failing dependency for a period, to let it recover and fail fast. |
| **Dead-letter queue / store** | Where messages that cannot be processed are parked for investigation. |
| **Provider acknowledgment** | The telco's response accepting (or rejecting) a submitted message, e.g. with its own message ID. It means the telco took responsibility for delivery — **not** that the phone received it. |
| **Ambiguous outcome** | A submission whose result is unknown (e.g. timeout with no response). The telco may or may not have accepted it; retrying risks a duplicate SMS. |
| **`UNKNOWN` status** (proposed) | A message with no final delivery information within the configured window. Uncertain — not proof of non-delivery; a late receipt can still update it. |
| **Request fingerprint** | A hash of a request's meaningful content, stored with an idempotency key to detect the same key being reused for a different request. |
| **Monetary scale** | How many decimal places an amount is stored with (e.g. 2 for centavos). Must cover the precision of real rates. |
| **Rounding mode** | The rule for rounding a value that falls between two representable amounts (e.g. half-up, half-even). Must be explicit and consistent. |
| **Compensating entry** | A new ledger entry that corrects or reverses an earlier one, instead of editing it. |
| **Headless UI primitives** | Unstyled components providing behaviour (keyboard handling, focus management, accessibility attributes) for complex widgets such as dialogs and menus; styling is applied separately (here, with Tailwind). |
| **Ledger** | An append-only list of financial entries. Balances are computed from entries; entries are never edited. |
| **Credit reservation (hold)** | Temporarily setting aside credit for a message that has been accepted but not yet charged. |
| **Reconciliation** | Comparing two records of the same reality (our ledger vs. telco CDRs/invoices) and investigating differences. |
| **RBAC** | Role-Based Access Control — permissions granted via roles. |
| **RLS** (Row-Level Security) | PostgreSQL feature that restricts which rows a database session can see, usable as extra tenant protection. |
| **ADR** | Architecture Decision Record — a short document recording a decision, alternatives, and consequences. |
| **SLA** | Service Level Agreement — contractual commitments (availability, latency, support). |
| **E.164** | International phone number format, e.g. `+639171234567`. |
