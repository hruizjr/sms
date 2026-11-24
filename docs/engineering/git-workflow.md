# Git and Pull Request Workflow

| Status | Owner | Last updated |
| ------ | ----- | ------------ |
| Draft — pending owner approval (ADR-0017) | Project owner | 2026-11-24 (current state added) |

A deliberately simple, trunk-based workflow for **one human owner and multiple AI agents**.

## 0. Current repository state and required owner actions

**As of 2026-11-24, the target workflow below is not yet in place.**

| Item | Current state |
| ---- | ------------- |
| `main` branch | **Does not exist** |
| Default branch | `claude/ecstatic-gates-ji0o5b` (it became the default because it was the first branch pushed to an empty repository) |
| Branch protection | **Not configured** |
| Open documentation PRs | Target `claude/ecstatic-gates-ji0o5b`, because no `main` exists |

AI agents MUST NOT create `main`, change the default branch, or change repository settings. These are
**owner actions**:

1. **Review and merge** any open Phase 1 documentation PRs into `claude/ecstatic-gates-ji0o5b`, so the
   history to be promoted is complete.
2. **Create `main`** from that branch's latest commit — either on GitHub (branch selector → type `main` →
   "Create branch: main from claude/ecstatic-gates-ji0o5b"), or locally:

   ```bash
   git fetch origin
   ```

   ```bash
   git push origin origin/claude/ecstatic-gates-ji0o5b:refs/heads/main
   ```

3. **Set `main` as the default branch**: repository Settings → General → Default branch (GitHub menu
   names may change over time).
4. **Configure branch protection** for `main` as described in §6 (Settings → Rules/Rulesets or Branches).
5. **Retarget** any still-open PRs to `main` (PR page → Edit → base branch).
6. Optionally, delete `claude/ecstatic-gates-ji0o5b` once nothing depends on it.
7. From then on, **every task branch is created from the latest `main`**, and `AGENTS.md` §14 "Current state"
   and this §0 are updated to reflect it (a small docs task).

Until these actions are done, agents branch from the current default branch and target PRs at it.

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

Configure in GitHub → Settings → Branches (or Rulesets) once the repository has a `main` branch
(see §0 — not yet configured):

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
