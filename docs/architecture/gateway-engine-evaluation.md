# SMS gateway engine evaluation — Jasmin vs. a custom Go gateway

| Field | Value |
| ----- | ----- |
| Status | **Concluded — the tested Jasmin 0.11.0 deployment is rejected for production adoption** (owner decision, 2026-11-25). Jasmin may be consulted as a **reference implementation** only. No ADR status changed; no code, dependency, or infrastructure added to the product. |
| Date | 2026-11-25 (source and documentation review, then the time-boxed spike, then the owner decision) |
| Subject | Jasmin SMS Gateway ([repository][repo], [documentation][docs]) as the SMS gateway engine under our Go platform |
| Evidence base | **Review:** Jasmin source at commit [`0aac58e`][head] (`master`, 2026-04-26), PyPI, Docker Hub, the official docs site, and the OSV vulnerability database, all read on 2026-11-25. **Spike:** `jookies/jasmin:0.11.0` with RabbitMQ 4.4.1, Redis 8.12.0 and `jookies/smppsim:1.0.1`, run on 2026-11-25 (14:22–15:04 UTC); exact images and configuration in §9.2 |

Labels used throughout:

- **[V]** verified fact from the source or documentation review, with a cited source;
- **[A]** assumption or inference that was not tested;
- **[R]** recommendation;
- **[T-PASS] / [T-FAIL]** result **observed in the spike**, for the tested deployment only (§1.2);
- **[T-INC]** spike result that was **inconclusive**;
- **[NT]** **not tested** in the spike.

Sections 2–8 are the analysis written **before** the spike. They are kept as the record of the research, with spike results added where a scenario tested an item. Section 9 holds the spike, and section 11 the direction now in force.

---

## 1. Outcome (owner decision, 2026-11-25)

### 1.1 Decision and direction

1. **Rejected for production.** The tested Jasmin 0.11.0 deployment is **not adopted** as our production SMS gateway, and Jasmin is **not added as a runtime dependency** of the product.
2. **Reference implementation only.** Jasmin may be consulted as a reference for messaging concepts when we build our own gateway functionality, for example:
   - SMPP connector and session lifecycle;
   - PDU handling and submission workflows;
   - segmentation, GSM encoding and Unicode handling;
   - delivery-receipt parsing and correlation;
   - routing, filters and provider selection;
   - throughput management, SMPP windows and throttling;
   - connection recovery and provider state.

   Its implementation and its known failure modes (§9.3) must not be reproduced blindly. Our own reliability, security, multi-tenancy and data-integrity rules apply.
3. **No source-code adoption.** No Jasmin source code is copied, ported or adopted. Any selective reuse in the future would need a separate owner decision, after checking the exact source revision, the Apache-2.0 obligations (§7), attribution, and the dependencies it would bring.
4. **No SMPP library has been selected.** Selecting one is a dependency decision for the owner (AGENTS.md §17), taken when gateway development is scheduled.
5. **Go-owned gateway direction.** The intended direction is gateway functionality owned and written in Go, behind the provider adapter (ADR-0012). The gateway architecture, the SMPP library, and any gateway packages remain **subject to a separate architecture decision**. Nothing is being implemented now.
6. **Spike assets stay outside the repository.** The spike harness, its fault-injecting SMPP proxy, the simulator configuration, the logs and the evidence archive are kept outside this repository, so they can be reused to test our own SMPP implementation later. They are not merged into the product, and none of their dependencies are added to it.
7. **No further Jasmin work.** No further Jasmin research, testing or experimentation is authorized.

### 1.2 Scope of the conclusion

The spike results apply to **the tested deployment only**:

- the `jookies/jasmin:0.11.0` image (Python 3.11.6), the latest tagged release;
- RabbitMQ 4.4.1, run with a temporary compatibility permit (§9.2), and Redis 8.12.0;
- the upstream SMPP simulator `jookies/smppsim:1.0.1`, never a real telco;
- the configuration in §9.2, with shortened timers for the timeout scenario.

They **do not prove** that every Jasmin version or deployment configuration is unsuitable. In particular:

- RabbitMQ 3.x, and RabbitMQ 4.x releases other than 4.4.1, were not tested;
- later Jasmin commits or releases were not tested;
- patched or hardened builds were not tested.

The decision rests on the tested evidence, together with the cost of carrying and maintaining the patches that would be needed (§9.6).

Two links between the earlier review and the spike:

- **Same code paths.** The review read `master` at `0aac58e`. The key code paths it cites — timeout requeue, DLR purge, Redis-down skip, and queue declaration — are identical in the 0.11.0 tag, as checked with `git grep` during the spike.
- **Review-only findings.** Findings that come only from the source or documentation review (marked **[V]**) were not all exercised by the spike. Section 9.5 lists what was not tested.

**Simulator, not a mobile network.** Wherever this document says a message was "delivered" or "reached the SMSC" in the spike, the evidence is the SMPP simulator's behaviour. That means the `submit_sm` PDUs it received, the SMSC message IDs it assigned, and the `DELIVRD` receipts it generated. No real mobile network or handset was involved, so **actual delivery to recipient handsets was not tested** — including whether repeated submissions would reach a recipient as duplicate SMS.

### 1.3 Spike findings in brief

Full results are in §9.3–§9.5.

| Area | Result in the tested deployment |
| ---- | ------------------------------- |
| SMPP submission timeout | **[T-FAIL] Blind resubmission → duplicate upstream submission attempts.**<br>- One withheld `submit_sm_resp` produced **2** `submit_sm` PDUs for one message; three withheld produced **4**.<br>- Our side recorded only one SMSC message ID per logical message, so **the repeated submissions were invisible to the platform**.<br>- Shorter timers changed only the timing.<br>- **[NT]** Whether recipients would actually receive duplicate SMS on their handsets was not tested (simulator only).<br>- This remains a **confirmed reliability failure and a production blocker**: after an ambiguous timeout, Jasmin resubmits blindly, which AGENTS.md rule 23 forbids. |
| RabbitMQ restart | **[T-FAIL] Non-durable queues; message loss.**<br>- All Jasmin queues are non-durable. After a broker restart every queue was gone: **10 of 10 queued messages were lost**.<br>- Jasmin did not restore its consumers.<br>- A new `/send` then returned **HTTP 200 `Success`** with an embedded failure — a failure reported as success. |
| RabbitMQ 4.4.1 compatibility (the release tested) | **[T-FAIL] by default.**<br>- Jasmin 0.11.0 cannot declare its queues on RabbitMQ 4.4.1 (`transient_nonexcl_queues` is deprecated and refused).<br>- It runs only with a temporary permit that RabbitMQ says will be removed.<br>- **[NT]** RabbitMQ 3.x and other 4.x releases. |
| Redis | **Mandatory dependency for DLR correlation.**<br>- **[T-PASS]** A graceful Redis restart kept correlation.<br>- **[T-FAIL]** If Redis **lost its data**, or was **down at submit time**, the SMS was still sent once, but **its receipt never reached Go**. |
| Unicode receipts | **[T-FAIL] reproducible.**<br>- A UCS2 delivery receipt from the simulator caused RabbitMQ 4.4.1 to reject an AMQP frame and close Jasmin's connection.<br>- Jasmin reconnected but **never re-created its consumers**, and the final receipt was lost.<br>- Afterwards the HTTP API still answered `Success` while nothing was submitted, until a manual restart.<br>- The encoding itself was correct.<br>- **[T-INC]** The root cause (Jasmin/txAMQP framing versus stricter parsing in RabbitMQ 4.4.1) is unresolved. |
| DLR callbacks to Go | **Bounded retries.**<br>- **10 s and 45 s outages:** **[T-PASS]** for endpoint recovery: Jasmin retried, and the final `DELIVRD` callback arrived. **[T-INC]** for callback-sequence completeness: Go recorded only 1 of the 2 expected callbacks, and the missing one is unexplained.<br>- **[T-FAIL]** After a 150 s outage the receipt was **purged** after 4 tries (no dead-letter). The simulator reported the message delivered, but Go never learned of it.<br>- **[T-INC]** The documented "404 → immediate purge" path was not observed. |
| Jasmin process restart | **[T-PASS]** Queued and in-flight messages were each submitted once and reported delivered by the simulator (two sub-cases). |
| Correlation | **[T-PASS]** Our reference in `dlr-url` and Jasmin's ID were present in every callback.<br>**[T-FAIL] (edge case)** An SMSC message ID of `"0"` arrived as an empty `id_smsc`. |
| Multipart GSM | **[T-PASS]** Two parts with UDH and correct encoding at the simulator; receipt delivered to Go. |
| Security defaults | **[T-PASS]** Invalid HTTP API and jCli credentials were rejected.<br>**Findings:** in the official image:<br>- jCli accepts the published default credentials and binds `0.0.0.0`;<br>- RabbitMQ `guest/guest` has administrator rights from another container;<br>- Redis has no password;<br>- the HTTP API is plain HTTP;<br>- `messages.log` holds message content in clear text. |

No result supports exactly-once delivery, and none is claimed.

### 1.4 Recommendation before the spike (historical)

The text below is the recommendation as written before the spike. It is kept unchanged as the record. The spike tested it; the outcome is §1.1.

> **[R] Do not adopt Jasmin as the gateway engine yet. Build to the Option C boundary, and decide with a short spike.**
>
> 1. **Ownership stays in Go.** Our Go platform keeps everything business-related: acceptance, the authoritative message record, the ledger, routing policy, uncertain-outcome handling, and reconciliation. This holds whichever engine sits underneath.
> 2. **One narrow boundary.** The gateway engine sits behind one Go interface — the provider/gateway adapter from ADR-0012.
> 3. **Jasmin is a candidate engine, not a decision.** The source code shows four behaviours that conflict with our non-negotiable messaging rules if used as shipped:
>    - **Resubmits on timeout.** A timed-out `submit_sm` is automatically resubmitted. This risks duplicate SMS to real people (AGENTS.md rule 23).
>    - **Drops receipts.** Delivery receipts are purged after 3 failed callbacks.
>    - **Loses data on broker restart.** Queues and messages are declared non-durable.
>    - **Can lose receipts silently.** If Redis is down at submit time, the receipt mapping is skipped with only a warning.
>
>    Section 3 has the sources for each.
> 4. **Maintenance and security posture are weak.**
>    - **Releases:** the last release was 0.11.1 in March 2024.
>    - **Twisted:** the pinned Twisted 23.10 has open advisories, including HTTP request smuggling.
>    - **Admin access:** the admin console uses telnet.
>    - **Message format:** internal messages are Python `pickle` objects.
> 5. **No published throughput figures exist.** Nothing supports the reported 5-billion-per-month figure, or any figure.
>
> The spike (section 9) is designed to test exactly these risks. **[R]** Adopt Jasmin (Option C, Jasmin variant) only if its acceptance criteria pass with configuration and Go-side controls alone, or with a small patch you approve. Otherwise, build the dispatch and SMPP layer in Go (Option A) behind the same adapter. The adapter boundary means the work in Groups 1–2 and the Go business layer are unaffected either way.

All four behaviours in item 3 were observed in the spike, with one difference: the receipt was purged after **4** tries, that is 3 retries (§9.3). The resubmission was observed as **repeated `submit_sm` PDUs at the simulator**. Duplicate SMS on real handsets was not tested (§1.2).

---

## 2. Comparison of options (pre-spike analysis)

- **A — Fully custom Go gateway.** We build SMPP connectivity, dispatch, throttling, routing, queueing and DLR processing in Go.
- **B — Go platform with Jasmin as the gateway engine.** Jasmin owns submission, gateway routing and failover, retries, and DLRs; Go calls Jasmin and consumes its callbacks.
- **C — Hybrid.** Go owns acceptance, the outbox, dispatch decisions, retries and uncertain outcomes. Jasmin (or another engine) provides only SMPP connectivity, encoding and multipart handling, and DLR forwarding, behind a Go adapter. Custom Go fills the gaps (for example customer-facing SMPP).

| Criterion | A — custom Go | B — Jasmin engine | C — hybrid (Go-owned dispatch, pluggable engine) |
| --------- | ------------- | ----------------- | ------------------------------------------------ |
| Development effort | **Highest.** SMPP session handling, windowing, throttling, reconnects, UDH/SAR multipart, encodings, DLR parsing, simulator. **[A]** Go SMPP libraries exist; selecting one is a separate ADR per ADR-0012. | **Lowest to a first simulated SMS.** But retries, durability and correlation must be compensated for in Go, or patched in Python. | **Medium.** Go dispatch, retry and reconciliation (needed in every option) plus an adapter. The engine is configured, not extended, unless the spike forces a patch. |
| Operational complexity | One Go process plus PostgreSQL. | Adds Jasmin (Python/Twisted, several daemons), RabbitMQ and Redis; a telnet admin console; pickle files for configuration. | The same as B if Jasmin is used, or as A if the engine is Go. |
| Maintainability | One language and toolchain; fully under our control. | Depends on a slowly maintained Python codebase (§7). Fixing a blocker means a fork in a second language. | The boundary keeps engine churn local, but a Jasmin fork is still a Python codebase we would own. |
| Performance and scaling | Unknown until load-tested; capacity is set by telco throughput limits (unconfirmed). | **[V]** No published throughput figures (§3.7). **[A]** Unknown until benchmarked. | Unknown; the spike measures Jasmin, and Go dispatch is needed anyway. |
| Reliability and recovery | Designed to our rules: outbox, uncertain states, idempotent DLR intake (ADR-0020/0021). | **[V]** Conflicts as shipped: timeout resubmission, DLR purge, non-durable queues, Redis-down mapping skip (§3.5). | Go enforces the rules. Engine behaviours that cannot be neutralized are blockers for that engine. |
| Security | Our own code, reviewed under our baseline. | **[V]** Telnet admin, plaintext HTTP API, no SMSC certificate verification, pickle over AMQP, outdated Twisted (§6). Must be fully isolated in a private network. | Same exposure as B, contained behind the adapter and a private network. |
| Financial integrity | Ledger in PostgreSQL. | Jasmin has its own balances and quotas (**[V]** in-memory, persisted every 60 s by default). Must be disabled or set to unlimited, and never used for charging. | Same as B: Go's ledger is authoritative and the engine is never consulted for money. |
| Multiple telcos | One adapter per telco (ADR-0012). | One Jasmin SMPP connector per upstream; route filters by destination, tag and more. | Go chooses the provider (commercial policy) and passes it to the engine as a connector or tag choice. |
| Dependencies and infrastructure | None new beyond an SMPP library (owner approval). | Python 3.11–3.12, Twisted 23.10, RabbitMQ, Redis, Celery (REST API only). | Same as B or A, depending on the engine. |
| Long-term ownership | Full ownership, full cost. | Shared with a small upstream; patches carried by us. | Ownership of the business core is guaranteed; the engine stays replaceable. |

**[R] Why C over B.** Under B, Jasmin's retry and failover logic would make delivery decisions our platform cannot audit and that contradict rules 23/24. C keeps those decisions in Go and uses an engine only where it is demonstrably safe.

**After the spike.** For the tested Jasmin 0.11.0 deployment, Option B and the Jasmin variant of Option C are both ruled out. The defects were found in the dispatch and receipt path, which Option C would still delegate to Jasmin (§9.7). The intended direction is a Go-owned gateway (Option A) behind the same adapter boundary, subject to a separate architecture decision (§1.1).

---

## 3. Jasmin capabilities and limitations (source and documentation review)

### 3.1 Release, runtime and dependencies

- **[V] Version.** `master` declares version `0.12` ([pyproject.toml L7][pyproject-version]), but no 0.12 tag or release exists.
- **[V] Latest releases.**
  - GitHub: 0.11.0, tagged 2023-11-11.
  - PyPI: 0.11.1, 2024-03-27.
  - Docker Hub `jookies/jasmin:latest`: 2023-11-10.
- **[V] Python and Twisted pins.** `requires-python = ">=3.11,<3.13"` ([L10][pyproject-python]); `Twisted~=23.10.0` ([L31][pyproject-twisted]). Only 23.10.0 exists in that range.
- **[V] AMQP and Redis clients.** `txAMQP3`, `txredisapi`, plus `celery` and `redis` for the optional REST API.
- **[V] RabbitMQ and Redis are mandatory.** The installation guide says Jasmin "needs a working RabbitMQ and Redis servers". Redis is "used mainly for mapping message ID's when receiving delivery receipts" ([installation][docs-install]).
- **[V] Reference compose file** ([docker-compose.yml][compose]) uses `redis:alpine` (unpinned) and `rabbitmq:3.10-management-alpine`. It publishes 2775 (SMPP server), 8990 (jCli) and 1401 (HTTP API) on all interfaces.
- **[A] RabbitMQ versions.** 3.10 is no longer a supported RabbitMQ release line. Compatibility of Jasmin's non-durable shared queues with currently supported RabbitMQ versions is unverified.

### 3.2 Gateway capabilities

| Capability | Finding |
| ---------- | ------- |
| SMPP client (upstream SMSC) | **[V]** SMPP v3.4 client connectors managed via jCli; optional TLS ([factory.py L148][smpp-ssl]). |
| SMPP server (for customers) | **[V]** Present, on `0.0.0.0:2775` by default ([configs.py L244][smpps-bind]). It authenticates **Jasmin's own users**, which would bypass our Go identity and billing (§4.6). |
| HTTP API | **[V]** `GET/POST /send` on port 1401. Success returns `Success "<uuid>"`, documented as "Message is successfully queued" ([HTTP API][docs-http]). Errors are `Error "<reason>"` with HTTP 4xx (e.g. `No route found`). |
| REST API | **[V]** A separate Falcon/Celery service with batch sending ([REST API][docs-rest]). Optional; adds Celery workers. |
| Routing | **[V]** Default, Static, RandomRoundrobin and Failover MT routes; filters on user, group, source, destination, content, time, tags, and arbitrary Python (`EvalPyFilter`) ([Filters.py L250][evalpy], [L272][tagfilter]). |
| Least-cost routing | **[V] Not implemented.** Advertised in the docs feature list, but no such route exists in the source. `BestQualityMTRoute` raises `NotImplementedError` ("Work in progress") ([Routes.py L371][bestquality]). |
| Failover | **[V] Decided once, at enqueue time.** `FailoverMTRoute` picks the first connector currently `BOUND_*` ([send.py L233][failover]). A message already queued for a connector is not moved if that connector fails later. |
| Throttling | **[V]** Per connector, `submit_throughput` (messages per second); per user, HTTP/SMPP throughput quotas ([jCli modules][docs-jcli-modules]). |
| Multipart / Unicode | **[V]** Long content split via UDH (default) or SAR, with `long_content_max_parts` default 5 ([http configs L34][longsplit]); `coding` parameter for UCS2 and others ([HTTP API][docs-http]). Telco-specific behaviour is unconfirmed (no telco specs). |
| Delivery receipts | **[V]** Levels 1 (SMSC), 2 (terminal), 3 (both). Pushed to a per-message `dlr-url`, which must reply `200` with body exactly `ACK/Jasmin` ([HTTP API][docs-http]). |
| Message IDs | **[V]** Jasmin generates a UUID per message (the `id` in responses and DLRs). The SMSC's `message_id` arrives as `id_smsc` in level-2/3 DLRs. Redis keys `dlr:<uuid>` and `queue-msgid:<smsc id>` hold the correlation ([dlr.py L254][queue-msgid]). |
| Client reference / idempotency | **[V] None.** `/send` parameters: `to, from, coding, username, password, priority, sdt, validity-period, dlr, dlr-url, dlr-level, dlr-method, tags, content, hex-content, custom_tlvs` ([HTTP API][docs-http]). There is no idempotency key and no lookup by client reference. |
| Monitoring | **[V]** Prometheus `/metrics` on the HTTP API port, jCli `stats`, Grafana examples ([installation][docs-install]). |
| Interceptors | **[V]** Run operator-supplied Python scripts on MO/MT traffic ([interception][docs-interception]). Powerful, but it is code execution inside the gateway. |

### 3.3 Persistence behaviour

- **[V] Configuration** (connectors, routes, users) is changed through jCli and persisted as Python **pickle** files ([e.g. httpccm.py L110][pickle-store]), flushed every `persistence_timer_secs` (default 60 s) ([routing configs L22][persist-timer]).
- **[V] User balances and quotas are in memory** and persisted on the same timer. The billing documentation warns that a timer set too high "can loose" critical data ([billing][docs-billing]).
- **[V] DLR correlation** lives in Redis with a TTL of `dlr_expiry`, default 86400 s (24 h) ([smpp configs L92][dlr-expiry]). **[A]** Receipts arriving after expiry cannot be correlated by Jasmin. Some telcos may deliver receipts later than 24 h; this is unconfirmed (telco checklist).

### 3.4 Throughput claims

- **[V]** The official index states only that Jasmin "is designed for performance, high traffic loads and full in-memory execution" ([index][docs-index]).
- **[V]** The Kubernetes section says its manifests "were used for making stress tests and performance metering" ([installation][docs-install]), but publishes **no figures**. We found no upstream benchmark.
- **[V]** The `submit_throughput 110` in the docs is a sample setting for a fictional SMSC, not a measurement.
- **No capacity can be claimed for Jasmin** — and certainly not for the unverified ~5 billion/month figure — without our own benchmark under verified requirements.

### 3.5 Reliability behaviours that conflict with our rules

| # | Behaviour (verified in source) | Conflicts with | Can configuration or Go controls neutralize it? |
| - | ------------------------------ | -------------- | ----------------------------------------------- |
| J1 | **Timeout resubmission.** On `SMPPRequestTimoutError` (no `submit_sm_resp`), the message is requeued and **sent again** after `requeue_delay` (default 120 s) ([listeners.py L277][timeout-requeue], [configs L180][requeue-delay]). No retry cap exists on this path; it stops only if the optional `expiration` (from `validity-period`) has passed ([L186][expired]). | AGENTS.md rule 23 (ambiguous outcomes are never blindly retried). Repeated upstream submission attempts; a **potential** downstream consequence is duplicate SMS to recipients (actual handset duplication was not tested, §9.5) | **[A] Probably not** with configuration alone. `validity-period` only bounds the window. Needs a code change (fork or upstream patch) or proof from the spike that a timeout never leads to a second `submit_sm`. **Blocker until resolved.** |
| J2 | **DLR purge.** A DLR callback that fails `max_retries` (default 3, delay 30 s), or receives HTTP 404, is "purged from queue" — rejected with no dead-letter queue ([throwers.py L626][dlr-noretry], [L638][dlr-purge]). | Rule 24 (receipts may be late, but must not be silently lost); Go DLR endpoint outages | **Partly.** Raise `max_retries` and `retry_delay`, make the Go endpoint highly available, never return 404, and reconcile. Loss is still possible during long outages. |
| J3 | **Non-durable queues and messages.** Queues are declared without `durable`, and messages are published without a persistent delivery mode (no such setting anywhere in the source; [factory.py L200][queue-declare], [L221][publish]). **[A]** By AMQP semantics, a RabbitMQ restart loses queued submits and DLRs. | ADR-0020 (no silent loss); recovery requirements | **Partly.** Go reconciliation can detect missing outcomes and mark them `UNKNOWN`. True durability needs a code change. To be confirmed in the spike. |
| J4 | **Redis-down mapping skip.** If Redis is disconnected when a message is submitted, it is still sent, but the DLR mapping is skipped with only a warning ([clients.py L599][redis-skip]). | Rule 24; silent loss of correlation | **Partly.** Correlate in Go using our own ID in `dlr-url` (§4.3), monitor Redis, and alert. |
| J5 | **Broker-down submit refusal.** If AMQP is disconnected, submission returns failure ([clients.py L567][amqp-down]). | — | Acceptable: Go sees a clear error and keeps the message queued on our side. |
| J6 | **Silent discard.** Messages waiting for a disconnected connector are discarded when older than `submit_max_age_smppc_not_ready` ([listeners.py L196][overaged]). **[A]** Whether the originating application receives a failure receipt was not confirmed. | Rule 24 / observability | To be tested in the spike. |
| J7 | **In-memory retry counters.** Retry counters and requeue timers are held in memory ([throwers.py][throwers]) and reset on restart. | Predictable retry limits | Bounded by J2 mitigations; observe in the spike. |

**Spike results for J1–J7** (tested deployment only, §1.2; details in §9.3–§9.5):

| # | Result |
| - | ------ |
| J1 | **[T-FAIL] Confirmed (S8)** as duplicate upstream submission attempts. One withheld `submit_sm_resp` led to 2 `submit_sm` PDUs for the same message; three withheld led to 4. Our side saw one SMSC ID per logical message. Actual duplicate delivery to handsets **[NT]**. |
| J2 | **[T-FAIL] Bounded retries confirmed (S7c).** After a 150 s endpoint outage the receipt was purged after 4 tries. For the 10 s and 45 s outages, endpoint recovery was **[T-PASS]** (the final `DELIVRD` arrived), but callback-sequence completeness was **[T-INC]**: 1 of 2 expected callbacks was recorded, unexplained. **[T-INC]** The "404 → immediate purge" path was not observed, because Jasmin reported our 404 as a generic `HttpApiError()`. |
| J3 | **[T-FAIL] Confirmed (S5).** All queues are non-durable, and 10 of 10 queued messages were lost on a broker restart. Beyond the review, Jasmin also did not recover its consumers and reported a failed `/send` as `Success`. On RabbitMQ 4.4.1 (the only release tested), the non-durable queues are refused by default (§9.2). |
| J4 | **[T-FAIL] Confirmed (S6c).** Redis was down at submit time and the receipt never reached Go. **[T-FAIL]** Redis data loss between submit and receipt had the same effect (S6b). **[T-PASS]** A graceful Redis restart kept correlation (S6a). |
| J5 | **[NT]** A broker that is down at submit time was not tested as such. Related observation (S5, S9): after a broker restart or a lost consumer, `/send` returned HTTP 200 `Success` although the message was never submitted. A `Success` response therefore cannot be treated as proof that the message was queued. |
| J6 | **[NT]** |
| J7 | **[NT]** |

---

## 4. Integration architecture as evaluated (Option C with Jasmin — not being built)

This section describes how Jasmin **would** have been integrated. Following the owner decision (§1.1) it is **not being built**. It is kept because its Go-side principles are gateway-neutral and carry forward:

- our own ID travels with every submission;
- an ambiguous submission is never resent automatically;
- raw receipts are stored append-only and acknowledged only after commit;
- the ledger stays authoritative.

```
Customer ─▶ Go API ─▶ PostgreSQL: message ACCEPTED + credit reservation + outbox (one transaction)
                         │
            Go dispatcher (claims outbox; chooses provider/route by commercial policy)
                         │  records SUBMISSION ATTEMPT before calling the engine
                         ▼
            GatewayEngine adapter (Go) ──HTTP /send──▶ Jasmin ──AMQP──▶ SMPP connector ─▶ SMSC simulator
                         ▲                                                   │
            Go DLR intake ◀──── HTTP callback (dlr-url) ◀── DLRThrower ◀──────┘
                 │ stores raw receipt (append-only), applies status precedence, settles/releases via ledger
```

### 4.1 Submission

- **[R] What Go sends.** The adapter calls Jasmin's `/send` on a private network with:
  - a single Jasmin system user;
  - `dlr=yes`, `dlr-level=3`, `dlr-method=POST`;
  - a bounded `validity-period`;
  - `tags` (or a per-provider user) so that **Go's** routing decision selects the Jasmin connector.

  Jasmin routes are configured as static mappings that obey the tag. Jasmin's own failover is not relied on for business routing.
- **[V] What "success" means.** `Success "<uuid>"` means *queued in Jasmin*, not submitted to the telco. Go records the milestone as "accepted by gateway engine", never as "sent".
- **[R] Error handling.** The adapter classifies `Error` responses: route or authorization errors are permanent; "Cannot send submit_sm" or broker errors are retryable; a transport timeout or connection reset is **ambiguous**. Retryable errors are retried from Go with backoff (the message was never queued).

### 4.2 Ambiguous outcomes and duplicates

- **[V] Jasmin has no idempotency key.** If `/send` times out after Jasmin queued the message, Go cannot ask Jasmin whether it exists.
- **[R] Uncertain, not resent.** Go moves the message to `SUBMIT_UNCERTAIN` (ADR-0021) and never resends automatically. It is resolved by an arriving DLR (§4.3) or by reconciliation, under the owner's per-provider policy (D-095).
- **Repeated upstream submission attempts from the engine are J1** — a blocker for Jasmin until the spike resolves it.

### 4.3 Message ID mapping

- **[R] Our ID travels with the message.** Go generates its own message ID and attempt ID and puts them in the per-message `dlr-url` query string (for example `…/internal/dlr/jasmin?attempt=<id>`).
- **Every DLR is self-correlating to our record**, even if the `/send` response was lost. **[A]** This assumes Jasmin preserves the query string when appending DLR parameters; verify in the spike. **[T-PASS]** Confirmed for the tested setup: our reference was present in every callback (S3, §9.3).
- **[R] Stored mapping:** our message → attempt → Jasmin UUID (`id`) → SMSC `message_id` (`id_smsc`). Telco and Jasmin IDs are stored as mappings, never as our primary identity (AGENTS.md §11.2).

### 4.4 Delivery receipts

- **[V] The protocol.** Jasmin pushes level-1 (SMSC acceptance; `message_status` is the `submit_sm_resp` status) and level-2 (terminal) receipts as HTTP callbacks. It considers a receipt delivered only on HTTP 200 with body `ACK/Jasmin`.
- **[R] Go DLR intake:**
  1. authenticates the call (§6);
  2. writes the raw receipt to append-only history;
  3. applies status precedence idempotently, keyed by (attempt, level, status, `id_smsc`);
  4. replies `ACK/Jasmin` **only after commit**;
  5. never returns 404, which triggers an immediate purge (J2).
- Duplicate callbacks are harmless by design (rule 24). Missing receipts become `UNKNOWN` after the per-provider window.

### 4.5 The ledger stays authoritative

- **[R]** Credit is reserved, settled and released only in PostgreSQL (financial integrity; ADR-0011), based on Go's message state.
- **[R]** Jasmin's user balance and quota features are configured as unlimited and never read. Jasmin is not told about tenants, prices or balances. **[V]** Those features are in-memory with periodic persistence (§3.3), so they would be unsuitable as a source of truth anyway.

### 4.6 Customer SMPP access

- **[V] Jasmin's SMPP server authenticates Jasmin users.** Letting customers bind to it would bypass Go's identity, authorization, idempotency and ledger.
- **[R] Keep it disabled** or bound to a private interface in Option C. Customer-facing SMPP, if required, is a separate decision: a Go SMPP server, or a Jasmin server whose users are provisioned by Go and whose traffic passes through Go controls. This is a key Option C gap.

### 4.7 Recovery after restarts

| Restart / outage | Expected effect | Go-side control |
| ---------------- | --------------- | --------------- |
| Jasmin process | **[A]** Unacknowledged AMQP messages are redelivered. In-memory retry counters reset (J7). An in-flight `submit_sm` at the moment of crash may be sent twice. | Attempts in `SUBMITTED_TO_ENGINE` older than a threshold without a level-1 DLR → `UNKNOWN`. Never auto-resubmit. |
| RabbitMQ | **[A]** Queued submits and DLRs lost (J3). | Same reconciliation; alert on broker restarts. |
| Redis | **[V]** New submissions lose DLR mapping while Redis is down (J4). **[A]** If Redis restarts without persistence, existing mappings are lost. | Our ID in `dlr-url`; Redis persistence and monitoring; reconciliation. |
| Go DLR endpoint | **[V]** Callbacks retried 3× at 30 s, then purged (J2). | High availability; tuned retries; reconciliation against SMSC-side data if a telco offers it (unconfirmed). |

**Spike results for this table** (§9.3):

- **Jasmin process — [T-PASS] (S4).** Queued and in-flight messages were each submitted once and reported delivered by the simulator, in two sub-cases. A duplicate from a crash during an in-flight `submit_sm` was **not observed**, but this is not proof that it cannot happen.
- **RabbitMQ — [T-FAIL] (S5).** The loss of queued submits was confirmed, and consumers were not recovered.
- **Redis — [T-FAIL] (S6b, S6c).** Receipts were lost both when Redis was down at submit time and when Redis lost its data.
- **Go DLR endpoint — [T-FAIL] (S7c).** The receipt was purged after 4 tries.

### 4.8 Keeping the engine replaceable without over-abstraction

**[R]** Use one Go interface at the existing ADR-0012 seam, e.g. `Submit(ctx, attempt) (EngineRef, Outcome)`, plus an inbound normalized receipt type.

- **The Jasmin adapter** translates to `/send`, and turns callbacks into normalized receipts.
- **A Go-native engine** (Option A) implements the same interface with its own SMPP sessions.

There is no generic plugin system and no "multi-engine router" — one interface, one implementation chosen by configuration, and a shared contract test suite run against the simulator.

---

## 5. Infrastructure implications

*Pre-spike analysis. Jasmin is not adopted (§1.1), so neither RabbitMQ nor Redis is introduced. The points below remain only as the record of what adoption would have required.*

- **[V] Jasmin requires both systems.** It cannot run without RabbitMQ and Redis (§3.1).
- **Redis (ADR-0007).** Used this way, Redis is a **private dependency of the gateway engine**, not an application cache. That does not by itself contradict ADR-0007's "defer Redis for the application" recommendation. ADR-0007's constraint — Redis must never be the source of truth for credits, balances, idempotency or message state — is still satisfied, because Go remains authoritative.
- **Redis holds correlation state** whose loss causes receipt loss (J4). It therefore needs:
  - persistence (AOF/RDB) and memory sizing;
  - monitoring (memory, evictions — an evicted mapping is a lost DLR);
  - restricted network access, and authentication (**[A]** the reference compose sets no Redis password).
- **RabbitMQ (ADR-0006).** RabbitMQ was considered as option E in ADR-0006 and noted as "not in owner's preferences". Using it **only inside the gateway engine** does not change our internal event design: the PostgreSQL outbox (ADR-0020) stays. It does, however, add a broker to operate:
  - cluster or single node;
  - upgrades (the reference image 3.10 is outdated);
  - user and vhost permissions;
  - monitoring (queue depth, unacknowledged messages, consumers, connection churn).

  Because Jasmin's queues are non-durable (J3), backups do not protect in-flight messages.
- **Jasmin itself:** several daemons in the Kubernetes reference (`jasmind`, `interceptord`, `dlrd`, `dlrlookupd`, `deliversmd`) ([pyproject.toml L68][pyproject-scripts]); Python runtime patching; jCli configuration management (pickle stores, which must be backed up); and log management with `log_privacy`.
- **[V] `log_privacy` defaults to `False`** ([managers/configs.py L78][logprivacy]). **[A]** Message content may be logged by default — relevant to RA 10173; must be enabled. **Observed in the spike (S10):** with the default configuration, `messages.log` contained message content in clear text.
- **Adoption is an owner decision.** Adopting RabbitMQ and Redis — even engine-private — is "adopting new infrastructure components" (AGENTS.md §17). Nothing is introduced by this evaluation.

---

## 6. Security and reliability assessment

| Area | Finding | Required control if Jasmin is used |
| ---- | ------- | ---------------------------------- |
| HTTP API authentication | **[V]** `username`/`password` as request parameters; plain HTTP listener, default bind `0.0.0.0` ([http configs L16][http-bind]); no TLS option found. | Private network only (Go → Jasmin). TLS via a sidecar or proxy if the traffic crosses hosts. Dedicated credentials from a secret manager. Never exposed to customers. |
| DLR callbacks to Go | **[A]** No signature or authentication on callbacks (none documented). | Private network, plus an unguessable per-attempt token in `dlr-url` checked by Go. Bind the Go intake to an internal listener. |
| Admin access (jCli) | **[V]** Telnet ([cli/factory.py L10][jcli-telnet]); default `127.0.0.1:8990`, `jcliadmin/jclipwd` ([cli configs L16–20][jcli-cfg]). The reference compose publishes 8990. Affected by the Twisted Telnet DoS advisory GHSA-cwm5-jh59-mwxr. | Loopback only, accessed via an SSH or bastion tunnel. Change the credentials. Never published. |
| SMPP upstream TLS | **[V]** TLS supported, but the context sets no certificate verification ([factory.py L207][ctxfactory]) — encrypted but unauthenticated. | Prefer telco private links or VPN (telco checklist). Patch verification if TLS is used over untrusted networks. |
| Internal message format | **[V]** AMQP bodies and configuration stores are Python pickle ([listeners.py L138][pickle-amqp], [throwers.py L241][pickle-dlr]). | Anyone able to publish to that RabbitMQ can execute code in Jasmin: a dedicated broker or vhost, strong credentials, no other producers, network isolation. |
| Code execution by design | **[V]** Interceptors and `EvalPyFilter` run operator-supplied Python ([Filters.py L250][evalpy]). | Admin-only; configuration under change control and review. |
| Dependency vulnerabilities | **[V]** OSV lists 15 advisories for Twisted 23.10.0, including HTTP request smuggling (GHSA-q7qq-6mhp-9rc4) and pipelining (GHSA-c8m8-j448-xjx7), both relevant to the HTTP API. It also lists advisories for the `requests`, `cryptography` and `pyOpenSSL` versions pinned on `master`. The `~=23.10.0` pin cannot receive Twisted fixes. | Rebuild with patched dependencies (a fork) and run our own scanning. Treat as a blocker for internet-adjacent use. |
| Tenant isolation | Jasmin has no knowledge of our tenants (by design in §4). | Tenant data stays in Go. Jasmin sees only destination, content, sender ID and opaque tags. |
| Abuse prevention | Sender ID approval, content policy and rate limits per tenant live in Go. **[V]** Jasmin has per-user throughput quotas and filters, used only as a safety net. | Go enforces policy before submission. |

**Reliability summary.** As shipped, Jasmin's submission behaviour at the SMPP submission boundary is at-least-once: a `submit_sm` whose response times out is submitted again (J1). It is also lossy for receipts and queued messages on failures (J2–J4). Our rules require the opposite at the SMPP submission boundary: never resubmit an ambiguous submission, and never lose evidence silently. Go-side reconciliation can detect losses; it cannot undo a repeated submission, or any duplicate SMS that may result from it. Whether repeated submissions actually reach handsets as duplicate SMS was not tested (§9.5). **This is why J1 is the decisive spike criterion.**

**Spike confirmation (S8, S10).** J1 was confirmed as repeated `submit_sm` PDUs after an ambiguous timeout, and the repeated submissions were invisible to our platform. Whether they would reach recipients as duplicate SMS was not tested (simulator only). The finding remains a production blocker.

In the official image and reference setup, the spike also observed these defaults:

- jCli on `0.0.0.0` with the published default credentials (the source default is loopback, see above);
- RabbitMQ `guest/guest` with administrator rights;
- Redis without a password;
- a plain-HTTP API.

Invalid credentials were rejected. The other rows in the table above come from the source, documentation and OSV review only and were not tested (§9.5).

---

## 7. License and maintenance assessment

- **[V] License: Apache-2.0** ([LICENSE][license]).
  - Permissive; patent grant included.
  - Modification and private forks are allowed, with NOTICE and attribution obligations.
  - Running it as a service imposes no copyleft obligations.
  - **[A]** Dependency licences were not audited one by one; to be done before adoption.
- **[V] Release cadence:** 0.10.12 (2022-09), 0.11.0 (2023-11), PyPI 0.11.1 (2024-03); nothing since.
- **[V] Commit activity:** 176 commits in 2022, 70 in 2023, 21 in 2024, 1 in 2025, and 29 in 2026 (all in April, by a small group). Latest commit 2026-04-26.
- **[V] GitHub:** 40 open issues, 5 open pull requests, about 1.2k stars, 610 forks; no published security advisories.
- **[V] `SECURITY.md` is stale:** it still lists only 0.10.x as supported ([SECURITY.md][security]). Vulnerability reports go to support@jasminsms.com with a "48 hours" response statement.
- **[V] Docs are partly stale:** the installation page still says it pulls "latest jasmin v0.10" ([installation][docs-install]).
- **[V] Support:** community support via GitHub issues and a Google Group. Commercial support, hosting and consulting are offered by Jookies ([support][docs-support]). **[A]** Terms, SLAs and cost are unknown.
- **Relevance after the decision:** Apache-2.0 would govern any **future** selective reuse of Jasmin source. Such reuse is not authorized now (§1.1); it would need a check of the exact revision, NOTICE and attribution obligations, and the dependencies it brings.
- **[R] Assessment (pre-spike):** usable as a starting point, but adopting it means **we** carry security patching, dependency upgrades (e.g. a Twisted major upgrade) and any reliability patches. Budget for that as owning a fork.

---

## 8. Risks and unknowns

The "How resolved" column is the pre-spike plan, which used the originally proposed scenario numbers. The last column records the outcome.

| # | Risk / unknown | Type | How resolved (pre-spike plan) | Outcome after the spike |
| - | -------------- | ---- | ----------------------------- | ----------------------- |
| R1 | J1 timeout resubmission causes duplicate SMS | Verified behaviour | Spike S5; if confirmed, a patch decision or Option A | **[T-FAIL] Confirmed** as duplicate upstream submission attempts (executed as S8). Duplicate delivery to handsets **[NT]**. One of the reasons for rejection. |
| R2 | Receipt and queue loss on restarts or callback outages (J2–J4, J6) | Verified behaviour, effect assumed | Spike S6–S8 | **[T-FAIL] Confirmed** for J2–J4 (S5, S6b, S6c, S7c), plus the Unicode-receipt failure (S9b/c). J6 **[NT]**. |
| R3 | Throughput unknown; no upstream benchmark | Unknown | Separate load test after functional criteria pass, against verified requirements | **[NT]** No longer relevant for Jasmin. Capacity must still be measured for any gateway. |
| R4 | Maintenance velocity and stale dependency pins | Verified | Owner decision on carrying a fork, or on commercial support | Unchanged. One reason the required patches are not carried (§9.7). |
| R5 | Compatibility with supported RabbitMQ/Redis versions | Assumption | Spike runs current supported versions, pinned | **[T-FAIL]** RabbitMQ 4.4.1 refuses Jasmin's queues by default and works only with a temporary permit. **[T-PASS]** Redis 8.12.0 worked. RabbitMQ 3.x and other 4.x releases **[NT]**. |
| R6 | Telco SMPP specifics (TON/NPI, encodings, receipt formats, binds, throughput) | Unknown — no telco specs | Telco discovery checklist; never invented | Unchanged. Applies equally to the Go-owned gateway. |
| R7 | DLR expiry (24 h default) versus telco receipt latency | Unknown | Telco answers; set `dlr_expiry` accordingly | No longer relevant for Jasmin. Receipt latency is still a telco question for the Go gateway. |
| R8 | Query-string preservation in `dlr-url` | Assumption | Spike S4 | **[T-PASS]** Preserved in every callback (executed as S3). |
| R9 | Customer-facing SMPP requirement and design | Open business question | Owner decision (§4.6) | Unchanged: an open business question. Jasmin's SMPP server **[NT]**. |
| R10 | Hidden cost of operating Python + RabbitMQ + Redis with a small team | Assumption | Owner staffing decision | No longer relevant: not adopted. |

---

## 9. Technical spike (completed 2026-11-25)

### 9.1 Scope as approved and executed

- **Approval.** The owner approved the spike on 2026-11-25, strictly time-boxed to 3 working days. All runs took place between 14:22 and 15:04 UTC the same day.
- **Isolation.** The spike ran outside the product repository. No product code, migration, ADR, dependency or `main` was modified.
- **Excluded.** No real telco, no real phone numbers, no load or throughput test.
- **Scenarios.** The owner approved a list of ten scenarios that replaced the list originally proposed in this section. The owner also requested confirmation runs:
  - S3 rerun, with SMSC message IDs `"0"` and unique non-zero IDs;
  - S9 rerun, isolated after a clean reset with verified consumer health;
  - S6 rerun.
- **Results** were recorded as PASS, FAIL or INCONCLUSIVE, independently per scenario and sub-case.

**Mapping of the originally proposed criteria to what was executed:**

| Originally proposed | Executed as | Notes |
| ------------------- | ----------- | ----- |
| S1 one `submit_sm` | S1 | — |
| S2 DLR to Go | S2 | — |
| S3 forced duplicate DLR callback | — | **[NT]** Not in the approved list |
| S4 `/send` response lost; Go marks uncertain and does not resend | S3 (correlation via the `dlr-url` reference) | **[NT]** The lost-response case itself was not tested; reference preservation was |
| S5 SMPP timeout (blocking criterion) | S8 | **[T-FAIL]** |
| S6 Jasmin restart | S4 | — |
| S7 RabbitMQ and Redis restarts | S5, S6 | — |
| S8 Go DLR endpoint down | S7 | — |
| S9 multipart and UCS2 | S9 | — |
| S10 security posture | S10 | Default exposure was **observed**; a hardened, network-isolated configuration was **[NT]** |

### 9.2 Environment

The stack ran under isolated Docker Compose. A Go test harness drove it; it used only the standard library and was not part of the product module. The harness contained:

- an SMPP proxy that counts every `submit_sm` and injects faults;
- a jCli client and a Jasmin HTTP client;
- a DLR receiver backed by a Go-side authoritative message record.

| Component | Tag | Digest |
| --------- | --- | ------ |
| Jasmin (latest tagged release) | `jookies/jasmin:0.11.0` | `sha256:3f049692d22fd66ab08a55073f79db96fe442473ede9615e8ac085ac505a1064` |
| SMPP simulator (upstream) | `jookies/smppsim:1.0.1` | `sha256:2f775f7ee8d3d8857e711d75bbb8e1188874f50eaf0b85973405cb7a122561b6` |
| RabbitMQ | `rabbitmq:4.4.1-management-alpine` (from the `mirror.gcr.io/library/` mirror) | `sha256:ad4bab5e9f3a2651ef2e0533721e5a52864e884e27986c4e02967ebd7e96f387` |
| Redis | `redis:8.12.0-alpine` (from the same mirror) | `sha256:5af80107c1eeecb484a2269b80a26d12bfa6521787da4663bd9f19f60a163483` |

RabbitMQ and Redis were pulled from Google's Docker Hub mirror because Docker Hub rate-limited anonymous pulls. The digests above identify the exact images.

**Configuration (spike only):**

- **Simulator:** configured to report 100% of messages as delivered, with fast receipts. `SMPP_CONNECTION_HANDLERS` was raised from 1 to 5 for the confirmation runs (§9.4).
- **Jasmin connector:** `res_to 10` and `requeue_delay 20`, shortening the default 120 s timers. This changes only timing: with the defaults the code path is the same, with about 4 minutes between repeated submissions.
- **RabbitMQ 4.4.1 by default:** **[T-FAIL]** Jasmin cannot declare its queues. RabbitMQ refuses with `Feature 'transient_nonexcl_queues' is deprecated … not permitted anymore`, and `RouterPB` and other services fail to start.
- **RabbitMQ 4.4.1 with a workaround:** with `deprecated_features.permit.transient_nonexcl_queues = true`, Jasmin starts. RabbitMQ states that the permit is temporary and the feature will be removed. **All other results were obtained with this permit.**

### 9.3 Confirmed results

"Delivered" in this table means the SMPP simulator reported delivery. No real mobile network or handset was involved (§1.2).

| # | Scenario | Result | Evidence summary |
| - | -------- | ------ | ---------------- |
| S1 | Submission Go → Jasmin → simulator | **[T-PASS]** | `Success "<uuid>"`; exactly 1 `submit_sm` at the proxy; `submit_sm_resp` `ESME_ROK` |
| S2 | Delivery receipt returned to Go | **[T-PASS]** | Level-1 (`ESME_ROK`) and level-2 (`DELIVRD`, generated by the simulator) callbacks; the Go record reached delivered |
| S3 | Correlation with unique non-zero SMSC IDs | **[T-PASS]** | SMSC IDs 1, 2, 3 arrived as `id_smsc` 1, 2, 3. Our reference (in `dlr-url`) and the Jasmin ID were present in every callback; this confirms the §4.3 assumption for the tested setup |
| S3-edge | Correlation with SMSC message ID `"0"` | **[T-FAIL]** (edge case) | Reproduced twice: `"0"` arrives as an **empty** `id_smsc`. Correlation by our reference still worked |
| S4 | Jasmin restart with messages queued or in flight | **[T-PASS]** | (a) 10 queued, (b) 20 throttled with a mid-stream restart: each submitted once, all reported delivered by the simulator |
| S5 | RabbitMQ restart with messages queued | **[T-FAIL]** | All queues `durable=false`.<br>- After the restart every Jasmin queue was gone: **10 of 10 queued messages lost**, and none reached the simulator.<br>- Consumers were not recovered.<br>- A new `/send` returned **HTTP 200 `Success "[Failure instance: … ChannelClosed … NOT_FOUND - no queue 'submit.sm.sim']"`** |
| S6a | Redis graceful restart between submit and receipt | **[T-PASS]** | Receipt correlated and delivered to Go |
| S6b | Redis data lost between submit and receipt | **[T-FAIL]** | Message submitted once and reported delivered by the simulator; **the receipt never reached Go** |
| S6c | Redis down at submit time | **[T-FAIL]** | Message submitted once to the simulator. Jasmin logged `DLR is not enqueued … RC is not connected`; **the receipt never reached Go** |
| S7a | Callback endpoint returning an HTTP error for 10 s | **[T-PASS]** for endpoint recovery; **[T-INC]** for callback-sequence completeness | Jasmin retried, the endpoint recovered, and the final `DELIVRD` callback arrived. Go recorded only **1 of the 2** expected callbacks (levels 1 and 2); the missing callback is unexplained (§9.4). Not a complete DLR-delivery pass |
| S7b | Callback endpoint down for 45 s | **[T-PASS]** for endpoint recovery; **[T-INC]** for callback-sequence completeness | Connection refused ×3, then the endpoint recovered and the final `DELIVRD` callback arrived. Go recorded only **1 of the 2** expected callbacks; the missing callback is unexplained (§9.4). Not a complete DLR-delivery pass |
| S7c | Callback endpoint down for 150 s | **[T-FAIL]** | 4 attempts over about 65 s, then `Message try-count is 4 … purged from queue`. The simulator reported the message delivered; **Go never learns it** |
| S8 | SMPP submission timeout | **[T-FAIL]** — confirmed reliability failure and production blocker | Duplicate upstream submission attempts:<br>(a) 1 withheld response: **2 `submit_sm` PDUs** for one message, given two different SMSC IDs by the simulator.<br>(b) 3 withheld: **4 `submit_sm` PDUs**.<br>- Jasmin logged `request timed out … message requeued`.<br>- The Go record showed **one** SMSC ID per message, so **the repeated submissions were invisible to the platform**.<br>- **[NT]** Actual duplicate delivery to recipient handsets was not tested.<br>- Each timeout also re-bound the session, delaying other traffic by about 30 s |
| S9a | Multipart GSM (250 characters), isolated after a clean reset | **[T-PASS]** | 2 parts with UDHI, `data_coding` 0; receipt delivered; consumers healthy for 120 s |
| S9b | Short Unicode (UCS2), isolated after a clean reset | **[T-FAIL]**, reproducible | Encoding was correct: 1 part, `data_coding` 0x08.<br>- About 45 s later the AMQP connection broke, and consumers were not restored within 120 s.<br>- The final receipt was lost.<br>- Reproduced in 2 of 2 isolated runs, and in the earlier runs |
| S9c | Unicode multipart (120 characters), isolated after a clean reset | **[T-FAIL]** | Segmentation and encoding were correct: 2 parts, UDHI, `data_coding` 0x08.<br>- The AMQP connection broke, and consumers were not restored within 120 s.<br>- The final receipt was lost (1 isolated run) |
| S10 | Invalid credentials and unauthorized access | **[T-PASS]** for rejection, with findings | **Rejected:** a wrong password or unknown user returned HTTP 403 `Error "Authentication failure …"`; no credentials returned HTTP 400; jCli rejected a wrong password.<br>**Findings in the official image and reference setup:**<br>- jCli accepts the published defaults `jcliadmin/jclipwd` and binds `0.0.0.0`;<br>- RabbitMQ `guest/guest` has administrator rights from another container;<br>- Redis answers without a password;<br>- the HTTP API is plain HTTP;<br>- `messages.log` contains message content in clear text |

**Totals over all messages submitted.**

- **Repeated upstream submissions:** only in S8, with 5 extra `submit_sm` PDUs across 2 messages. None in S1–S7 or S9. Whether any would reach a handset as a duplicate SMS was not tested.
- **Lost messages** (never submitted): 10, all in S5.
- **Lost receipts** for messages the simulator had accepted: S6b, S6c and S7c (one each), S9b (twice in isolated runs), and S9c (once).

**S9 chain of events, as far as the evidence shows.** This is a factual description, not a fix. It repeated in every failing run:

1. The simulator returns the level-2 receipt for a UCS2 message.
2. Jasmin publishes the receipt to AMQP, with the receipt text in the message's header table.
3. RabbitMQ 4.4.1 rejects the frame (`rabbit_binary_parser:parse_table` error) and closes Jasmin's connection.
4. Jasmin reconnects, but logs `Channel open failed: … AlreadyCalledError` and never re-creates its consumers.
5. The HTTP API keeps answering `Success`, but nothing is submitted and no receipts are thrown until Jasmin is restarted manually. A restart restored the consumers.

### 9.4 Inconclusive results and harness issues

**Inconclusive results:**

- **S9 root cause — [T-INC].** Two questions remain open: whether the frame is malformed by Jasmin/txAMQP or only rejected by stricter parsing in RabbitMQ 4.4.1, and whether RabbitMQ 3.x or other 4.x releases accept it. Neither was tested.
- **S7 "404 → immediate purge" — [T-INC].** Jasmin reported our 404 as a generic `HttpApiError()`, not the `"404"` string its no-retry rule matches. The documented immediate-purge behaviour (J2) was therefore **not observed** in this setup.
- **S7a/S7b callback-sequence completeness — [T-INC], unexplained.** In both cases the endpoint recovered and the final `DELIVRD` arrived, but Go recorded only 1 of the 2 expected callbacks (levels 1 and 2). Jasmin logs both levels under one message ID and keys its retry counter by that ID, so it could not be determined which callback was missing or why. Endpoint recovery is therefore counted as a pass, but neither case is a complete DLR-delivery pass.

**Harness defects.** These were found during the spike and handled; their results are **not counted**:

- **Main-run S6a/S6b — [T-INC], superseded.** The simulator allowed only one bind, so the connector was not bound for about 90 s and the cases were scored before submission. After the fix (5 handlers and a bind check), S6 was rerun cleanly; §9.3 shows the rerun.
- **First confirmation batch for S9 — [T-INC], superseded.** A health gate ran before the connector's consumer appeared. After the fix (the gate now waits up to 30 s), all four runs were repeated.
- **Overwritten C09b evidence file.** The second C09b run reused the same ID and overwrote the first run's summary file. The first run's verdict line and its complete service logs are preserved.
- **Exploratory first run.** It gave the first observation of the S9 failure. It is not counted and is superseded by the isolated runs.

### 9.5 Not tested

| Item | Status |
| ---- | ------ |
| Any real telco (Smart, Globe, PLDT, DITO): receipt formats, message-ID formats, latency, throughput | **[NT]** Simulator only |
| Load and throughput | **[NT]** No capacity claim is made (§3.4 still applies) |
| RabbitMQ 3.x (the reference compose uses 3.10), RabbitMQ 4.x releases other than 4.4.1, later Jasmin commits, patched or hardened builds | **[NT]** §1.2 |
| Actual delivery to recipient handsets, including whether repeated `submit_sm` PDUs (S8) would produce duplicate SMS | **[NT]** Simulator only |
| SMPP TLS to upstream (the review found no certificate verification, §6) | **[NT]** |
| IPv6 | **[NT]** |
| Jasmin's SMPP server for customers (§4.6), and the REST API (Celery) | **[NT]** |
| Forced duplicate DLR callbacks, and a lost `/send` response (originally proposed S3/S4) | **[NT]** |
| Overaged messages discarded for a disconnected connector (J6) | **[NT]** |
| Retry counters reset by a restart during DLR retries (J7) | **[NT]** |
| Pickle-based AMQP and configuration stores, interceptors and `EvalPyFilter` code execution, Twisted advisories, `log_privacy=True` behaviour | **[NT]** Source, documentation and OSV review only (§6) |
| A hardened, network-isolated deployment (the original S10 criterion) | **[NT]** The spike observed the defaults |

Management-API message counts lag. Queue durability and consumer counts were therefore read from the API, and message loss was determined from the proxy counts and Jasmin's logs.

### 9.6 Classification of the issues

| Issue | Fixable by configuration | Needs a maintained patch | Fundamental limitation |
| ----- | :----------------------: | :----------------------: | :--------------------: |
| RabbitMQ 4.4.1 refuses Jasmin's queues (§9.2) | Temporary permit only | ✔ durable queue declaration | |
| A Unicode receipt breaks AMQP; consumers not restored (S9b/c) | Unknown (RabbitMQ 3.x and other 4.x releases untested) | ✔ frame encoding and reconnect logic | |
| Timeout → automatic resubmission → duplicate upstream submission attempts (S8) | ✘ timers change only timing | ✔ the requeue-on-timeout path | ✔ the repeated submissions are invisible to the platform |
| Queued messages lost on a broker restart; no consumer recovery; failure reported as `Success` (S5) | ✘ | ✔ durability, recovery, error handling | |
| Receipt purged after bounded retries (S7c) | Partly: larger `max_retries`/`retry_delay`, a highly available endpoint | ✔ dead-letter instead of purge | Bounded retries by design |
| Redis data loss loses receipts (S6b) | ✔ Redis persistence and high availability | | |
| Redis down at submit loses receipts (S6c) | Partly (Redis high availability) | ✔ the skip-with-warning path | |
| SMSC ID `"0"` → empty `id_smsc` (S3-edge) | | ✔ minor | |
| Default credentials, jCli on `0.0.0.0`, RabbitMQ guest administrator, Redis without authentication, plain HTTP, content in logs (S10) | ✔ hardening plus network isolation | | |

### 9.7 Spike recommendation and owner decision

**Spike recommendation (exactly one, as the owner required):** reject Jasmin, and implement the required gateway functionality in Go behind the ADR-0012 provider adapter.

- **Why not adopt with mitigations.** At least four code patches would have to be carried and maintained in a Python/Twisted codebase with stale pins and a slow upstream (§7):
  - resubmission on timeout;
  - queue durability and recovery;
  - AMQP frame and reconnect handling for Unicode receipts;
  - Redis-down handling.

  Our core rules — no blind resubmission after an ambiguous outcome (the rule that guards against duplicate SMS), and no silent loss — would then depend on that fork.
- **Why not hybrid (Option C with Jasmin).** The defects are in the SMPP dispatch and receipt path. That is exactly what a hybrid would keep Jasmin for, so a hybrid does not avoid them.

**Owner decision (2026-11-25):** the tested Jasmin 0.11.0 deployment is rejected for production adoption, with the direction recorded in §1.1.

**What carries forward:**

- the gateway-neutral boundary: our reference travels in receipts, Go owns the authoritative record, and uncertain states are explicit;
- the spike's fault-injecting SMPP proxy and simulator setup, kept outside the repository as a basis for testing our own SMPP implementation.

### 9.8 Evidence

The evidence is kept **outside this repository**, by owner decision, together with the harness. It contains:

- the consolidated report;
- image tags and digests;
- the RabbitMQ 4.4.1 default-mode failure log;
- per-scenario files for the main run: procedure, expected and actual results, PDU logs with timestamps, DLR callbacks, Go records, verdicts;
- service logs for the main run and for every confirmation run, each taken after a clean reset;
- the superseded first S9 confirmation batch.

---

## 10. ADRs and decisions affected (no ADR changed)

No ADR status or text was changed by this evaluation or by the owner decision. Current statuses come from the [decision log](../engineering/decision-log.md):

- ADR-0006, ADR-0007, ADR-0009 and ADR-0012 are **Proposed**;
- ADR-0021 is **Accepted — limited scope**.

The table below replaces the pre-spike list "ADRs that may need revision". It records what the outcome means for each ADR, for use in the separate gateway architecture decision.

| ADR | Relevance after the decision |
| --- | ---------------------------- |
| ADR-0012 Provider abstraction | The Go-owned gateway sits behind this adapter boundary. Its option C ("third-party gateway software") now has this evidence against the tested Jasmin 0.11.0 deployment. The SMPP-library selection the ADR anticipates is **still open**, and is a separate owner decision. |
| ADR-0006 Broker, ADR-0007 Redis | **No change needed.** The engine-private RabbitMQ and Redis cases discussed in §5 arose only if Jasmin were adopted. Neither is introduced. |
| ADR-0021 Message status model | **No change needed** for the accepted customer-visible statuses. The spike confirms why `submission_uncertain` and `unknown` must exist: repeated upstream submissions and lost receipts can occur below the platform. Internal engine milestones are a design detail of the future gateway. |
| ADR-0009 Control/data plane | **No change needed.** The data plane stays in Go; the question raised in the pre-spike list no longer arises. |
| Future ADR | **"SMS gateway engine and SMPP library"** — to record the Go-owned gateway decision, its architecture and the library choice when gateway development is scheduled. It will cite this document. It is not written now. |

---

## 11. Direction and next steps

- **In force:** §1.1 (owner decision, 2026-11-25).
- **Not started:** gateway implementation, SMPP library selection, gateway packages and ADR changes. They wait for a separate architecture decision when the roadmap reaches gateway development.
- **Unaffected:** product development continues on the roadmap. Groups 1–3 (Go API skeleton, PostgreSQL foundation, OpenAPI contract) are merged, and the public API contract is gateway-independent ([`contracts/openapi/`](../../contracts/openapi/README.md)).
- **No further Jasmin work** is authorized.
<!-- Sources (Jasmin at commit 0aac58e466d583d0f0436df7b8afa3dc96191263, read 2026-11-25) -->
[repo]: https://github.com/jookies/jasmin
[docs]: https://docs.jasminsms.com/en/latest/
[head]: https://github.com/jookies/jasmin/tree/0aac58e466d583d0f0436df7b8afa3dc96191263
[license]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/LICENSE
[security]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/SECURITY.md
[compose]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/docker-compose.yml#L38
[pyproject-version]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/pyproject.toml#L7
[pyproject-scripts]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/pyproject.toml#L68
[pyproject-python]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/pyproject.toml#L10
[pyproject-twisted]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/pyproject.toml#L31
[smpp-ssl]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/protocols/smpp/factory.py#L148
[ctxfactory]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/protocols/smpp/factory.py#L207
[smpps-bind]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/protocols/smpp/configs.py#L244
[dlr-expiry]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/protocols/smpp/configs.py#L92
[requeue-delay]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/protocols/smpp/configs.py#L180
[http-bind]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/protocols/http/configs.py#L16
[longsplit]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/protocols/http/configs.py#L34
[failover]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/protocols/http/endpoints/send.py#L233
[bestquality]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/routing/Routes.py#L371
[evalpy]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/routing/Filters.py#L250
[tagfilter]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/routing/Filters.py#L272
[persist-timer]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/routing/configs.py#L22
[pickle-store]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/protocols/cli/httpccm.py#L110
[timeout-requeue]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/managers/listeners.py#L277
[expired]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/managers/listeners.py#L186
[overaged]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/managers/listeners.py#L196
[pickle-amqp]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/managers/listeners.py#L138
[throwers]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/routing/throwers.py
[pickle-dlr]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/routing/throwers.py#L241
[dlr-noretry]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/routing/throwers.py#L626
[dlr-purge]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/routing/throwers.py#L638
[amqp-down]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/managers/clients.py#L567
[redis-skip]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/managers/clients.py#L599
[queue-msgid]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/managers/dlr.py#L254
[queue-declare]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/queues/factory.py#L200
[publish]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/queues/factory.py#L221
[jcli-cfg]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/protocols/cli/configs.py#L16
[jcli-telnet]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/protocols/cli/factory.py#L10
[logprivacy]: https://github.com/jookies/jasmin/blob/0aac58e466d583d0f0436df7b8afa3dc96191263/jasmin/managers/configs.py#L78
[docs-index]: https://docs.jasminsms.com/en/latest/index.html
[docs-install]: https://docs.jasminsms.com/en/latest/installation/index.html
[docs-http]: https://docs.jasminsms.com/en/latest/apis/http/index.html
[docs-rest]: https://docs.jasminsms.com/en/latest/apis/rest/index.html
[docs-jcli-modules]: https://docs.jasminsms.com/en/latest/management/jcli/modules.html
[docs-billing]: https://docs.jasminsms.com/en/latest/billing/index.html
[docs-interception]: https://docs.jasminsms.com/en/latest/interception/index.html
[docs-support]: https://docs.jasminsms.com/en/latest/support/index.html
