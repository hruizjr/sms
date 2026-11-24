# AI Agent Workflow

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — pending owner review | Project owner | 2026-11-24 |

How AI coding agents receive tasks, plan, implement, test, and submit work — designed for **one human
owner coordinating several AI agents**.

## 1. Roles

| Role | Responsibilities |
| ---- | ---------------- |
| **Project owner** (human) | Defines and prioritizes tasks, answers questions, approves ADRs and dependencies, reviews and merges PRs. The only person who can approve. |
| **Implementing agent** | Executes one task on one branch, following `AGENTS.md`, and submits a PR with a completion report. |
| **Reviewing agent** (optional) | A separate agent session asked by the owner to review a PR against `AGENTS.md` and the task's acceptance criteria. It advises; it does not approve or merge. |

## 2. Task lifecycle

```
 Define ──▶ Clarify ──▶ Plan ──▶ Implement ──▶ Test ──▶ Submit PR ──▶ Review ──▶ Merge ──▶ Close
   owner      agent+owner  agent    agent        agent    agent         owner(+agent) owner    owner
```

### Step 1 — Define (owner)
Tasks are created as GitHub Issues using the **Agent task** template
(`.github/ISSUE_TEMPLATE/agent-task.md`). A good task has:
- a single, clear **objective**;
- **scope**: what is in, what is out, and which folders/files may be touched;
- **acceptance criteria** that can be checked objectively;
- **references**: relevant docs and ADRs;
- **dependencies**: tasks that must be merged first.

Size guideline: a task should produce a PR that can be reviewed in under ~30 minutes (roughly a few
hundred lines of meaningful change). Split anything bigger.

### Step 2 — Clarify (agent)
Before writing code, the agent reads `AGENTS.md`, the referenced documents, and the relevant code. If
anything is ambiguous, conflicts with documentation, or requires an owner decision (`AGENTS.md` §17),
the agent **asks** and waits rather than guessing.

### Step 3 — Plan (agent)
For anything beyond a trivial change, the agent writes a short plan (in the issue, or at the top of the
draft PR) covering:
- approach and affected files/modules;
- any schema, contract, or dependency changes (each requires explicit owner approval);
- test plan;
- risks and assumptions.

For tasks touching hot files, migrations, contracts, billing, security, or providers, the owner approves
the plan before implementation.

### Step 4 — Implement (agent)
- Create a branch from the latest `main` (naming in [git-workflow.md](git-workflow.md)). Until `main`
  exists, branch from the repository's current default branch
  ([git-workflow §0](git-workflow.md#0-current-repository-state-and-required-owner-actions)).
- Make only the changes in scope. Commit in logical steps with Conventional Commit messages.
- Update documentation affected by the change in the same branch.
- Never touch files outside the declared scope; report needed changes elsewhere instead.

### Step 5 — Test (agent)
- Write tests per [testing standards](../standards/testing-standards.md).
- Run formatting, linting, and all relevant tests. Record exact commands and results.
- If tests cannot run (missing tool, environment limitation), say so explicitly — never imply success.

### Step 6 — Submit (agent)
- Push the branch and open a **draft** PR using the PR template.
- Include the completion report (`AGENTS.md` §16).
- Mark ready for review only when all checks pass and the report is complete.

### Step 7 — Review (owner, optionally assisted)
The owner checks (see the review checklist in [git-workflow.md](git-workflow.md)):
- acceptance criteria met; nothing extra;
- tests meaningful and passing;
- schema/contract/dependency changes declared and acceptable;
- assumptions and open issues understood.
The owner may ask a separate reviewing agent for an independent review. Change requests go back to the
implementing agent as PR comments.

### Step 8 — Merge and close (owner)
Squash-merge, delete the branch, close the issue, and update the discovery register / decision log /
roadmap if the task resolved anything.

## 3. Running several agents in parallel

Parallel work is safe when tasks do not overlap. Rules:

1. **Declare file scope in every task.** Two concurrently running tasks must not share files in scope.
2. **Serialize hot files.** Only one open task at a time may modify any of: `AGENTS.md`, `contracts/`,
   migrations, `go.mod`/`go.sum`, `package.json`/lockfiles, CI configuration, `infra/`, shared UI tokens.
3. **Contracts first.** When frontend and backend work depend on a new API, a contract task (OpenAPI
   change) merges first; then backend and frontend tasks proceed in parallel against it.
4. **Timestamp-named migrations** avoid numbering collisions; still, only one migration task in flight.
5. **Rebase/merge from `main` before marking ready**; resolve conflicts within the task's own scope only.
6. A good parallel split is **by module** (e.g. billing vs. contacts), not by layer within one feature.

## 4. What agents must never do

- Approve or merge their own PRs, or push to `main`.
- Mark ADRs or discovery items as accepted/confirmed.
- Add dependencies, infrastructure, or services without approval.
- Invent telco specifications, commercial terms, credentials, or regulatory facts.
- Break the critical correctness rules in `AGENTS.md` §2.1 (no exactly-once claims, no blind retries of
  ambiguous submissions, receipts treated as unreliable, no floating-point money, append-only audit history).
- Use real personal data or send real messages.
- Disable, skip, or weaken tests or checks to get a green build.
- Claim untested work is working.

## 5. Handling uncertainty

| Situation | Agent action |
| --------- | ------------ |
| Requirement ambiguous | Ask the owner; propose options with a recommendation |
| Telco behaviour unknown | Add `TODO(telco-confirm)`, record in the telco checklist, use the simulator |
| Docs genuinely conflict with each other | Stop the affected work; report both documents, sections, and wording; request owner resolution ([`AGENTS.md` §1.1](../../AGENTS.md#11-document-authority-and-conflicts)) |
| Task needs a decision reserved for the owner | Stop; write a Proposed ADR or question |
| Implementation depends on an open discovery item or a Proposed ADR (e.g. monetary scale, idempotency retention, uncertain-status policy) | Stop and ask for owner approval; do not pick a value |
| Found a bug outside scope | Report it in the completion report; do not fix it in this PR |
| Tests fail for reasons outside scope | Report with output; do not "fix" unrelated code silently |

## 6. Session hygiene for agents

- Start each task by re-reading `AGENTS.md` (it may have changed).
- Do not rely on memory from earlier tasks; rely on the repository's documents.
- Keep the PR description updated as the single record of what was done.
