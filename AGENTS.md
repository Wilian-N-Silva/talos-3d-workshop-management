# AGENTS.md

## Mission

Implement the product defined in `PRD.md` using small, auditable tasks from `IMPLEMENTATION_TASKS.md`.

Optimize for correctness, maintainability, low context usage, and minimal unrelated change.

---

## Source of truth

Priority:

1. active task;
2. `PRD.md`;
3. approved ADRs;
4. this file;
5. existing code.

Do not change a closed PRD decision without an ADR and approval.

---

## Task discipline

For every task:

- implement only the requested scope;
- do not perform broad refactors;
- do not rename unrelated modules;
- do not replace frameworks or libraries unless requested;
- do not add speculative abstractions;
- do not silently expand scope;
- do not leave the repository broken;
- document assumptions when unavoidable.

If a task exposes a separate issue, record it as a follow-up task instead of solving it opportunistically.

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
- Never edit a migration already considered released.
- Fresh database must migrate to latest.
- Migration failure must prevent readiness.
- No schema mutation outside migrations.

---

## Testing

Backend task completion normally requires:

```text
go test ./...
```

and affected integration tests.

Run race detector when concurrency is touched.

Desktop task completion normally requires:

```text
lint
typecheck
build
```

plus targeted tests.

Do not weaken assertions to make tests pass.

---

## Documentation

Update documentation when the task changes:

- public behavior;
- configuration;
- schema assumptions;
- deployment;
- architecture;
- security boundaries.

Use ADRs only for meaningful architectural decisions, not minor implementation details.

---

## Reuse of third-party code

Daedalus and other MIT sources may be used selectively.

When copying or substantially adapting code:

- record repository;
- record commit/ref;
- preserve required license notices;
- update `THIRD_PARTY_NOTICES.md`;
- port only the needed module/logic.

Do not import upstream architecture merely to reuse one feature.

---

## Completion report

At the end of a task, report:

```text
What changed
Files changed
Migrations
Tests run
Known limitations
Follow-up tasks discovered
```

Do not claim tests passed unless they were actually run.
