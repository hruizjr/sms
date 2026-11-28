# Security Baseline

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — pending owner review (MFA, MVP roles and API key scopes in §3 updated for owner decisions of 2026-11-26; portal sessions and passwords in §3 for decision log §20, 2026-11-27; S-3 status for §21; S-3 and S-8 for G4-7 and G4-8, 2026-11-28; §3, §7, §9 and S-8 for Phase 4 Group 6, decision log §23, 2026-11-28; §3, §7, §9, S-8 and S-12 for the Group 6 security review remediation, G6-10 to G6-20, 2026-11-28; §3, §9, §9.1, S-12, S-13 and S-14 for the final remediation, G6-21 to G6-25, 2026-11-28) | Project owner | 2026-11-28 |

This baseline sets the minimum security principles for all work. It is **not** a complete security
program; §11 lists the security engineering still required. Nothing here is legal advice.

**Authority:** this document is authoritative for security principles and controls, as defined in [`AGENTS.md` §1.1](../../AGENTS.md#11-document-authority-and-conflicts);
`AGENTS.md` §12 summarizes it. It does not grant permissions, widen task scope, or override restrictions in `AGENTS.md`. If it genuinely conflicts with another document, stop the affected work, report the conflict, and request owner resolution ([`AGENTS.md` §1.1](../../AGENTS.md#11-document-authority-and-conflicts)).

## 1. What we are protecting

| Asset | Why it matters |
| ----- | -------------- |
| Customer credits and financial records | Direct financial loss; disputes; fraud |
| Telco connections and credentials | Abuse could get the business **suspended by a telco**, ending service for all customers |
| Sender IDs | Hijacked sender IDs enable impersonation and smishing |
| Recipient phone numbers and message content | Personal data under the Data Privacy Act (RA 10173); OTPs grant access to people's accounts |
| Customer API keys | Allow sending (and spending) as the customer |
| Admin accounts | Can change pricing, credits, routing, and see all tenants |
| Audit logs | Evidence for investigations, disputes, and regulators |

## 2. Main threats (initial)

1. **Platform abuse for smishing/scams** — criminals buying service or compromising a customer account to
   send fraudulent messages, especially impersonating banks, e-wallets, couriers, or government.
2. **Account takeover** — of customer users, API keys, or our admins.
3. **Cross-tenant data exposure** — a bug that shows one customer another's contacts or messages.
4. **Financial manipulation** — race conditions to overspend, unauthorized credit adjustments.
5. **Leakage of personal data** — via logs, exports, backups, analytics, or support access.
6. **Telco credential compromise** — leading to unauthorized traffic under our name.
7. **Injection and web vulnerabilities** — SQL injection, XSS, CSRF, SSRF (via webhooks), CSV injection.
8. **Denial of service** — API floods, giant imports, runaway campaigns exhausting credit or telco capacity.
9. **Supply chain** — malicious or vulnerable dependencies; leaked CI secrets.
10. **Insider misuse** — staff accessing message content or adjusting balances without cause.

## 3. Identity and access

- **Portal users:** server-side sessions in `HttpOnly`, `Secure`, `SameSite` cookies; session expiry and
  idle timeout; re-authentication for sensitive actions (credit adjustment, API key creation, pricing).
  **Implemented for client portal users** (Phase 4 Group 3; owner decisions G3-3 and G3-5, decision log §20):
  cookie `__Host-sms_session` (`HttpOnly`, `Secure`, `SameSite=Strict`), only a hash of the session secret stored,
  30-minute idle timeout, 12-hour absolute lifetime, server-side sign-out and revocation, CSRF token required on
  every state-changing request. Re-authentication for sensitive actions applies once such actions exist.
- **Passwords:** hashed with **Argon2id** (owner decision G3-2, 2026-11-27: OWASP parameters, 19 MiB, 2 iterations,
  1 lane); at least 12 characters (G3-3); five consecutive failed sign-ins lock the account for 15 minutes.
  **Layered per-source limits of sign-in are implemented** (Phase 4 Group 6, decision log §23, G6-11 and G6-12),
  shared by all API instances through PostgreSQL and answered with `429` and `Retry-After` before any credential
  check; they never lock an account and are independent of the lockout. Initial, configurable defaults: a budget of
  **failed** attempts of 100 per IPv4 address, 100 per IPv6 /64 and 500 per IPv6 /48 in 15 minutes (each attempt
  reserves a slot; a successful sign-in returns only its own slot), and an all-attempt token bucket of 10 requests
  per second with a burst of 100 per IPv4 address or IPv6 /64. Password hashing runs with bounded concurrency and a
  bounded wait queue; beyond it, `503` without checking the password. The client address is the direct peer unless
  the peer is a configured trusted proxy (`TRUSTED_PROXIES`, which **must be set explicitly in production**: proxy
  addresses or `none`; ranges broader than an IPv4 /8 or IPv6 /32 need an explicit opt-in and log a warning).
  Breached-password checks are **not implemented yet**.
  *Token bucket, precisely:*
  - one token per attempt, whatever its outcome; capacity 100 tokens; refill 10 tokens per second;
  - an idle source can use the full burst of 100 at once;
  - starting from a full bucket, about 200 attempts can be accepted within 10 seconds (100 plus 10 × 10, subject to
    token timing);
  - once the burst is spent, at most 10 per second (600 a minute) are sustained;
  - one shared row per source, so the same totals apply across all API instances.
- **Authentication availability (final review of Phase 4 Group 6).**
  - **What the controls do.** The per-source token bucket and the bounded password hasher protect memory and CPU.
  - **Valid credentials skip the failure budget.** A successful sign-in returns its reserved slots, so sign-ins with
    valid credentials (an attacker's own account, or several) are limited only by each source's token bucket.
  - **The queue can be monopolized.** Several valid accounts and source addresses can keep the one shared hashing
    queue full. Admission is first-come-first-served, not fair per source, so other users then receive `503`.
  - **IPv6.** There is no all-attempt limit per IPv6 /48: each /64 has its own bucket.
  - **Shared resources.** In a load test on a 4-vCPU development container, a sign-in flood also raised
    `/v1/messages` latency (p50 about 4 ms to about 80 ms), because the flood and the messages API share CPU and the
    database pool.
  - **Requirements.** The production requirements are in S-12; no new threshold is set without a measured rationale
    and owner approval.
- **MFA:** mandatory for **every human internal admin account** — Super Admin, Operations and Finance (owner
  decision D-021, widened on 2026-11-26, MVP-04); available (possibly mandatory by policy) for clients.
- **API keys:** random, high-entropy, with a visible prefix for identification; stored only as hashes;
  shown once; scoped (permissions, optional IP allowlist); revocable; last-used tracking. **MVP** (owner,
  2026-11-26, MVP-06): create, label, show once, store hashed and revoke, with the contract scopes
  `messages:send` and `messages:read`; IP allowlisting and advanced scope management are deferred. **Last-used
  tracking is implemented** (owner decision G4-6, decision log §23): written at most once per key per minute after a
  successful, authorized request, so it can lag the latest use by up to a minute.
- **Authorization:** RBAC with deny-by-default; every endpoint declares its required permission; checks
  happen server-side in a central mechanism, plus tenant scoping. Role design is an open task. **MVP roles**
  (owner, 2026-11-26): client **Owner** (manages users and API keys), **Sender** (sending functions) and
  **Viewer** (read-only), with no separate Billing role (MVP-03); internal **Super Admin**, **Operations** and
  **Finance**, with least privilege (MVP-04). Client financial history is initially visible to client Owners
  and authorized Finance staff only (MVP-22). Detailed permissions belong in the RBAC design (Phase 2B).
- **Admin access to tenant data:** read access is logged; any "act as customer" (impersonation) feature
  requires explicit justification, is time-limited, and is audited.
- **Separation of duties (proposed):** large credit adjustments and pricing changes may require a second
  approver when staff numbers allow.

## 4. Tenant isolation

- Tenant derived from the authenticated identity only.
- Database scoping by `tenant_id` on every tenant-owned record.
- **Row-Level Security is adopted** as defense-in-depth for **tenant-owned control-plane tables** (ADR-0010
  accepted 2026-11-26, after the [RLS evaluation](../architecture/rls-evaluation.md)). High-volume data-plane
  tables are deferred (RLS-5).
  - **What RLS is not.** It is **not** a substitute for application authorization: it only filters rows and
    knows nothing about roles or permissions.
  - **Tenant context.** It is set transaction-locally by one application helper that rejects missing or
    invalid tenant IDs, and never per pooled session.
  - **Roles.** The application role must not own tables, be a superuser, or hold `BYPASSRLS`. Database role
    separation is a prerequisite (RLS-4), implemented in Phase 4 Group 1: the API connects as `sms_app` and
    refuses an unsafe role; migrations run as `sms_migrator`, which owns the tables and, by owner decision
    (decision log §18, G1-5 and G1-9), has no `BYPASSRLS` either, so forced policies apply to its data
    queries too. **No blanket RLS bypass is granted to any role.** A future cross-tenant data migration, or a
    break-glass procedure, needs its own separately designed, controlled and approved procedure. (This
    supersedes the earlier rule that reserved `BYPASSRLS` for the migration role;
    [RLS evaluation §11.1](../architecture/rls-evaluation.md#111-superseded-bypassrls-for-the-migration-role-2026-11-26).)
  - **Policies** fail closed when the context is missing (the `NULLIF` form).
  - **Cross-tenant access** uses a dedicated, read-only reporting role, and is audited in the application.
  - **Threat model limit.** RLS protects against application query mistakes and missing tenant filters,
    **not** against a compromised application database credential or arbitrary SQL execution, which can set
    any tenant ([evaluation §3](../architecture/rls-evaluation.md#3-threat-model-owner-requirement-b)).
- Opaque, non-sequential identifiers.
- Exports, reports, caches, and background jobs are tenant-scoped too.
- Automated cross-tenant tests for every tenant-scoped feature.

## 5. Secrets management

- No secrets in Git, images, logs, frontend bundles, test fixtures, or documentation.
- Local development uses `.env` files (git-ignored) with development-only values.
- Deployed environments use a secret manager (choice depends on hosting — open decision).
- Telco credentials are stored separately from other configuration, accessible only to connector
  processes, and rotated per telco procedure.
- Leaked secrets are **rotated immediately**; removing them from Git history is not sufficient.
- CI secrets are scoped to the jobs that need them; forks/PRs from untrusted sources do not receive secrets.

## 6. Data protection and privacy

- **Personal data in scope:** recipient phone numbers, contact names and attributes, message content,
  user account data, IP addresses.
- **Minimization:** collect only what is needed; avoid storing message content longer than required
  (retention period to be decided — D-073, D-124).
- **Protection:** TLS for all traffic; encryption at rest for databases and backups; field-level
  encryption for especially sensitive data considered in design.
- **Logs:** phone numbers masked; message bodies and OTP values never logged.
- **Raw delivery receipts** are stored for audit, and some formats can include a fragment of message text;
  they are treated as personal data (restricted access, retention rules).
- **Access:** staff access to message content restricted by role and audited.
- **Data subject rights and breach response:** processes to be defined with privacy counsel under the
  Data Privacy Act; conflicts between erasure requests and mandatory retention to be resolved explicitly.
- **Roles:** for customer-provided contacts we are likely a *personal information processor* acting for
  the customer (a *personal information controller*) — to be confirmed by counsel and reflected in
  customer contracts (data processing agreements).

## 7. Application and API security

- Input validation at every boundary (HTTP, imports, telco callbacks, webhooks); allow-lists over
  deny-lists.
- Parameterized SQL only; output encoding in the UI; no HTML rendering of user content.
- CSRF protection for cookie-authenticated requests; strict CORS.
- Security headers (CSP, HSTS, `X-Content-Type-Options`, frame protection) on portal responses.
- Request size limits, timeouts, and pagination limits.
- Rate limiting per IP, per user, per tenant, per API key; stricter limits on authentication endpoints. *Status:*
  portal sign-in is limited per source in layers (failed-attempt budgets per IPv4 address, IPv6 /64 and /48; an
  all-attempt token bucket; bounded password hashing — Phase 4 Group 6, G6-11, G6-12); per-user, per-tenant and
  per-key limits of the public API are not implemented yet. Edge rate limiting or a WAF in front of the API is a
  production gate (S-12): the application limits do not stop an attacker controlling many networks, and a
  distributed flood can still make sign-in busy (`503`).
- File imports: size and row limits, type checking, processing in background with progress, no execution
  of content; exported CSV/Excel cells sanitized against formula injection.
- Outbound webhooks: HMAC signatures with timestamps (replay protection), URL validation against private,
  loopback, link-local and cloud metadata addresses (SSRF), timeouts, limited redirects.

## 8. Abuse prevention (messaging-specific)

- Customer verification (KYC) before production sending — requirements per D-085.
- Sender IDs approved before use; checks against impersonation of well-known brands and government;
  per-telco registration tracked.
- Campaign approval rules for new tenants and large campaigns (D-025, D-087).
- Content policies (e.g. URL handling, prohibited content) — pending telco and regulatory input (D-110, D-123).
- Velocity limits and anomaly alerts (sudden volume spikes, high failure rates, many opt-outs).
- Platform-wide suppression list and opt-out enforcement before dispatch.
- Ability to suspend a tenant, sender ID, or campaign immediately (kill switch), audited.

## 9. Audit logging

- Append-only storage under the application's permissions, plus tamper evidence that holds against privileged
  database access: an **external immutable export** (write-once storage outside the database's administrative
  domain) or **externally anchored integrity evidence** (for example periodic digests published or stored where the
  database administrators cannot change them). A hash chain kept inside the same PostgreSQL database does **not**
  protect against a database superuser, who can rewrite the chain together with the rows.
- Recorded: authentication events, MFA changes, role/permission changes, API key lifecycle, credit and
  pricing changes, sender ID decisions, campaign approvals, provider/routing configuration changes,
  data exports, admin access to tenant data, tenant suspension.
- Each record: timestamp (UTC), actor, actor type (user/API key/system), tenant, action, target, result,
  source IP, request ID, and relevant before/after values (excluding secrets).
- Retention: to be defined (likely long; regulatory input required).
- *Status (Phase 4 Group 6, decision log §23):* a durable audit log exists for portal sign-in, sign-out, failed
  sign-in, account lockout, sign-in rate limiting, API key creation (including failed password confirmation) and
  revocation, and refused API key and audit log requests. Tenant events are stored under forced RLS and readable by
  the organization's owners; events of no identifiable organization are stored in a platform table the application
  cannot read. Append-only is enforced by privileges (insert only, with the event ID and time assigned by
  PostgreSQL so the application can neither back-date nor collide with an event) and by triggers that reject
  updates, deletes and truncation by any role including the table owner. These controls stop ordinary application
  modification; they are **not** tamper-proof and do **not** protect against a database superuser (or the table
  owner after disabling the triggers). An account lock is recorded with the outcome `enforced`. Refused requests are
  recorded once per user, action and five minutes, and attempts on a locked account once per lock, through a unique
  deduplication key (no event is modified). Portal user creation, unlocking and session revocation by `smsctl` are
  recorded as system actions. Sender ID approvals (`smsctl sender-id approve`, `bootstrap`) are recorded as
  `sender_id.approve` in the organization's log, in the same transaction as the approval (a repeated approval changes
  and records nothing). Each record holds the fields above except before/after values (no audited action changes a
  value yet). The external immutable export (S-8) and the retention period (S-4) remain open; platform-wide events
  stay readable only by database administrators until an approved internal function exists.
- *Limitations of system actions:* `smsctl` actions are recorded with the actor `system` and `via: smsctl`. The tool
  has no operator identity, so the record does not say which person acted. **Not audited:** `smsctl` tenant creation,
  activation, suspension and reactivation (platform-owned tenant registry). Both need the platform administration
  audit design of S-14; no tenant-log workaround is used.

### 9.1 Operational monitoring of the security controls

The repository has no alerting infrastructure: the API exposes Prometheus metrics on its separate, non-public
metrics listener, and production monitoring must collect them and configure the alerts below. Labels never contain
addresses, users or tenants. The thresholds are **initial recommendations** to be tuned against production traffic;
they are alert conditions, not rate limits.

**Implemented metrics; the alert must be configured externally:**

| Concern | Metric | Recommended alert |
| ------- | ------ | ----------------- |
| Cleanup failure / last-success age | `sms_signin_limit_cleanup_last_success_timestamp_seconds` (per process; 0 until a cycle succeeds; a failed cycle never moves it) and `sms_signin_limit_cleanup_runs_total{result="failure"}` | `time() - sms_signin_limit_cleanup_last_success_timestamp_seconds > 300` for 2 min (five cycles missed) |
| Cleanup failure (rate) | `sms_signin_limit_cleanup_runs_total{result="failure"}` | `increase(...[15m]) >= 3` on any process |
| Cleanup backlog | `sms_signin_limit_cleanup_incomplete_runs_total` (a cycle stopped at its work bound of 50,000 rows per table) | `increase(...[15m]) >= 3`: new sources arrive faster than cleanup removes them. Check edge limits (S-12) and table growth (below) |
| Cleanup duration | `sms_signin_limit_cleanup_duration_seconds` (histogram) | p99 over 30 min `> 30` s (the cycle's timeout is 60 s) |
| Rows deleted by cleanup | `sms_signin_limit_cleanup_deleted_rows_total{table}` | No fixed alert; graph it. A sudden rise means many new sources (a distributed flood) |
| Rate-limit `429` | `sms_http_requests_total{route="/portal/v1/session",code="429"}` | More than 20 % of sign-in requests for 10 min: an attack, or a limit too low for a shared address |
| Password-hashing `503` | `sms_http_requests_total{route=~"/portal/v1/session\|/portal/v1/api-keys",code="503"}` | Any for 5 min (warning); more than 10 % of sign-in requests for 10 min (critical): the hashing queue is saturated |
| Effect on the messages API | `sms_http_request_duration_seconds{route=~"/v1/messages.*"}` | p99 above the agreed service level (to be set) for 10 min, especially during sign-in floods |
| Audit-write failure, same transaction | `sms_http_requests_total{code="500"}` on the audited portal routes. When an event cannot be written, the change is rolled back and the request fails | Any sustained `500`. Confirm through the error log |

**Monitoring not implemented yet** (no metric exists):

| Concern | Today | Required |
| ------- | ----- | -------- |
| Audit-write failure, refusal events | Logged as errors (`audit event of a refused request not recorded`, `audit event of a rate-limited sign-in not recorded`); no metric | A counter of failed audit writes by action, alerted on any increase. A log-based alert on those messages is the interim option |
| Hashing queue saturation | Visible only as `503` | Gauges of busy hashing slots and waiting callers, and a counter of refusals |
| Rate-limit table growth | Rows can be counted in PostgreSQL; no metric | Live rows and size of `portal_sign_in_failure_budgets` and `portal_sign_in_request_buckets` from the database monitoring, for example the PostgreSQL statistics views; alert above about 500,000 rows each (ten cycles' work of one process; initial value) |
| Database pool pressure during sign-in floods | No pool metric | pgx pool statistics: acquired connections, acquire wait time, empty acquires; alert when acquire waits rise with sign-in traffic |

## 10. Infrastructure and supply chain (when infrastructure exists)

- Least-privilege network access; databases not reachable from the internet; telco connectivity per telco
  requirements (VPN/IP allowlisting).
- Minimal container images, non-root users, pinned base images, vulnerability scanning.
- Dependency pinning with lockfiles; automated vulnerability checks; new dependencies require approval.
- Protected `main` branch; required reviews/checks; signed commits optional (owner decision).
- Backups encrypted and restore-tested.

## 11. Security work still required (not done in Phase 1)

| # | Work item | When |
| - | --------- | ---- |
| S-1 | Formal threat model (data flows, trust boundaries, STRIDE or similar) | Architecture phase |
| S-2 | RBAC role and permission design for client and admin portals | Before identity implementation |
| S-3 | Session, MFA, and API key design details | Client portal sessions and API keys designed and implemented, including owner-only API key management in the portal with password re-authentication before a key is created (decision log §19, §20, §21, G4-7); MFA, admin sessions and re-authentication for other sensitive actions remain open |
| S-4 | Data classification and retention schedule (with counsel) | Before storing production data |
| S-5 | Privacy impact assessment; DPO and NPC obligations (with counsel) | Before production |
| S-6 | Secret management and key rotation design | Hosting decision |
| S-7 | Abuse-prevention policy and tooling (KYC, sender ID vetting, content rules) | Before production sending |
| S-8 | Audit log tamper-evidence design | Before admin features, and before production in any case: API key creation and revocation must be durable audit events; application logs are not a substitute (owner decision G4-8, 2026-11-28). **Durable, append-only audit log implemented** (Phase 4 Group 6, decision log §23). **Production requirement (G6-19):** an external immutable export of the audit log, or externally anchored integrity evidence, so that alteration by privileged database access is detectable; an internal PostgreSQL hash chain does not satisfy it. Not implemented |
| S-9 | Incident response and breach notification runbooks | Before production |
| S-10 | Independent penetration test | Before production launch |
| S-11 | Telco-specific security requirements (from telco checklist) | When telco specs received |
| S-12 | Sign-in abuse controls at the edge, and production capacity measurements (G6-11, G6-12, G6-15, G6-24) | Before production (and edge limits before any public deployment): edge rate limiting or WAF in front of the API; the real `TRUSTED_PROXIES` of the hosting (D-145); Argon2 throughput, and the hashing concurrency, queue and wait, measured on the production hardware; database-pool contention tested during sign-in floods (effect on `/v1/messages`); the sign-in limit defaults reviewed against measured traffic (carrier-grade NAT); an evaluation of an aggregate all-attempt limit per IPv6 /48; an evaluation of fair or per-source hashing admission, so one source cannot monopolize the queue (valid-account sign-ins refund their failure slots, §3). No new threshold without a measured rationale and owner approval |
| S-13 | Production monitoring and alerting of the security controls (§9.1; G6-23) | Before production: collect the implemented metrics and configure the alerts of §9.1; implement the missing monitoring there (audit-write failure counter, hashing queue gauges, rate-limit table growth, database pool statistics) |
| S-14 | Platform administration audit (G6-25) | Before admin features, and before production in any case: an audit design for platform-level administration, recording tenant creation, activation, suspension and reactivation (today unaudited `smsctl` commands). Its records cannot live in a tenant's log, and need a writer role and a reader policy (D-021); actions must be attributed to authenticated operators (MFA-protected internal admin accounts, D-021) instead of the `system` actor. Owner: the admin portal / platform administration group, when the owner authorizes it |
