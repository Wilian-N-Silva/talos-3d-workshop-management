# Git Workflow

This document defines the Git workflow that implementation agents must follow.

It complements:

- `PRD.md` — product and architectural source of truth;
- `IMPLEMENTATION_TASKS.md` — atomic implementation requirements and dependencies;
- `IMPLEMENTATION_STATUS.md` — repository-verified progress ledger;
- `AGENTS.md` — agent execution rules.

`PRD.md`/approved ADRs remain authoritative for product and architecture. `AGENTS.md` remains authoritative for agent behavior.

---

# 1. Core rule

The default implementation model is:

```text
Mini Task
= atomic requirement/checklist item

Work Package
= coherent implementation unit
= one branch
= one pull request
```

Do **not** create one branch/PR per mini task by default.

Tasks from `IMPLEMENTATION_TASKS.md` may be combined when they form one coherent, reviewable capability and their dependencies can be satisfied in the same branch.

A Work Package should normally be explainable in one sentence and should not span unrelated domains.

---

# 2. Definitions

## Mini Task

A requirement section such as:

```text
AUTH-007
RBAC-001
CAT-004
COST-007
```

Mini tasks define goals, dependencies, and acceptance criteria. They remain useful for planning, verification, and progress tracking even when several are implemented in one PR.

## Work Package

A temporary execution grouping of related mini tasks.

Example:

```text
WP-AUTH-02 — Access Control & Session Management

- RBAC-001
- RBAC-002
- AUTH-008
```

The exact grouping is derived from repository state and dependencies. Work Packages are not permanent product requirements.

## Gate

A product checkpoint containing multiple capabilities/Work Packages. A Gate is normally too large to be one PR.

---

# 3. Repository state is the execution truth

When resuming work, do not assume the next task from:

- chat history;
- a previous agent summary;
- an old branch name;
- an old `IMPLEMENTATION_STATUS.md` entry;
- numeric task ordering alone.

The repository must be inspected first.

`IMPLEMENTATION_STATUS.md` is a derived ledger and must be reconciled when stale.

---

# 4. Repository reconciliation on continuation

When an agent is asked to continue an existing implementation, perform a bounded repository reconciliation **before selecting new work**.

## 4.1 Inspect Git state

At minimum inspect:

```bash
git status --short
git branch --show-current
git log --oneline --decorate -n 50
```

Also inspect relevant diffs/branches when needed.

If GitHub CLI or equivalent PR metadata is already available/authenticated, it may be used as supplementary evidence. Do not require external access merely to continue development.

## 4.2 Determine the likely implementation frontier

Use commit/task IDs, directory structure, migrations, routes, tests, and existing code to identify the likely current frontier.

Do not spend tokens deeply auditing all future tasks that clearly have no implementation footprint.

Prefer this sequence:

1. discover candidate completed/in-progress task IDs from Git history and code;
2. verify acceptance criteria for those candidates;
3. verify the contiguous dependency chain around the current frontier;
4. inspect adjacent eligible tasks;
5. stop reconciliation once the next useful Work Package can be selected confidently.

## 4.3 Evidence standard

A task is `verified_complete` only when the repository contains evidence satisfying its acceptance criteria.

Evidence can include:

- migration/schema;
- repository/service implementation;
- API route/handler;
- UI behavior/code;
- unit/integration tests;
- configuration/docs required by acceptance criteria;
- merged commit/PR metadata where available.

A task ID appearing in a commit message is not sufficient on its own.

## 4.4 Reconcile progress ledger

Update `IMPLEMENTATION_STATUS.md` with:

- last reconciled commit;
- verified completed tasks;
- partial/blocking tasks when relevant;
- concise evidence;
- currently selected Work Package.

Do not create a standalone PR solely for reconciliation unless explicitly requested. The ledger update may ship with the next coherent Work Package.

---

# 5. Main branch

`main` is the integration branch and must remain buildable.

Agents must not commit directly to `main` unless explicitly authorized.

All implementation changes normally reach `main` through a Work Package pull request.

Before creating a new Work Package branch:

```bash
git checkout main
git pull --ff-only
```

The working tree should be clean.

If unrelated local changes exist:

- do not discard or overwrite them;
- inspect whether they are active implementation work;
- include them only when they clearly belong to the selected Work Package;
- otherwise preserve/report them and avoid contamination.

---

# 6. Selecting a Work Package

After reconciliation, choose the next package from tasks whose dependencies are already satisfied or can be satisfied inside the same package.

A good Work Package normally has:

- one primary capability/domain;
- one demonstrable outcome;
- roughly 2–10 related tasks;
- a reviewable diff;
- no closed architecture change.

These are heuristics, not hard limits.

### Prefer smaller packages for

- auth/security;
- financial formulas;
- destructive/high-impact schema work;
- Bambu/protocol spikes;
- concurrency-sensitive behavior.

### Larger packages are acceptable for

- bootstrap/configuration;
- straightforward CRUD vertical slices;
- tightly coupled API + UI + tests;
- repetitive foundation work.

Do not combine tasks merely to reduce PR count if they do not form a coherent capability.

---

# 7. Work Package naming

Use a stable package ID:

```text
WP-<AREA>-<NN>
```

Examples:

```text
WP-AUTH-02
WP-INV-01
WP-JOB-03
WP-PRICE-01
```

Branch naming:

```text
work/<wp-id-lowercase>-<short-description>
```

Examples:

```text
work/wp-auth-02-access-control
work/wp-inv-01-materials
work/wp-price-01-channel-pricing
```

Avoid vague branch names such as:

```text
feature/backend
fix-stuff
improvements
misc
agent-changes
```

---

# 8. Starting a Work Package

Before implementation:

```text
[ ] reconcile repository if this is a continuation
[ ] read AGENTS.md
[ ] read this workflow
[ ] read selected task sections
[ ] read relevant PRD sections
[ ] verify dependencies
[ ] inspect relevant existing code
[ ] record the selected WP in IMPLEMENTATION_STATUS.md
[ ] confirm no closed architectural decision must change
```

Then create the branch:

```bash
git checkout -b work/<WP-ID>-<description>
```

---

# 9. Executing tasks inside a Work Package

Tasks inside the same Work Package may be implemented sequentially without intermediate PRs or approval.

Use task IDs as the internal checklist.

Example:

```text
WP-AUTH-02

[ ] RBAC-001
[ ] RBAC-002
[ ] AUTH-008
```

During implementation:

```text
DO:
- implement all WP tasks;
- satisfy acceptance criteria;
- add/update tests;
- add migrations when required;
- update directly affected docs;
- make small local refactors required by the capability.

DO NOT:
- change closed PRD decisions silently;
- add unrelated features;
- perform broad unrelated refactors;
- cross into another major domain without re-scoping;
- weaken tests/security to make checks pass.
```

Do not stop merely because one mini task ended. Continue until the Work Package is complete or a defined stop condition is reached.

---

# 10. Stop conditions

Pause the affected work and report when the package requires:

- changing a closed PRD/ADR decision;
- adding/replacing a major framework or structural dependency;
- destructive/unexpected migration strategy;
- weakening authentication, authorization, or another security invariant;
- creating Server → Printer or Server → Desktop command control;
- inventing a financial formula/domain rule absent from the PRD;
- expanding substantially into a different domain;
- resolving contradictory acceptance criteria by assumption.

Do not use stop conditions for ordinary implementation details already within the package's intent.

---

# 11. Commits

Commits should be coherent and may align with mini tasks or meaningful substeps.

Use Conventional Commit style and include task IDs when useful:

```text
<type>(<scope>): <TASK-ID> <description>
```

Examples:

```text
feat(rbac): RBAC-001 define roles and permissions
feat(auth): RBAC-002 enforce route permissions
feat(auth): AUTH-008 add session management
```

A Work Package may contain several such commits.

Avoid meaningless messages such as:

```text
changes
fix
wip
update
final
more fixes
codex changes
```

Squash merge may later collapse the package history on `main`.

---

# 12. Schema migrations

A schema change requires a migration.

Within an **unmerged Work Package branch**, a new migration created for that package may be refined as the capability evolves.

Once the migration is merged into `main`, treat it as immutable history.

Later schema changes require a new migration.

This avoids unnecessary migration churn caused solely by artificial mini-task boundaries while preserving released history.

---

# 13. Testing strategy

Do not repeatedly run every repository check after every mini task unless the change warrants it.

During implementation, prefer targeted checks:

```text
relevant unit tests
relevant integration tests
package/module lint/typecheck when available
```

Before opening the Work Package PR, run the complete required validation for all affected areas.

Baseline, where applicable:

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

CI runs the required suite again.

Tests must actually be executed before being reported as passing.

---

# 14. Progress tracking

`IMPLEMENTATION_STATUS.md` is the progress ledger.

Update it in the same Work Package branch.

For each task completed in the package, record:

- task ID;
- status;
- concise evidence;
- verification commit/ref when practical;
- notes only when needed.

Recommended statuses:

```text
verified_complete
partial
blocked
superseded
```

Tasks not listed are simply unverified in the ledger; do not automatically label all absent tasks as `not_started`.

Before PR creation, ensure completed tasks in the Work Package are reflected in the ledger.

Do not create one PR solely to mark a task complete.

---

# 15. Before opening a pull request

Review the full Work Package diff:

```bash
git status
git diff main...HEAD
```

Verify:

```text
[ ] WP outcome implemented
[ ] every included task acceptance criteria reviewed
[ ] no unrelated domain changes
[ ] migrations correct
[ ] tests added/updated
[ ] targeted checks passed
[ ] final required checks passed
[ ] docs updated
[ ] IMPLEMENTATION_STATUS.md updated
[ ] no credentials/secrets added
[ ] no mandatory TODO remains
[ ] no closed PRD decision changed silently
```

---

# 16. Pull requests

One coherent Work Package should normally produce one pull request.

PR title:

```text
[WP-ID] Short capability description
```

Example:

```text
[WP-AUTH-02] Add access control and session management
```

Recommended description:

```markdown
## Work Package

WP-AUTH-02 — Access Control & Session Management

## Tasks

- [x] RBAC-001
- [x] RBAC-002
- [x] AUTH-008

## What changed

...

## Files changed

...

## Migrations

None / migration identifiers

## Tests/checks run

- `go test ./...`
- ...

## Acceptance criteria

- [x] ...

## Progress ledger

- `IMPLEMENTATION_STATUS.md` updated

## Known limitations

None / ...

## Follow-ups discovered

None / ...
```

The PR must not hide unrelated changes.

---

# 17. CI

Pull requests must pass required CI before merge.

A failing check must not be bypassed by weakening tests or disabling validation.

Fix the implementation or report a genuine blocker.

---

# 18. Review and merge

Default flow:

```text
repository reconciliation
    ↓
select Work Package
    ↓
work/<WP-ID>-<slug>
    ↓
implement related mini tasks
    ↓
targeted checks during work
    ↓
final full validation
    ↓
update IMPLEMENTATION_STATUS.md
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

Agents must not merge their own PRs unless explicitly authorized.

Prefer Squash and Merge when one PR represents one coherent Work Package.

After merge, delete the Work Package branch.

---

# 19. Dependencies and parallel work

Tasks inside one Work Package may depend on each other and do not need intermediate merges.

Independent Work Packages may run in parallel from the same valid `main` when merge-conflict risk is acceptable.

Avoid stacking long chains of unmerged Work Package branches unless explicitly useful; prefer merging a coherent package before starting another package that depends on it.

---

# 20. Merge conflicts

When a Work Package branch conflicts with current `main`:

```text
update local main
rebase package branch onto main
resolve only relevant conflicts
rerun affected tests
review full diff again
```

Do not use conflict resolution to delete uncertain changes or perform unrelated refactors.

---

# 21. Third-party code

When copying or substantially adapting third-party code, including Daedalus:

```text
[ ] record repository
[ ] record source commit/ref
[ ] preserve applicable license notices
[ ] update THIRD_PARTY_NOTICES.md
[ ] port only required module/logic
```

Do not import third-party architecture merely because one implementation detail is reused.

---

# 22. Follow-up work

A discovered issue may be fixed inside the current Work Package when it is:

- directly required by the package acceptance criteria;
- a small bug blocking the package;
- a small local refactor required for correctness;
- a missing test/validation necessary for the capability.

Create/report follow-up work when it:

- belongs to another domain;
- adds a new product feature;
- changes architecture;
- is optional optimization/cleanup;
- would materially expand the package.

---

# 23. Release tags

Normal Work Package completion does not create release tags.

Release tags are created only after release validation.

Suggested format:

```text
v1.0.0
```

---

# 24. Agent completion

After opening/completing a Work Package PR, report:

```text
Work Package
Tasks
Branch
Commits
Pull request
What changed
Files changed
Migrations
Tests/checks run
CI status
IMPLEMENTATION_STATUS.md changes
Known limitations
Follow-ups discovered
```

Never claim a test, build, CI job, merge, or deployment succeeded unless actually executed/observed.
