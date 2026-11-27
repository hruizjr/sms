# Git and Pull Request Workflow

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Approved (ADR-0017 accepted 2026-11-25) | Project owner | 2026-11-27 (Windows frontend CI job removed) |

A deliberately simple, trunk-based workflow for **one human owner and multiple AI agents**.

## 0. Repository state

**As of 2026-11-25** (ADR-0017 accepted by the owner):

| Item | State |
| ---- | ----- |
| `main` branch | Exists; holds the merged Phase 1 and Phase 2A history |
| Default branch | `main` |
| Branch protection on `main` | Configured by the owner as **repository ruleset "R1"** (active, targets `refs/heads/main`): pull requests required, zero required approvals, conversation resolution required, force pushes (non-fast-forward) and deletion blocked. Classic branch protection is not enabled. *Verified by an agent on 2026-11-26 (Phase 3 Group 6), see below.* |
| Required CI status checks | **Nine checks**, added to ruleset R1 by the owner on 2026-11-26 (owner decision L9). They are listed in "Required status checks" below. Before that, only the first three were required (agent-verified 2026-11-26, Group 6). "Require branches to be up to date" was off at that verification |

Agents create every task branch from the latest `main` and open Draft pull requests against `main`.
Agents MUST NOT change the default branch, branch protection, or any other repository setting; these remain
owner actions.

**How the state above was verified (2026-11-26).** Read-only GitHub REST API calls from the Group 6 agent
session:
- `GET /repos/hruizjr/sms/rules/branches/main` and `GET /repos/hruizjr/sms/rulesets/28042656` returned the
  rules above for ruleset R1, active since 2026-11-25.
- `GET /repos/hruizjr/sms/branches/main` reported `protected: true` with classic protection
  `enabled: false` and no classic required checks.
- The classic protection endpoint itself (`/branches/main/protection`) was not readable with the
  session's permissions (403). Its settings are known only through the summary above.
- Ruleset bypass actors are not visible to these calls.

Re-verify after any change in repository settings.

**Required status checks** (owner decision L9; ruleset R1 updated by the owner on 2026-11-26). How this was
checked is in the Phase 3 closure pull request and [decision log §13](decision-log.md#13-phase-3-closure-2026-11-26).
The names are the job names exactly as they appear in the workflows; every workflow runs on every pull request
to `main`, with no path filters:

| Check | Workflow | Required? |
| ----- | -------- | --------- |
| Go test and build | Backend CI | Yes |
| Go lint | Backend CI | Yes |
| Go vulnerability check | Backend CI | Yes |
| Go generate check (Windows) | Backend CI | Yes |
| OpenAPI contract check | Contracts CI | Yes |
| Frontend check | Frontend CI | Yes |
| Local stack check | Local Stack CI | Yes |
| Secret scan | Security CI | Yes |
| Dependency review | Security CI | Yes. Runs on pull requests only |
| Local stack config (Windows) | Local Stack CI | No: optional |

The optional **Frontend check (Windows)** job was removed from Frontend CI by owner decision on 2026-11-27
([decision log §10](decision-log.md#10-owner-decisions--frontend-foundation-phase-3-group-5-2026-11-25), status
update). It was never a required check. Windows compatibility of the frontend is validated by the owner's local
Windows testing; the Windows jobs of Backend CI and Local Stack CI are unchanged.

The tool-assigned branch `claude/ecstatic-gates-ji0o5b` is reused by agent sessions as their working branch
(Phase 3 Groups 4–7 and the Phase 3 closure), updated from `main` by merge rather than history rewrite; its pull requests are
squash-merged.

## 1. Branches

- **`main`** — always deployable (once there is something to deploy), always passing checks. Protected.
- **Task branches** — short-lived, one per task, created from the latest `main`, deleted after merge.
- No `develop`, `release`, or long-lived feature branches (revisit when there are production releases to
  manage).

### Branch naming

```
<type>/<task-id>-<short-kebab-description>
```

| Type | Use |
| ---- | --- |
| `feat` | New functionality |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `refactor` | Code change with no behaviour change |
| `test` | Tests only |
| `chore` | Tooling, CI, dependencies, configuration |

Examples: `feat/42-credit-reservation`, `docs/17-telco-checklist-globe`, `chore/9-ci-go-lint`.

Some agent platforms assign branch names automatically (e.g. `claude/<random-name>`). That is acceptable;
the **PR title must then include the task ID**.

## 2. Commits

[Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>(<scope>): <summary in imperative mood>

<optional body: why, not what>

Refs: #<issue>
```

Example: `feat(billing): reserve credits when a message is accepted`

Commits must not contain secrets, generated binaries, or unrelated changes.

## 3. Pull requests

- One task → one PR. Open as **draft**; mark ready when checks pass and the completion report is filled in.
- Use the PR template (`.github/pull_request_template.md`). Required sections: summary, linked task,
  changed files, schema/contract/dependency changes, tests run, assumptions, open questions.
- Title: Conventional Commit style including the task ID, e.g. `feat(billing): credit reservation (#42)`.
- Keep PRs small; split large work into sequential PRs.

## 4. Review checklist (for the owner or a reviewing agent)

- [ ] Linked task; acceptance criteria all met; no out-of-scope changes.
- [ ] Follows `AGENTS.md` (structure, error handling, logging without PII, tenant scoping).
- [ ] Tests added for the behaviour; mandatory areas covered; checks green.
- [ ] Schema/contract changes declared, backward compatible or explicitly approved.
- [ ] New dependencies justified and approved.
- [ ] No secrets, real personal data, or real phone numbers.
- [ ] Documentation updated (discovery register, ADRs, READMEs) where affected.
- [ ] Assumptions and open questions acceptable or tracked.
- [ ] Integration claims match the evidence (simulator vs. telco test environment vs. mock).

## 5. Merging

- **Only the project owner merges.**
- **Squash merge** by default (one clean commit per task on `main`). The squash commit message uses the
  PR title.
- Delete the branch after merge.

## 6. Branch protection settings for `main` (recommended)

Configured in GitHub → Settings → Rules/Rulesets or Branches (see §0 for the current configuration):

| Setting | Recommendation | Why |
| ------- | -------------- | --- |
| Require a pull request before merging | **On** | No direct pushes to `main` |
| Required approvals | See note below | |
| Require status checks to pass | **On** (once CI exists) | Broken code cannot merge |
| Require branches to be up to date | **On** | Avoids merging against stale `main` |
| Require conversation resolution | **On** | Review comments are not ignored |
| Block force pushes | **On** | History cannot be rewritten |
| Block deletion | **On** | `main` cannot be deleted |
| Restrict who can push/merge | Owner only | Agents cannot merge |
| Allowed merge methods | Squash only | Clean history |

**Note on approvals:** GitHub does not allow the author of a PR to approve their own PR. If agents open PRs
under the owner's own GitHub account, a "required approvals ≥ 1" rule would block every merge. In that case
set required approvals to **0** and treat the owner's merge as the approval; the other protections still
prevent direct pushes and failing builds. If agents use a separate bot account, require **1** approval.

## 7. Releases (later)

When deployment exists: tag releases on `main` with semantic versions (`v0.1.0`), maintain a changelog,
and document deployment and rollback in `docs/operations/`. Not needed in Phase 1.

## 8. Recovery rules

- A bad merge is fixed with a **revert PR**, not by force-pushing `main`.
- A leaked secret is **rotated first**, then removed; the incident is recorded.
