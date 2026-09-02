# AGENTS.md

## Mission

Implement the product defined in `PRD.md` through coherent **Work Packages** composed from the atomic requirements in `IMPLEMENTATION_TASKS.md`.

Optimize for:

- correctness;
- maintainability;
- low context/token usage;
- minimal unrelated change;
- verifiable progress;
- a buildable `main` branch.

A mini task is a planning/checklist unit. It is **not** automatically a branch or pull request.

---

## Source of truth

Use the following precedence:

1. `PRD.md` and approved ADRs for product/architectural decisions;
2. the active Work Package scope;
3. acceptance criteria of the tasks included in that Work Package;
4. this file and `GIT_WORKFLOW.md` for agent execution rules;
5. existing code for implementation facts and established local conventions.

`IMPLEMENTATION_STATUS.md` is **derived progress state**, not a requirements source. If it disagrees with the repository, reconcile it against repository evidence.

Do not change a closed PRD decision without an ADR and approval.

---

## Repository-first continuation

Never assume the next task solely from conversation memory, a previous agent message, branch name, or an outdated status file.

When continuing an existing repository, first inspect the repository and reconcile actual implementation state as defined in `GIT_WORKFLOW.md`.

Completion must be supported by evidence such as:

- migrations/schema;
- implementation code;
- routes/contracts;
- tests;
- configuration/documentation where required;
- commits/PRs when available.

A task ID in a commit or PR title is useful evidence, but it is not sufficient by itself if acceptance criteria are not actually present in the codebase.

---

## Work Package discipline

A Work Package should deliver one coherent, reviewable capability.

Typical characteristics:

- one primary domain/capability;
- usually 2–10 related mini tasks;
- dependencies can be implemented sequentially in the same branch;
- one branch;
- one pull request;
- one final full validation pass.

Larger CRUD/foundation packages may contain more tasks when the change remains cohesive. Security, financial formulas, migrations with high impact, and empirical Bambu work should use smaller packages.

Do not stop for approval between tasks inside an already approved/selected Work Package.

---

## Scope discipline

Within an active Work Package:

### Allowed

- implement all listed tasks;
- satisfy their acceptance criteria;
- add/update tests;
- add required migrations;
- update directly affected documentation;
- perform small local refactors required to implement the capability correctly;
- fix small bugs directly caused by or blocking the Work Package;
- reuse existing abstractions when appropriate.

### Not allowed without stopping/escalating

- change a closed PRD/ADR decision;
- add or replace a structural framework/library without justification and approval;
- cross into a different domain merely because it is convenient;
- weaken security or validation;
- perform unrelated broad refactors;
- rename unrelated modules;
- create speculative abstractions;
- silently expand product scope.

If unrelated work is discovered, record it as a follow-up instead of implementing it opportunistically.

---

## Stop conditions

Stop and report a blocker before making the affected architectural change when completing the Work Package would require any of the following:

- changing a closed architectural/product decision;
- adding a major structural dependency/framework;
- destructive or unexpected migration strategy;
- weakening an authentication/authorization/security invariant;
- creating a Server → Printer or Server → Desktop command path;
- changing financial semantics/formulas not defined by the PRD;
- expanding the package into another major domain;
- resolving contradictory acceptance criteria by guessing.

Continue with unaffected portions only when doing so does not create throwaway work.

---

## Architecture invariants

### Server

- Go.
- PostgreSQL is the only production database.
- Desktop clients never connect directly to PostgreSQL.
- Files are accessed through server authorization.
- Money is stored as integer cents.
- Timestamps are UTC.
- Schema changes require migrations.
- SQL must be parameterized.
- No ORM without approved ADR.

### Desktop

- Windows-only for Release 1.
- Wails + React + TypeScript.
- Server communication through the API.
- Local secrets use Windows secure credential storage where applicable.
- Local cache is disposable.
- Desktop is not a remotely controlled server agent.

### Commercial boundaries

- Do not require customer/order fields for Job creation.
- Do not create revenue records for internal/test/prototype/personal Jobs.
- Do not infer a sale from a quote or order status.
- Pricing workspace must be usable directly from the catalog.
- Saving an official price version is explicit.
- Marketplace API integrations remain out of Release 1.

### Printer security

Never create a Server → Printer control path.

Do not add remote endpoints for:

- start;
- pause;
- resume;
- cancel.

Bambu credentials stay local to the authorized desktop.

Release 1 Bambu integration is telemetry-only plus Bambu Studio launching.

---

## Financial rules

- Never use float for persisted money.
- Do not call markup "margin".
- Channel percentage fees generally apply to sale price, not profit.
- Historical cost and replacement cost are different concepts.
- Cost snapshots are immutable.
- Job quality and printer completion are different concepts.
- Failure risk only applies to exposed costs.
- Batch cost per unit uses good output quantity.
- A Job does not require commercial context.
- Non-commercial Job cost is operational consumption, not commercial loss.
- Pricing calculations may exist without customer, quote, order or sale.
- Marketplace fee profiles are manual assumptions in Release 1; never claim they are live/official unless explicitly sourced by a future integration.
- Quote acceptance must never create an order implicitly.
- Quote → Order conversion requires an explicit action.
- Internal labor cost rate and billable labor/service rate are different concepts.
- Never hide human service value inside machine-hour cost.

Financial formulas require unit tests.

---

## Files

- Uploaded files are immutable objects.
- Never construct storage paths from untrusted original filenames.
- Hash with SHA-256.
- Validate size and type.
- Design changes create new versions instead of overwriting history.

---

## Authentication

- Passwords: Argon2id.
- Never log passwords, tokens, Bambu access codes, DB credentials or encryption secrets.
- Session tokens are stored hashed on the server.
- Authorization is based on permissions.
- Do not bypass RBAC to make tests pass.

---

## Migrations

- One-way numbered migrations.
- A migration may be refined while it exists only inside an unmerged Work Package branch.
- Once merged to `main`, migration history is immutable; later changes require a new migration.
- Fresh database must migrate to latest.
- Migration failure must prevent readiness.
- No schema mutation outside migrations.

---

## Testing

During a Work Package, prefer targeted checks for the affected area so development does not repeatedly pay the cost of the full suite.

Examples:

```text
go test ./internal/auth/...
```

or the equivalent focused desktop tests/checks.

Before opening the Work Package PR, run the complete checks required for every affected application area.

Backend final validation normally includes:

```text
go test ./...
```

plus configured lint/build checks and affected integration tests.

Run race detector when concurrency is touched.

Desktop final validation normally includes:

```text
lint
typecheck
build
```

plus targeted tests.

Do not weaken assertions to make tests pass.

---

## Progress tracking

`IMPLEMENTATION_STATUS.md` must be updated as part of implementation work.

Rules:

- never mark a task completed merely because an agent says it is done;
- mark completion only when its acceptance criteria are evidenced in the current repository state;
- record concise evidence and the verified commit when practical;
- record partial/blocking state when useful;
- tasks absent from the status ledger are not automatically "not started"; they are simply unverified there;
- after repository reconciliation, update stale entries before selecting new work.

Do not create a separate PR solely to flip task status when the status change naturally belongs to a Work Package PR.

---

## Documentation

Update documentation when the Work Package changes:

- public behavior;
- configuration;
- schema assumptions;
- deployment;
- architecture;
- security boundaries.

Use ADRs only for meaningful architectural decisions, not minor implementation details.

---

## Reuse of third-party code

Daedalus and other compatible sources may be used selectively.

When copying or substantially adapting code:

- record repository;
- record commit/ref;
- preserve required license notices;
- update `THIRD_PARTY_NOTICES.md`;
- port only the needed module/logic.

Do not import upstream architecture merely to reuse one feature.

---

## Completion report

At the end of a Work Package, report:

```text
Work Package
Tasks completed/partial
What changed
Files changed
Migrations
Tests/checks run
IMPLEMENTATION_STATUS.md updates
Known limitations
Follow-up tasks discovered
Branch
Pull request (if created)
```

Do not claim tests, CI, merge, or deployment succeeded unless they were actually executed/observed.
