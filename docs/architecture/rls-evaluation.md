# PostgreSQL Row-Level Security evaluation (ADR-0010, D-141)

| Field | Value |
| ----- | ----- |
| Status | **Concluded: owner decisions taken 2026-11-26** (§11). **ADR-0010 is Accepted**: option B for tenant-owned **control-plane** tables, under the conditions in §10. D-141 is resolved. The RLS strategy for high-volume **data-plane** tables is **deferred** (RLS-5). Nothing is implemented yet: database role separation (RLS-4) comes first, in a separately approved group. |
| Date | 2026-11-26 (Phase 3 Group 7) |
| Scope | The six evaluation requirements in [ADR-0010](../engineering/decision-log.md#adr-0010-multi-tenancy--shared-schema-with-tenant_id-rls-as-evaluated-defense-in-depth), plus the owner's requirements A–D for this group ([decision log §12](../engineering/decision-log.md#12-owner-decisions--rls-evaluation-spike-phase-3-group-7-2026-11-26)) |
| Evidence base | PostgreSQL **18.7** (`postgres:18.7-trixie@sha256:c699303802144847d238196173655dbbecf7a96aeca1565f2b02eb671a1f0b6b`, the image used by Compose and backend CI); Go 1.27.4; `github.com/jackc/pgx/v5` v5.13.0 (pgxpool). Tests: [`backend/internal/database/rlseval/`](../../backend/internal/database/rlseval/), 22 tests, run with `-race` |
| Not product code | The tests use **synthetic, test-only tables** (`rlseval.*`) in disposable databases. There is no migration, business table, runtime role change or product helper. |

Labels used:

- **[T]** observed in a committed test (the test name is given);
- **[D]** from the PostgreSQL 18 documentation, not tested here;
- **[A]** assumption or inference, not tested;
- **[R]** recommendation for the owner;
- **[NT]** not tested.

---

## 1. Summary

1. **RLS works as a fail-closed backstop with pgxpool, under conditions.** The conditions are:
   - the tenant is set **per transaction** with `set_config('app.tenant_id', $1, true)`;
   - the application role is subject to RLS;
   - every tenant table has RLS **enabled and forced**.

   Across 1,280 concurrent transactions on a 4-connection pool, no transaction saw another tenant's rows **[T]**.
2. **The current setup would get no protection from RLS.** Today the API and `migrate` both connect as `sms`, which is the PostgreSQL image's bootstrap **superuser**. Superusers always bypass RLS **[T]**. Adopting RLS therefore requires separate database roles first (§10). This group changes no runtime role.
3. **It protects against mistakes, not against a compromised application.** RLS here guards against application query mistakes and missing tenant filters. Code that can run arbitrary SQL as the application role can choose any tenant **[T]** (§3).
4. **The cost was not measurable in these indicative tests.** The policy is used as an index condition, or evaluated once per query as a one-time filter, and the differences were within run-to-run noise. Setting the context costs one extra statement, and therefore one extra round trip, per transaction (§12).
5. **[R] Recommendation: adopt ADR-0010 option B, with the conditions in §10.** Option B is RLS as defense-in-depth on top of mandatory application tenant filters. It would apply to tenant-owned control-plane tables, starting with the first tenant table. High-volume data-plane tables would be decided when they are designed.
   **Owner decision (2026-11-26): adopted as recommended for control-plane tables; data-plane tables deferred** (§11).

## 2. Method and test isolation

**How roles are used.** Each test creates one fixture:

- a disposable database (`dbtest.NewDatabase`);
- four **NOLOGIN** roles named `rlseval_<run>_<fixture>_<purpose>`;
- a schema `rlseval`, owned by the owner role, with seeded synthetic rows for two synthetic tenant UUIDs.

| Role | Purpose | Attributes |
| ---- | ------- | ---------- |
| `…_owner` | Owns the schema and tables. It stands in for a future migration role | NOSUPERUSER, NOBYPASSRLS |
| `…_app` | Stands in for the application runtime role | NOSUPERUSER, NOBYPASSRLS, not owner; SELECT, INSERT, UPDATE and DELETE only |
| `…_reporting` | Cross-tenant model 1: explicit read policies | NOSUPERUSER, NOBYPASSRLS |
| `…_bypass` | Cross-tenant model 2 | **BYPASSRLS** |

**How the tests connect.** The test process connects as the disposable server's superuser and takes on a role with `SET ROLE`; the owner required the roles to be NOLOGIN.
- PostgreSQL applies row security to the **current role**, so a session after `SET ROLE …_app` is filtered exactly as a login as that role would be **[T]**. `TestApplicationRoleCannotBypass` checks the role's attributes in the catalog.
- **Limitation:** a `SET ROLE` session can run `RESET ROLE`, and its session user stays the superuser. A real deployment would log in as the application role directly. That is untested here **[A]**.

**Isolation (owner requirement A).** PostgreSQL roles belong to the whole server, not to one database, so the tests run only when **all** of these hold. Each guard was checked by running against an unsafe target and seeing the run refused.

| Guard | Checked by |
| ----- | ---------- |
| Both build tags `integration` and `rlseval` are given. The ordinary `go test -tags integration ./...` run does not compile the package | Ordinary integration run contains no `rlseval` package |
| `RLS_EVAL_DISPOSABLE_SERVER=1` is set | Run without it: refused |
| The dbtest guards hold: `APP_ENV=test` and a loopback IP for `DB_HOST` | `APP_ENV=development`: refused |
| The server's maintenance database has **never been migrated**. If `goose_db_version` exists, the run is refused; this excludes the local Compose stack's persistent database | Run against a migrated maintenance database: refused |
| The connecting role is a superuser of that throwaway server | — |
| Role names must match the run's generated prefix before any `CREATE ROLE` or `DROP ROLE`; existing roles are never altered | — |
| Cleanup drops each fixture's roles after its database is dropped, also when a test fails | Deliberately broken policy: 9 tests failed and no role was left behind |
| `TestMain` fails the run if any `rlseval_` role remains on the server, from this run or any other | Prints `no rlseval_* roles remain on the server` |

**Where it runs:**

- **Locally:** [`scripts/rls-evaluation.sh`](../../scripts/rls-evaluation.sh) starts its own container from the same pinned image. It has a random name, its data directory on tmpfs, a random port bound to 127.0.0.1 and a random throwaway password. It is not part of any Compose project. The script runs the tests, checks that no `rlseval_` role or `sms_test_` database remains, and removes the container.
- **CI:** the "Go test and build" job runs the tests against its own ephemeral PostgreSQL service container, before the migration smoke test migrates that container's maintenance database. It then checks that no `rlseval_` role remains.

## 3. Threat model (owner requirement B)

- **The setting is not a security boundary.** The tenant context is `app.tenant_id`, an ordinary custom setting. Any session can set it without any privilege. RLS trusts **the application code** to set it to the authenticated principal's tenant (AGENTS.md §6.3) **[D]**.
- **Code that can run SQL as the application role can choose any tenant.** That includes a compromised application process, a stolen application database credential, or a SQL injection that can run statements. Such code can call `set_config('app.tenant_id', '<other tenant>', true)` and read or write that tenant's rows. `TestContextIsChosenByTrustedCode` shows this **[T]**.
- **What this RLS design protects against:**
  - a repository query that forgets the `tenant_id` filter, or filters on the wrong value;
  - a query added later that bypasses the repository layer;
  - joins, views and reports written by mistake against tenant tables;
  - a missing or malformed context, for example a background job that never set one. These fail closed (§5).
- **What it does not protect against:**
  - a compromised application role or arbitrary SQL execution;
  - a superuser, a table owner on an unforced table, or a `BYPASSRLS` role (§4);
  - flaws in authorisation logic, because RLS knows nothing about users, roles or permissions (ADR-0010).

  Those threats need separate controls: credential protection and rotation, least-privilege roles, parameterised SQL only (AGENTS.md §7.8), and auditing (security baseline).
- **[A] Signed or verified tenant context was not evaluated.** An example would be a signed token checked by a `SECURITY DEFINER` function. It would add complexity, and it still trusts whatever holds the signing key.

## 4. Results by ADR-0010 requirement

| # | Requirement | Result | Tests |
| - | ----------- | ------ | ----- |
| 1 | Transaction-scoped context | **[T] Pass.** `set_config(..., true)` applies only inside the transaction; after COMMIT or ROLLBACK, the next transaction sees no rows. Rolling back a savepoint restores the previous tenant. **Negative case:** `set_config(..., false)` and `SET app.tenant_id` outlive the transaction, so the next request on the same pooled connection runs as the previous tenant. Session-level settings must never be used | `TestContextIsTransactionLocal`, `TestContextRolledBackWithSavepoint`, `TestSessionLevelContextLeaks` |
| 2 | Connection pooling (pgxpool) | **[T] Pass.** With one pooled connection, transactions for tenant A, then none, then B, then none, then A run on the same backend process (same `pg_backend_pid`), and each sees only what its context allows. 32 goroutines × 40 transactions over 4 connections, mixing tenants A, B, C and none, saw only their own rows under `-race`. A failed or cancelled transaction leaves no context behind. pgxpool does not reset session state between acquisitions, so transaction scope is what makes this safe. **External poolers: [NT]** (§9) | `TestPooledConnectionReuse`, `TestPoolConcurrentTenants`, `TestContextClearedAfterFailedTransaction` |
| 3 | Fail closed | **[T] Pass with the `NULLIF` policy.** Without context: SELECT returns no rows; INSERT fails with 42501; UPDATE and DELETE affect **zero rows without an error**. `WITH CHECK` blocks inserting a row for another tenant and moving a row to another tenant, and tenant B's rows are unaffected by tenant A's updates and deletes. A malformed value fails with 22P02. Details in §5 | `TestMissingContextFailsClosed`, `TestWithCheckBlocksCrossTenantWrites`, `TestMalformedContextIsAnError`, `TestPolicyExpressionVariants` |
| 4 | Cross-tenant access | **[T]** Both models behave as expected. Comparison and recommendation in §6 | `TestCrossTenantExplicitPolicies`, `TestCrossTenantBypassRole` |
| 5 | Performance | **[T] Indicative only, manual.** No measurable policy overhead; one extra statement per transaction (§12) | `BenchmarkRLSOverhead` |
| 6 | Tests (missing context, pooled reuse, no bypass) | **[T] Pass.** See below | `TestApplicationRoleCannotBypass`, `TestWhoBypassesRLS` |

Results for requirement 6:

- **The application role:**
  - is not superuser, has no BYPASSRLS and is not the owner;
  - with `row_security = off`, gets an **error** (42501) instead of unfiltered rows;
  - cannot disable or un-force RLS, drop or create a policy, take over the table, or give itself BYPASSRLS. All of these fail with 42501.
- **Who bypasses RLS:**
  - a **superuser** bypasses it, even on forced tables;
  - a **table owner** bypasses it unless the table has `FORCE ROW LEVEL SECURITY`;
  - on a forced table, an owner with no policy of its own sees **nothing**;
  - a **`BYPASSRLS` role** bypasses it.

## 5. Fail-closed behaviour and the policy expression

**[T] A PostgreSQL behaviour the policy must handle** (`TestContextIsTransactionLocal`):

- `current_setting('app.tenant_id', true)` returns **NULL** on a connection that has never used the setting;
- it returns the **empty string** once a transaction-local value has ended;
- pooled connections are long-lived, so both states occur in practice.

| Policy expression | Fresh connection, no context | Used connection, no context | With context |
| ----------------- | ---------------------------- | --------------------------- | ------------ |
| `tenant_id = current_setting('app.tenant_id', true)::uuid` | No rows | **Error 22P02** (`''::uuid`) | Tenant's rows |
| `tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid` | No rows | No rows | Tenant's rows |
| `tenant_id = rlseval.require_tenant_id()` (PL/pgSQL, raises if unset) | Error P0001 | Error P0001 | Tenant's rows |

- **The raw cast behaves inconsistently.** It sometimes returns nothing and sometimes errors, depending on the connection's history. It should not be used.
- **[R] Use the `NULLIF` form** (adopted by the owner, RLS-2). It is consistent, it is the form that was benchmarked, and the planner uses it as an index condition (§12). A missing context is a programming error, but this form doesn't surface it, so the application helper must always set the context.
- **The `require_tenant_id()` form** fails loudly, which surfaces bugs at once. Its performance was **not** benchmarked **[NT]**. It is an alternative for the owner to consider.
- **[T] UPDATE and DELETE fail silently.** On rows the role cannot see, or with no context, they affect zero rows and raise no error. **[R]** Repositories should treat "0 rows affected" on a targeted UPDATE or DELETE as "not found", as they would without RLS.

## 6. Cross-tenant access (owner requirement C)

Admin reporting, reconciliation and background jobs may legitimately need to work across tenants. Neither model below is implemented in product code.

| | Model 1: explicit cross-tenant policies on a dedicated role | Model 2: a dedicated `BYPASSRLS` role |
| - | --- | --- |
| **Scope** | **Opt-in per table and per command.** A table with no policy for the role returns nothing **[T]** | **Every table** the role has a privilege on, including tables added later, with no policy evaluated **[T]** |
| **Writes** | Denied unless a write policy exists, **even when the privilege is granted** (INSERT failed with 42501) **[T]** | Any tenant's rows can be written wherever a write privilege is granted, without any context **[T]** |
| **Narrowing** | The policy can limit columns, via grants, or rows, for example to a tenant list or a status | Only table privileges limit it |
| **Failure mode** | Forgetting a policy makes data invisible: fails closed | Forgetting to revoke a privilege exposes everything: fails open |
| **Review** | Each new policy is visible in the migration diff | One attribute grants everything. Easy to over-use |
| **Cost** | One policy per table and command; more migration text | Minimal setup |

**[R] Recommended least-privilege model** (adopted by the owner, RLS-3):

1. **Cross-tenant work uses its own role, credential and connection pool.** It never uses the application pool. Reporting gets model 1: **read-only explicit policies**, added table by table in migrations, with column grants where some data must not be exposed. Cross-tenant writes, such as reconciliation corrections, get narrowly scoped write policies only when an approved feature needs them.
2. **`BYPASSRLS` is reserved for the migration role.** It needs it anyway to run data migrations on forced tables (§10), and a break-glass role would also have it. It is never used by any application or worker pool.
3. **No product code exists for either model yet.** The roles, policies and pools are designed with the first feature that needs cross-tenant access.

**Auditing implications** (the security baseline requires admin access to tenant data to be audited):

- **The database cannot tell why a query ran.** It only sees which role ran it. Every cross-tenant operation must therefore be recorded in the application's append-only audit log, with the actor, purpose, tenants or scope, and time.
- **A separate role and credential make cross-tenant sessions attributable at the database level.** The session user appears in `pg_stat_activity` and in server logs.
- **[A] Statement logging is possible for that role only**, for example `ALTER ROLE … SET log_statement = 'all'`, without logging the application's traffic. This is not tested here, and logged statements may contain personal data, so it needs a retention decision.
- **Model 2 makes audit gaps more dangerous.** A `BYPASSRLS` credential reads every tenant's data in every table, so its use must be rarer, logged, and alerted on.

## 7. Pitfalls

| Pitfall | Observed | Mitigation **[R]** | Test |
| ------- | -------- | ------------------ | ---- |
| **Unique constraints check all rows** | **[T]** With a global `UNIQUE (code)`, tenant B's insert fails with 23505 when **another** tenant already uses the code: an existence side channel. PostgreSQL 18.7 **omits the key value** from the error detail when RLS applies to the caller, so the other tenant's value is not repeated | Tenant-scoped uniqueness: `UNIQUE (tenant_id, …)` (AGENTS.md §10.7). Global uniqueness only where it is a deliberate business rule | `TestUniqueConstraintRevealsOtherTenants` |
| **Foreign-key checks ignore RLS** | **[T]** With an FK on `parent_id` alone, tenant B attached a child to tenant A's invisible parent (a cross-tenant reference). A missing parent fails differently (23503), so B learns whether an ID exists | Composite foreign keys `(tenant_id, parent_id) → (tenant_id, id)`. Both cases then fail identically | `TestForeignKeyChecksIgnoreRLS` |
| **Views run as their owner** | **[T]** A view owned by the owner of an unforced table showed **every tenant** to the application role. With `security_invoker = true` it was filtered. On a forced table the owner-run view returned nothing (default deny) | `WITH (security_invoker = true)` on every view over tenant tables; FORCE on tenant tables | `TestViewsAndSecurityDefinerFunctionsRunAsOwner` |
| **`SECURITY DEFINER` functions run as their owner** | **[T]** Same leak as views; `SECURITY INVOKER` was filtered | No `SECURITY DEFINER` over tenant tables without explicit review; FORCE | same |
| **`TRUNCATE` ignores RLS** | **[T]** With the TRUNCATE privilege, the application role in tenant A's context emptied the table for **all** tenants. Without it, the statement was refused | Never grant TRUNCATE (or REFERENCES, TRIGGER, ownership) to the application role | `TestTruncateIgnoresRLS` |
| **Tables without protection** | **[T]** One catalog query lists every table with a `tenant_id` column whose RLS is not enabled, not forced, or has no policy | **Proposed** CI guard once tenant tables exist (§10) | `TestCatalogGuardFindsUnprotectedTables` |
| **Leaky functions and operators in queries** | **[D]** Policy conditions are applied before user-supplied conditions unless the functions involved are `LEAKPROOF`, which can change plans | Avoid custom functions over tenant data in WHERE clauses; check plans of hot queries | [NT] |
| **`COPY`, replication, backups** | **[D]** `COPY TO` applies RLS. Logical replication and `pg_dump` (run as owner or superuser) see all rows | Treat backups and replicas as full-trust data | [NT] |

## 8. pgx statement cache and plan caching (future sqlc queries)

sqlc-generated code sends each query as a fixed SQL string. With pgx's default `QueryExecModeCacheStatement`:

- each query becomes a **prepared statement** reused across transactions, and therefore across tenants;
- PostgreSQL may switch a prepared statement to a cached **generic plan**.

`TestStatementAndPlanCachesFollowTheContext` covers this **[T]**:

- **Modes:** all five pgx execution modes (`CacheStatement`, `CacheDescribe`, `DescribeExec`, `Exec`, `SimpleProtocol`).
- **Plan settings:** each mode is run with `plan_cache_mode` forced to generic and to custom.
- **Queries:** two queries, one relying on the policy alone and one with an explicit tenant filter.
- **Sequence:** 12 transactions switching between tenants A, B and C and no context.

Results:

- Every execution returned exactly its own tenant's rows.
- In `CacheStatement` mode, `pg_prepared_statements` confirmed the reuse: 24 generic-plan executions and no custom plans when forced generic, and the reverse when forced custom.
- **Why it is safe:** the policy reads the tenant through `current_setting()`, a STABLE function evaluated at execution time. Neither cache carries one tenant's filter into another tenant's execution.

**sqlc compatibility [A].** sqlc's generated `New(db DBTX)` accepts a `pgx.Tx`. A tenant transaction helper can therefore set the context and hand the transaction to unchanged generated queries.
- `withtenant_test.go` holds a **test-only sketch** of such a helper. It validates the UUID in Go, then calls `set_config('app.tenant_id', $1::uuid::text, true)` with a bound parameter.
- It is not product code. No sqlc queries were generated.

## 9. External poolers: not tested [NT]

- **No external pooler was tested.** None, PgBouncer included, is decided, approved or present (owner restriction), and none was run.
- **[A] What should hold:** `set_config(..., true)` is transaction-scoped, so it should be compatible in principle with transaction-mode pooling, where every statement of a transaction runs on one server connection.
- **What must be re-checked before adopting one:**
  - requirements 1–3 and the concurrency test, run through the pooler;
  - session-level state, including the forbidden `set_config(..., false)` and `SET ROLE`, under the pooler's reset behaviour;
  - prepared statements, which pgx caches by default, under the pooler's prepared-statement support. Otherwise pgx would need `QueryExecModeExec` or `SimpleProtocol`, both of which passed in §8;
  - statement-mode pooling, which breaks multi-statement transactions and is **incompatible** with this design.

## 10. Conditions of adoption (adopted by the owner, RLS-1)

The owner adopted RLS on 2026-11-26 **conditional on all of these requirements**. They apply to tenant-owned control-plane tables (RLS-5 defers data-plane tables). **Status (Phase 4 Group 1, [decision log §18](../engineering/decision-log.md#18-owner-decisions--database-role-separation-and-tenant-registry-phase-4-group-1-2026-11-26)):** condition 1 (roles), the helper of condition 3 and the catalog guard are implemented; no tenant-owned table exists yet, so conditions 2 and 4 to 8 apply from the first one.

1. **Separate database roles first.** Implemented in Phase 4 Group 1 (`sms_migrator`, `sms_app`).
   - A migration role owns the tables (DDL). The evaluation proposed giving it `BYPASSRLS` for data migrations on forced tables; **the owner decided otherwise on 2026-11-26** (Phase 4 Group 1, G1-5): `sms_migrator` has **no** `BYPASSRLS`, does not own the schema, and cannot create roles. A data migration on tenant-owned rows therefore needs its own approved approach (an open follow-up in decision log §18).
   - An application runtime role has `LOGIN, NOSUPERUSER, NOBYPASSRLS`, is not the owner, and has at most `SELECT, INSERT, UPDATE, DELETE` on the tables it uses. It gets no TRUNCATE, REFERENCES, TRIGGER or ownership. (`sms_app` has exactly what each migration grants; for `tenants`: `SELECT`, `INSERT (display_name)`, `UPDATE (status, updated_at)`.)
   - The bootstrap superuser is not used by the API (the API refuses an unsafe role).
2. **Every tenant-owned table** has `ENABLE` and `FORCE ROW LEVEL SECURITY` and a `NULLIF`-form policy with matching `USING` and `WITH CHECK`, all written in its migration.
3. **Context is transaction-scoped only,** set by one application helper as the first statement of the transaction. The helper rejects missing or invalid tenant IDs before any tenant query runs (RLS-2). Session-level settings are never used.
4. **Application tenant filters stay mandatory** (AGENTS.md §10.7). RLS is the backstop, not the mechanism.
5. **Keys include `tenant_id`.** Unique constraints and foreign keys on tenant data include it; views use `security_invoker = true`; there is no `SECURITY DEFINER` over tenant tables without review.
6. **Cross-tenant access follows §6:** explicit read-only policies on a separate role and pool, audited in the application; `BYPASSRLS` only for the migration and break-glass roles.
7. **A catalog CI check** fails if any table with `tenant_id` is unprotected (§7).
8. **Re-evaluate** before introducing an external pooler (§9), and when designing high-volume or partitioned data-plane tables (policy cost at scale, partition behaviour) **[NT]**.

**Test-standard changes.** These were proposed here while the decision was pending. Following the adoption decision, they are now project standards in [testing standards §3.2](../standards/testing-standards.md#32-integration-tests):

- for every tenant-scoped table, a test that the runtime role sees no rows without context;
- a test that `WITH CHECK` rejects cross-tenant writes;
- the catalog guard as a CI check;
- a pooled-connection reuse test for the transaction helper.

The standards also require a test that the helper rejects missing or invalid tenant IDs (RLS-2), and that the reporting role cannot write (RLS-3).

## 11. Owner decisions (2026-11-26)

The owner reviewed this evaluation and decided as follows. The authoritative record is
[ADR-0010](../engineering/decision-log.md#adr-0010-multi-tenancy--shared-schema-with-tenant_id-rls-as-evaluated-defense-in-depth)
and [decision log §12.1](../engineering/decision-log.md#121-owner-decisions-on-rls-adoption-2026-11-26).

**Accepted:**

| ID | Decision | Compared with the recommendation |
| -- | -------- | -------------------------------- |
| RLS-1 | **Adopt option B:** RLS as defense-in-depth for tenant-owned control-plane tables, on top of mandatory application tenant filtering, conditional on all of §10 | As recommended |
| RLS-2 | **`NULLIF` policy expression**, with equivalent `USING` and `WITH CHECK`. Context set transaction-locally by one application helper, which **rejects missing or invalid tenant IDs before any tenant query**. Never session-level | As recommended. The owner made the helper's validation an explicit requirement. The `require_tenant_id()` alternative was not adopted |
| RLS-3 | **Explicit cross-tenant policies** on a dedicated reporting role with its own credentials and pool; read-only policies; minimum table and column privileges. Cross-tenant writes need separate approval and narrow policies. `BYPASSRLS` only for the migration role and a separately controlled break-glass role. Cross-tenant admin operations audited in the application | As recommended |
| RLS-4 | **Role separation is a prerequisite**, in a separate, explicitly approved group, before or alongside the first tenant-owned business table. The API must not use the bootstrap superuser | As recommended. **Implemented in Phase 4 Group 1** (decision log §18), with the migration role **without** `BYPASSRLS` (owner decision G1-5) |

**Deferred:**

| ID | Decision | What it needs |
| -- | -------- | ------------- |
| RLS-5 | **High-volume data-plane (message) tables** are excluded from the adoption. The control-plane decision is not extended to them automatically | A separate evaluation once their design and workload are known: write throughput, partitioning, query patterns, representative volume |

## 12. Performance (owner requirement D): indicative only

**Setup.** This was run manually with `RUN_BENCHMARKS=1 BENCH_COUNT=3 BENCH_TIME=2s bash scripts/rls-evaluation.sh` on 2026-11-26. It is not a CI gate and has no threshold.

- **Machine:** the agent's sandbox, 4 vCPU Intel Xeon at 2.10 GHz, 15 GB RAM, Docker 29.9.3, PostgreSQL 18.7 with default settings and its data on tmpfs.
- **Data:** 1,000,000 synthetic rows over 100 tenants, so 10,000 per tenant, in two identical tables with `PRIMARY KEY (tenant_id, id)` and an index on `(tenant_id, created_at)`. One table has no RLS; the other has RLS forced with the `NULLIF` policy.
- **Per iteration:** each iteration is one complete transaction over the Docker port mapping, using pgx's default cached prepared statements.

**Median of 3 runs, µs per transaction:**

| Query | Plain, explicit filter | Plain + `set_config` | RLS + explicit filter + `set_config` | RLS policy only + `set_config` |
| ----- | ---------------------: | -------------------: | -----------------------------------: | -----------------------------: |
| Point lookup by `(tenant_id, id)` | 297 | 384 | 368 | 382 |
| Latest 50 rows of a tenant | 351 | 482 | 509 | 575 |
| Count of a tenant's 10,000 rows | 1,562 | 1,851 | 1,871 | 1,790 |

Individual runs varied by up to about ±20%. Raw figures and plans are in the PR description.

**Reading the results:**

- **The policy itself.** Comparing RLS with plain, both with `set_config`, the differences go both ways and are within run-to-run noise. The plans explain why **[T]**:
  - **Policy only:** the policy becomes the **index condition**, for example `Index Cond: (tenant_id = (NULLIF(current_setting(...), ''))::uuid)`. The same index is used as with an explicit filter.
  - **Explicit filter as well:** the planner checks the policy once, as a `One-Time Filter` above the same index scan.
- **Setting the context.** `set_config` adds one statement, and so one extra round trip, per transaction: about +90 to +290 µs here, noisy. That depends on network latency, not on RLS.
- **[A] It might be avoidable.** Sending `set_config` and the first query together in one pgx batch might remove the extra round trip. **[NT]**
- **Not covered [NT]:** write throughput, large scans, joins, partitioned tables, concurrency at scale, and production hardware. These are not capacity figures (AGENTS.md §3.1).

## 13. Not tested, and limitations

- **[NT]** External poolers (§9).
- **[NT]** Logging in directly as LOGIN roles. The tests use NOLOGIN roles with `SET ROLE` (§2).
- **[NT]** Partitioned tables; write throughput; production-scale data.
- **[NT]** The cost of the `require_tenant_id()` policy.
- **[NT]** `LEAKPROOF` behaviour, logical replication, and `pg_dump`.
- **[NT]** Statement logging for the cross-tenant role.
- **[NT]** Batching `set_config` with the first query.
- **Version scope.** All results are for PostgreSQL 18.7 and pgx v5.13.0. Re-run the suite (`bash scripts/rls-evaluation.sh`) after upgrading either, and before adopting PostgreSQL 19 (ADR-0005).
