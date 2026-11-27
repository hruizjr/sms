# Security Baseline

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — pending owner review (MFA, MVP roles and API key scopes in §3 updated for owner decisions of 2026-11-26) | Project owner | 2026-11-26 |

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
- **Passwords:** hashed with Argon2id (proposed) using parameters from current guidance; breached-password
  and minimum-length checks; rate-limited login with lockout/backoff.
- **MFA:** mandatory for **every human internal admin account** — Super Admin, Operations and Finance (owner
  decision D-021, widened on 2026-11-26, MVP-04); available (possibly mandatory by policy) for clients.
- **API keys:** random, high-entropy, with a visible prefix for identification; stored only as hashes;
  shown once; scoped (permissions, optional IP allowlist); revocable; last-used tracking. **MVP** (owner,
  2026-11-26, MVP-06): create, label, show once, store hashed and revoke, with the contract scopes
  `messages:send` and `messages:read`; IP allowlisting and advanced scope management are deferred.
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
- Rate limiting per IP, per user, per tenant, per API key; stricter limits on authentication endpoints.
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

- Append-only, tamper-evident storage (approach to be designed — e.g. restricted DB role with insert-only
  rights, periodic hash chaining or export to write-once storage).
- Recorded: authentication events, MFA changes, role/permission changes, API key lifecycle, credit and
  pricing changes, sender ID decisions, campaign approvals, provider/routing configuration changes,
  data exports, admin access to tenant data, tenant suspension.
- Each record: timestamp (UTC), actor, actor type (user/API key/system), tenant, action, target, result,
  source IP, request ID, and relevant before/after values (excluding secrets).
- Retention: to be defined (likely long; regulatory input required).

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
| S-3 | Session, MFA, and API key design details | Before identity implementation |
| S-4 | Data classification and retention schedule (with counsel) | Before storing production data |
| S-5 | Privacy impact assessment; DPO and NPC obligations (with counsel) | Before production |
| S-6 | Secret management and key rotation design | Hosting decision |
| S-7 | Abuse-prevention policy and tooling (KYC, sender ID vetting, content rules) | Before production sending |
| S-8 | Audit log tamper-evidence design | Before admin features |
| S-9 | Incident response and breach notification runbooks | Before production |
| S-10 | Independent penetration test | Before production launch |
| S-11 | Telco-specific security requirements (from telco checklist) | When telco specs received |
