# Git Workflow

This document defines the Git workflow that implementation agents must follow.

It complements:

- `PRD.md` — product and architectural source of truth;
- `IMPLEMENTATION_TASKS.md` — implementation tasks and dependencies;
- `AGENTS.md` — agent execution rules.

`AGENTS.md` remains authoritative for agent behavior.

---

# 1. Core rule

The default unit of implementation is:

```text
1 task
=
1 branch
=
1 pull request
```

Tasks from `IMPLEMENTATION_TASKS.md` must not be combined into the same branch or pull request unless explicitly approved.

A branch must address only the active task.

Unrelated issues discovered during implementation must be reported as follow-up tasks instead of being implemented opportunistically.

---

# 2. Main branch

`main` is the integration branch and must remain in a buildable state.

Agents must not commit directly to `main`.

All implementation changes must reach `main` through a pull request.

Before starting a task:

```bash
git checkout main
git pull --ff-only
```

The working tree must be clean before a new task branch is created.

If unrelated local changes already exist, the agent must not discard, overwrite or include them in the task.

---

# 3. Task dependencies

Before creating a branch, inspect the active task's `Depends on` field in `IMPLEMENTATION_TASKS.md`.

Required dependencies must already be merged into `main`.

Example:

```text
BOOT-002 depends on BOOT-001
```

Therefore:

```text
BOOT-001 merged into main
        ↓
create task/BOOT-002-go-server
```

Independent tasks may be developed in parallel from the same valid `main`.

Do not base a task on another unmerged task branch unless explicitly instructed.

---

# 4. Branch naming

Use:

```text
task/<TASK-ID>-<short-description>
```

Examples:

```text
task/BOOT-001-repository-structure
task/BOOT-002-go-server
task/DB-002-migration-runner
task/AUTH-006-login-endpoint
task/CAT-003-catalog-desktop-ui
task/COST-007-failure-risk
```

The task ID must match `IMPLEMENTATION_TASKS.md`.

Avoid branches such as:

```text
feature/backend
feature/auth
fix-stuff
improvements
misc
agent-changes
```

The branch name must make the active task immediately identifiable.

---

# 5. Starting a task

After updating `main`:

```bash
git checkout -b task/<TASK-ID>-<description>
```

Before implementation, the agent must:

```text
[ ] read AGENTS.md
[ ] read the complete active task
[ ] read referenced/relevant PRD sections
[ ] verify task dependencies
[ ] inspect only relevant existing code
[ ] confirm the task does not require changing a closed architectural decision
```

Do not scan or refactor unrelated areas merely because they could be improved.

---

# 6. Scope discipline

During implementation:

```text
DO:
- implement the active task;
- satisfy its acceptance criteria;
- add/update relevant tests;
- add migrations when required;
- update affected documentation;
- reuse existing abstractions when appropriate.

DO NOT:
- implement another task;
- perform unrelated refactors;
- rename unrelated modules;
- replace frameworks/libraries without approval;
- create speculative abstractions;
- change closed PRD decisions silently;
- fix unrelated issues opportunistically.
```

If another issue is discovered, include it in the completion report as a follow-up.

---

# 7. Architectural changes

If completing the task appears to require changing a closed PRD or architectural decision:

```text
STOP implementation of that architectural change.
```

Do not silently modify the architecture.

Create or propose an ADR according to `AGENTS.md` and wait for approval when approval is required.

Implementation may continue only for portions of the active task that do not violate the existing decision.

---

# 8. Commits

Commits should be small, coherent and limited to the active task.

Use Conventional Commit style with the task ID:

```text
<type>(<scope>): <TASK-ID> <description>
```

Examples:

```text
chore(repo): BOOT-001 initialize repository structure

feat(server): BOOT-002 initialize Go server

feat(auth): AUTH-006 add login endpoint

test(auth): AUTH-006 cover invalid login attempts

docs(config): CFG-001 document environment variables
```

Recommended types:

```text
feat
fix
test
docs
refactor
chore
build
ci
```

Do not create meaningless commit messages such as:

```text
changes
fix
wip
update
final
more fixes
codex changes
```

Temporary local commits are acceptable while working, but the pull request history should clearly represent the task.

---

# 9. Before opening a pull request

The agent must review its own diff.

At minimum:

```bash
git status
git diff main...HEAD
```

Verify:

```text
[ ] active task objective implemented
[ ] acceptance criteria covered
[ ] only task-related files changed
[ ] migrations added when required
[ ] tests added or updated
[ ] relevant tests pass
[ ] lint passes
[ ] affected builds pass
[ ] documentation updated
[ ] no credentials or secrets added
[ ] no mandatory TODO remains
[ ] no closed PRD decision changed silently
```

Tests must actually be executed before being reported as passing.

---

# 10. Pull requests

One task should normally produce one pull request.

PR title:

```text
[TASK-ID] Short task description
```

Example:

```text
[AUTH-006] Implement login endpoint
```

The pull request description must contain:

```markdown
## Task

AUTH-006 — Login endpoint

## What changed

...

## Files changed

...

## Migrations

None / migration identifiers

## Tests run

- `go test ./...`
- ...

## Acceptance criteria

- [x] ...
- [x] ...

## Known limitations

None / ...

## Follow-up tasks discovered

None / ...
```

The PR must not hide unrelated changes.

---

# 11. CI

Pull requests must pass the repository's required CI checks before merge.

The baseline defined by the implementation plan includes, where applicable:

```text
Backend:
- test
- lint
- build

Desktop:
- lint
- typecheck
- build
```

A failing check must not be bypassed by weakening tests or disabling validation.

Fix the implementation or report a genuine blocker.

---

# 12. Review and merge

Agents must not merge their own pull requests unless explicitly authorized.

The default flow is:

```text
task branch
    ↓
Pull Request
    ↓
CI
    ↓
Review
    ↓
Squash and Merge
    ↓
main
```

Prefer **Squash and Merge** because one pull request represents one implementation task.

Recommended resulting commit:

```text
AUTH-006 Implement login endpoint (#42)
```

After merge, delete the task branch.

---

# 13. Tasks blocked by another pull request

Do not normally stack dependent implementation branches.

Instead:

```text
PR A
 ↓
merge into main
 ↓
update main
 ↓
create branch for Task B
```

This keeps each task independently auditable and prevents agents from accidentally importing unfinished work.

Parallel branches are appropriate only when their tasks are independent.

---

# 14. Merge conflicts

When a task branch conflicts with current `main`:

```text
update local main
rebase task branch onto main
resolve only conflicts relevant to the task
rerun affected tests
review diff again
```

Do not use conflict resolution as an opportunity to refactor unrelated code.

Never resolve uncertainty about another contributor's change by deleting it.

---

# 15. Third-party code

When copying or substantially adapting third-party code, including Daedalus:

```text
[ ] record repository
[ ] record source commit/ref
[ ] preserve applicable license notices
[ ] update THIRD_PARTY_NOTICES.md
[ ] port only the required module or logic
```

The third-party architecture must not be imported merely because one implementation detail is reused.

---

# 16. Schema migrations

A schema change requires a migration.

Once a migration has been released/merged as immutable history, do not rewrite it to accommodate a later task.

Create a new migration instead.

Migration files belong to the task that requires the schema change.

---

# 17. Follow-up work

If implementation reveals additional work outside the task:

```text
do not implement it
        ↓
report it
        ↓
create/propose a separate task
        ↓
implement later in its own branch
```

This applies to:

- bugs;
- refactors;
- missing validation;
- architectural improvements;
- documentation gaps outside the active task;
- optimization opportunities.

---

# 18. Release tags

Normal task completion does not create release tags.

Release tags are created only by the release validation process after the Release 1 acceptance criteria pass.

Suggested format:

```text
v1.0.0
```

Do not tag partial gates or ordinary task merges as releases unless explicitly requested.

---

# 19. Agent completion

After opening or completing the pull request, report:

```text
Task
Branch
Commit(s)
Pull request
What changed
Files changed
Migrations
Tests run
CI status
Known limitations
Follow-up tasks discovered
```

Never claim that a test, build, lint check, CI job or merge succeeded unless it was actually executed or observed.

---

# 20. Standard task lifecycle

```text
Select task
    ↓
Check dependencies
    ↓
Update main
    ↓
Create task/<ID>-<slug>
    ↓
Read task + AGENTS + relevant PRD
    ↓
Inspect relevant code
    ↓
Implement only task scope
    ↓
Add/update tests
    ↓
Run required checks
    ↓
Review git diff
    ↓
Commit
    ↓
Push
    ↓
Open PR
    ↓
CI
    ↓
Review
    ↓
Squash merge
    ↓
Delete branch
    ↓
Next eligible task
```