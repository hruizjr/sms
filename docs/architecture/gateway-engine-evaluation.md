# SMS gateway engine evaluation — Jasmin vs. a custom Go gateway

| Field | Value |
| ----- | ----- |
| Status | **Evaluation for owner decision — nothing adopted.** No ADR status changed; no code, dependency, or infrastructure added. |
| Date | 2026-11-25 |
| Subject | Jasmin SMS Gateway ([repository][repo], [documentation][docs]) as the SMS gateway engine under our Go platform |
| Evidence base | Jasmin source at commit [`0aac58e`][head] (`master`, 2026-04-26), PyPI, Docker Hub, the official docs site, and the OSV vulnerability database, all read on 2026-11-25 |

Labels used throughout: **[V]** verified fact with a cited source · **[A]** assumption or inference that still needs testing · **[R]** recommendation.

---

## 1. Executive recommendation

**[R] Do not adopt Jasmin as the gateway engine yet. Build to the Option C boundary, and decide with a short spike.**

1. **Ownership stays in Go.** Our Go platform keeps everything business-related: acceptance, the authoritative message record, the ledger, routing policy, uncertain-outcome handling, and reconciliation. This holds whichever engine sits underneath.
2. **One narrow boundary.** The gateway engine sits behind one Go interface — the provider/gateway adapter from ADR-0012.
3. **Jasmin is a candidate engine, not a decision.** The source code shows four behaviours that conflict with our non-negotiable messaging rules if used as shipped:
   - **Resubmits on timeout.** A timed-out `submit_sm` is automatically resubmitted. This risks duplicate SMS to real people (AGENTS.md rule 23).
   - **Drops receipts.** Delivery receipts are purged after 3 failed callbacks.
   - **Loses data on broker restart.** Queues and messages are declared non-durable.
   - **Can lose receipts silently.** If Redis is down at submit time, the receipt mapping is skipped with only a warning.

   Section 3 has the sources for each.
4. **Maintenance and security posture are weak.**
   - **Releases:** the last release was 0.11.1 in March 2024.
   - **Twisted:** the pinned Twisted 23.10 has open advisories, including HTTP request smuggling.
   - **Admin access:** the admin console uses telnet.
   - **Message format:** internal messages are Python `pickle` objects.
5. **No published throughput figures exist.** Nothing supports the reported 5-billion-per-month figure, or any figure.

The spike (section 9) is designed to test exactly these risks. **[R]** Adopt Jasmin (Option C, Jasmin variant) only if its acceptance criteria pass with configuration and Go-side controls alone, or with a small patch you approve. Otherwise, build the dispatch and SMPP layer in Go (Option A) behind the same adapter. The adapter boundary means the work in Groups 1–2 and the Go business layer are unaffected either way.

---

## 2. Comparison of options

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

---

## 3. Jasmin capabilities and limitations

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
| J1 | **Timeout resubmission.** On `SMPPRequestTimoutError` (no `submit_sm_resp`), the message is requeued and **sent again** after `requeue_delay` (default 120 s) ([listeners.py L277][timeout-requeue], [configs L180][requeue-delay]). No retry cap exists on this path; it stops only if the optional `expiration` (from `validity-period`) has passed ([L186][expired]). | AGENTS.md rule 23 (ambiguous outcomes are never blindly retried); duplicate SMS to real people | **[A] Probably not** with configuration alone. `validity-period` only bounds the window. Needs a code change (fork or upstream patch) or proof from the spike that a timeout never leads to a second `submit_sm`. **Blocker until resolved.** |
| J2 | **DLR purge.** A DLR callback that fails `max_retries` (default 3, delay 30 s), or receives HTTP 404, is "purged from queue" — rejected with no dead-letter queue ([throwers.py L626][dlr-noretry], [L638][dlr-purge]). | Rule 24 (receipts may be late, but must not be silently lost); Go DLR endpoint outages | **Partly.** Raise `max_retries` and `retry_delay`, make the Go endpoint highly available, never return 404, and reconcile. Loss is still possible during long outages. |
| J3 | **Non-durable queues and messages.** Queues are declared without `durable`, and messages are published without a persistent delivery mode (no such setting anywhere in the source; [factory.py L200][queue-declare], [L221][publish]). **[A]** By AMQP semantics, a RabbitMQ restart loses queued submits and DLRs. | ADR-0020 (no silent loss); recovery requirements | **Partly.** Go reconciliation can detect missing outcomes and mark them `UNKNOWN`. True durability needs a code change. To be confirmed in the spike. |
| J4 | **Redis-down mapping skip.** If Redis is disconnected when a message is submitted, it is still sent, but the DLR mapping is skipped with only a warning ([clients.py L599][redis-skip]). | Rule 24; silent loss of correlation | **Partly.** Correlate in Go using our own ID in `dlr-url` (§4.3), monitor Redis, and alert. |
| J5 | **Broker-down submit refusal.** If AMQP is disconnected, submission returns failure ([clients.py L567][amqp-down]). | — | Acceptable: Go sees a clear error and keeps the message queued on our side. |
| J6 | **Silent discard.** Messages waiting for a disconnected connector are discarded when older than `submit_max_age_smppc_not_ready` ([listeners.py L196][overaged]). **[A]** Whether the originating application receives a failure receipt was not confirmed. | Rule 24 / observability | To be tested in the spike. |
| J7 | **In-memory retry counters.** Retry counters and requeue timers are held in memory ([throwers.py][throwers]) and reset on restart. | Predictable retry limits | Bounded by J2 mitigations; observe in the spike. |

---

## 4. Integration architecture (if Jasmin is used, Option C)

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
- **Engine-side duplicates are J1** — a blocker for Jasmin until the spike resolves it.

### 4.3 Message ID mapping

- **[R] Our ID travels with the message.** Go generates its own message ID and attempt ID and puts them in the per-message `dlr-url` query string (for example `…/internal/dlr/jasmin?attempt=<id>`).
- **Every DLR is self-correlating to our record**, even if the `/send` response was lost. **[A]** This assumes Jasmin preserves the query string when appending DLR parameters; verify in the spike.
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

### 4.8 Keeping the engine replaceable without over-abstraction

**[R]** Use one Go interface at the existing ADR-0012 seam, e.g. `Submit(ctx, attempt) (EngineRef, Outcome)`, plus an inbound normalized receipt type.

- **The Jasmin adapter** translates to `/send`, and turns callbacks into normalized receipts.
- **A Go-native engine** (Option A) implements the same interface with its own SMPP sessions.

There is no generic plugin system and no "multi-engine router" — one interface, one implementation chosen by configuration, and a shared contract test suite run against the simulator.

---

## 5. Infrastructure implications

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
- **[V] `log_privacy` defaults to `False`** ([managers/configs.py L78][logprivacy]). **[A]** Message content may be logged by default — relevant to RA 10173; must be enabled.
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

**Reliability summary.** As shipped, Jasmin's delivery semantics are at-least-once at the SMSC (J1) and lossy for receipts and queued messages on failures (J2–J4). Our rules require the opposite at the SMSC boundary: never resubmit an ambiguous submission, and never lose evidence silently. Go-side reconciliation can detect losses; it cannot undo a duplicate SMS. **This is why J1 is the decisive spike criterion.**

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
- **[R] Assessment:** usable as a starting point, but adopting it means **we** carry security patching, dependency upgrades (e.g. a Twisted major upgrade) and any reliability patches. Budget for that as owning a fork.

---

## 8. Risks and unknowns

| # | Risk / unknown | Type | How resolved |
| - | -------------- | ---- | ------------ |
| R1 | J1 timeout resubmission causes duplicate SMS | Verified behaviour | Spike S5; if confirmed, a patch decision or Option A |
| R2 | Receipt and queue loss on restarts or callback outages (J2–J4, J6) | Verified behaviour, effect assumed | Spike S6–S8 |
| R3 | Throughput unknown; no upstream benchmark | Unknown | Separate load test after functional criteria pass, against verified requirements |
| R4 | Maintenance velocity and stale dependency pins | Verified | Owner decision on carrying a fork, or on commercial support |
| R5 | Compatibility with supported RabbitMQ/Redis versions | Assumption | Spike runs current supported versions, pinned |
| R6 | Telco SMPP specifics (TON/NPI, encodings, receipt formats, binds, throughput) | Unknown — no telco specs | Telco discovery checklist; never invented |
| R7 | DLR expiry (24 h default) versus telco receipt latency | Unknown | Telco answers; set `dlr_expiry` accordingly |
| R8 | Query-string preservation in `dlr-url` | Assumption | Spike S4 |
| R9 | Customer-facing SMPP requirement and design | Open business question | Owner decision (§4.6) |
| R10 | Hidden cost of operating Python + RabbitMQ + Redis with a small team | Assumption | Owner staffing decision |

---

## 9. Proposed technical spike (not started)

**Scope:** one isolated, throwaway environment (a separate directory or repository, not merged into `main`). It contains:
- our Go API (unchanged) plus a minimal throwaway spike adapter;
- Jasmin from a pinned commit or image, a pinned supported RabbitMQ, and a pinned Redis;
- an SMPP simulator — `jookies/smppsim`, the image used by Jasmin's own Kubernetes manifests (Docker Hub, 2022-08-07), or another local simulator.

**Excluded:** no real telco, no real phone numbers, no business tables merged, no new dependency in the product.

**Duration:** time-boxed. **[R]** 3–5 working days, then a written result.

| # | Scenario | Acceptance criterion |
| - | -------- | -------------------- |
| S1 | Go submits; Jasmin queues; the simulator receives exactly one `submit_sm`. | Go record: accepted → submitted-to-engine with the Jasmin UUID stored; the simulator log shows one PDU. |
| S2 | The simulator returns a delivery receipt. | Go receives level-1 and level-2 callbacks, correlates them to the **Go** message, stores them append-only, and acknowledges only after commit. The final status is set by Go. |
| S3 | Duplicate DLR callback (forced). | One status transition, one ledger effect. The duplicate is recorded as a duplicate. |
| S4 | `/send` response lost (Go times out after Jasmin queued). | Go marks `SUBMIT_UNCERTAIN` and does **not** resend. The later DLR still correlates via the `dlr-url` token. |
| S5 | **The simulator never answers `submit_sm` (timeout).** | **Pass only if** the simulator receives no second `submit_sm` for that message, under configuration we control. Otherwise J1 is confirmed and a patch or Option A decision is required. **Blocking criterion.** |
| S6 | Restart Jasmin with messages queued and in flight. | Every Go message ends either submitted exactly once or in an explicit uncertain/unknown state. None silently disappears. Duplicates at the simulator are counted. |
| S7 | Restart RabbitMQ with queued messages; restart Redis between submit and receipt. | Same as S6. Losses are **detected** by Go reconciliation within the configured window. |
| S8 | Go DLR endpoint down longer than Jasmin's retry window. | Receipts lost by Jasmin are detected (`UNKNOWN` after the window), not silently missing. Measure with tuned `max_retries`/`retry_delay`. |
| S9 | Multipart and UCS2 messages. | Correct segment count, UDH and encoding at the simulator; Go's own segment calculation matches. |
| S10 | Security posture. | Jasmin, RabbitMQ and Redis reachable only on a private network; jCli loopback-only with changed credentials; `log_privacy` on; no secret in logs. |

**Deliverable:** a short report with pass/fail per criterion and evidence (logs, PDU counts). Then a proposed ADR revision (§10) for owner decision. A throughput benchmark is a **separate** follow-up, and only if S1–S10 pass.

---

## 10. ADRs that may need revision (all remain Proposed; no status changed)

| ADR | Why |
| --- | --- |
| ADR-0012 Provider abstraction | Add the "gateway engine" seam: Jasmin, or a Go-native engine, as an implementation behind the provider/engine interface. Its option C ("third-party gateway software") would be re-evaluated with this evidence. The SMPP-library ADR it anticipates depends on the outcome. |
| ADR-0006 Broker | Clarify that an engine-private RabbitMQ does not change the internal event design (PostgreSQL outbox), if Jasmin is adopted. |
| ADR-0007 Redis | Add an "engine-private Redis" case with its constraints (no source of truth, persistence and monitoring because of DLR correlation). |
| ADR-0021 Message status model | Add the "accepted by gateway engine" milestone, and how engine-level uncertainty maps to `SUBMIT_UNCERTAIN` / `UNKNOWN`. |
| ADR-0009 Control/data plane | The data plane would partly run outside Go (in Jasmin); document the boundary. |
| New ADR | "SMS gateway engine selection" — records this comparison, the spike result, and the decision. |

---

## 11. Recommended next step

**[R] Approve (or reject) the time-boxed spike in §9.** Separately, the open PR #5 (Group 2) and Group 3 are unaffected by this decision and can proceed on their own schedule.

No implementation, dependency, infrastructure, or ADR change has been made. The spike starts only with your approval.

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
