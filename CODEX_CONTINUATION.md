# Codex Continuation Prompt

Use this prompt when handing an existing repository to a coding agent after adopting the Work Package workflow.

---

## Prompt

Continue implementation of this repository from its **actual current state**.

Do not assume the project is currently at `AUTH-007` or at any other task based on conversation history. Determine the real implementation frontier by inspecting the repository.

### Read first

1. `PRD.md`
2. `AGENTS.md`
3. `GIT_WORKFLOW.md`
4. `IMPLEMENTATION_TASKS.md`
5. `IMPLEMENTATION_STATUS.md`
6. approved ADRs, if any

### Phase 1 — Repository reconciliation

Before selecting new work:

1. inspect current Git state and recent history;
2. inspect existing migrations, server/desktop code, routes, tests, and relevant documentation;
3. identify task IDs that appear completed or partially implemented;
4. verify those candidate tasks against their actual acceptance criteria;
5. identify the contiguous dependency frontier and adjacent eligible work;
6. do not deeply audit clearly future/unimplemented modules merely to enumerate all 136 tasks;
7. update `IMPLEMENTATION_STATUS.md` with repository-verified evidence and the commit used as the reconciliation baseline.

A task ID in a commit/branch/PR is evidence, but is not enough by itself. Repository behavior/code/tests must support the acceptance criteria.

If the working tree or current branch contains unfinished work, preserve it. Determine whether it belongs to an in-progress requirement rather than discarding or overwriting it.

### Phase 2 — Select the next Work Package

Based on the reconciled repository state:

1. group the next related eligible mini tasks into one coherent Work Package;
2. prefer one reviewable capability rather than one PR per task;
3. respect task dependencies;
4. keep security/financial/Bambu packages smaller;
5. record the Work Package in `IMPLEMENTATION_STATUS.md`.

Do not ask for approval between mini tasks inside that selected package.

### Phase 3 — Implement

Implement every task in the selected Work Package sequentially.

Use targeted tests/checks during iteration. Run the complete required validation for all affected areas once the Work Package is complete.

You may make small local refactors or fixes that are directly required for the package to work correctly. Record unrelated improvements as follow-ups instead of expanding scope.

Stop before making a change only if it hits a stop condition defined by `AGENTS.md`/`GIT_WORKFLOW.md`, such as changing a closed architecture decision, weakening security, inventing a financial rule, or crossing substantially into another domain.

### Phase 4 — Record progress and prepare one PR

Before opening the PR:

1. update `IMPLEMENTATION_STATUS.md` for every completed/partial task in the package;
2. review `git diff main...HEAD`;
3. run the final required checks;
4. ensure migrations/docs are correct;
5. open **one PR for the Work Package**, not one PR per mini task.

PR title format:

```text
[WP-ID] Short capability description
```

PR description must list all included task IDs and their acceptance status.

### Completion report

Report:

```text
Reconciliation baseline
Verified tasks discovered
Partial/blocking tasks discovered
Selected Work Package
Tasks implemented
Branch
Commits
PR
Tests/checks run
Status ledger updates
Known limitations
Follow-ups
```

Never claim a task, test, CI check, PR merge, or deployment succeeded unless it was actually verified.
