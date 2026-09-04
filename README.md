# Oficina 3D — Specification & Agent Workflow Pack

## Product specification

- `PRD.md` — Product PRD v1.1. Product/domain decisions are unchanged by this workflow revision.

## Agent execution

- `AGENTS.md` — architectural invariants and execution rules for AI agents.
- `GIT_WORKFLOW.md` — Work Package based Git workflow.
- `IMPLEMENTATION_TASKS.md` — atomic Release 1 requirements; tasks are checklists, not PR units.
- `IMPLEMENTATION_STATUS.md` — repository-verified progress ledger.
- `CODEX_CONTINUATION.md` — ready-to-use handoff prompt for an existing repository.

## Development entrypoints

- `cmd/server/` — Go server executable;
- `internal/` — server domain, application, infrastructure, and HTTP layers;
- `desktop/` — Windows Wails application with React and TypeScript;
- `migrations/` — embedded, numbered, one-way PostgreSQL migrations;
- `docs/adr/` — approved architecture decisions (none currently);
- `docs/architecture/` — implemented architecture and security behavior;
- `scripts/` — operational/development script documentation.

The current contiguous implementation frontier is recorded in
`IMPLEMENTATION_STATUS.md`; do not infer it from this README.

## Standard validation

Run commands from the repository root.

Backend:

```powershell
go test ./...
go test -race ./...
go vet ./...
go build -o ./bin/server.exe ./cmd/server
```

PostgreSQL integration tests run when `TALOS_TEST_DATABASE_URL` is set. They are
skipped otherwise. CI supplies a disposable PostgreSQL 18 service.

Desktop frontend:

```powershell
npm.cmd --prefix ./desktop/frontend ci
npm.cmd --prefix ./desktop/frontend run lint
npm.cmd --prefix ./desktop/frontend run typecheck
npm.cmd --prefix ./desktop/frontend run build
```

Desktop native layer:

```powershell
Push-Location ./desktop
go test ./internal/...
go vet ./internal/...
Pop-Location
```

Complete Windows desktop:

```powershell
Push-Location ./desktop
wails build
Pop-Location
```

The pull-request workflow in `.github/workflows/ci.yml` repeats backend and
desktop validation. It does not deploy or publish artifacts.

## Running the server

Go 1.27 or a compatible newer toolchain is required. Copy `.env.example` as a
reference and provide configuration through the process environment; the
server does not load `.env` files itself.

```powershell
$env:TALOS_DATABASE_URL = "postgres://talos:change-me@localhost:5432/talos?sslmode=disable"
go run ./cmd/server
```

Important defaults include a 30-day session lifetime, five login attempts per
minute and peer, a 100 MiB upload limit, `pt-BR`, `BRL`, and
`America/Sao_Paulo`. `TALOS_DATABASE_URL` is required and may contain secrets,
so it must never be logged. See `.env.example` for the complete setting list.
Workshop presentation environment values seed a fresh database only; persisted
settings are not overwritten on restart.

Health routes are `/health/live` and `/health/ready`. Readiness checks
PostgreSQL, migration state, and writable file storage; printer state is
intentionally excluded. Versioned product APIs live below `/api/v1` and return
the standard JSON error envelope documented by the HTTP tests.

Catalog design routes expose item parts at `/api/v1/catalog/items/{itemID}/parts`,
immutable version history at `/api/v1/catalog/parts/{partID}/design-versions`, and
role-based immutable file links at
`/api/v1/catalog/design-versions/{versionID}/files`.

Filament inventory APIs expose material and spool CRUD under
`/api/v1/inventory/materials` and `/api/v1/inventory/spools`. Append-only spool
weighings use `/api/v1/inventory/spools/{spoolID}/measurements`; exact physical
values are JSON decimal strings and persisted as PostgreSQL `NUMERIC`.

Non-filament supplies use `/api/v1/inventory/supplies`, with append-only signed
stock deltas at `/api/v1/inventory/supplies/{supplyID}/movements`. Negative stock
is rejected atomically. `/api/v1/inventory/low-stock` derives low supplies and
active low spools without persisting alert rows; `spool_threshold_g` defaults to
100 grams.

## Authentication and access control

First-owner setup is available through `GET /api/setup/status` and
`POST /api/setup/admin` only until the first identity is created. Desktop login
uses `POST /api/v1/auth/login`, returning the opaque token once; PostgreSQL
stores only its SHA-256 digest.

Bearer authentication rejects missing, malformed, expired, revoked, and
disabled-user sessions uniformly. Last-used timestamps are throttled to avoid a
write on every request. Product handlers authorize concrete permissions through
five fixed Release 1 profiles: Owner, Operator, Designer, Commercial, and
Viewer. Authenticated users can list their device sessions at
`GET /api/v1/auth/sessions` and revoke one with
`POST /api/v1/auth/sessions/{session_id}/revoke`; cross-user management requires
`users.manage`. Details are in `docs/architecture/authentication.md`,
`docs/architecture/authorization.md`, and `docs/architecture/sessions.md`.

Workshop settings are available to authenticated users at
`GET /api/v1/settings`. `PUT /api/v1/settings` requires `settings.manage` and
updates the workshop name, locale, currency, display timezone, and default
theme. The public metadata endpoint reflects the persisted workshop name.
Users with `settings.manage` can upload a validated PNG/JPEG logo with
`POST /api/v1/settings/logo`. Metadata returns the fixed current-logo URL
`/api/v1/meta/logo`, which is safe for pre-login branding because it cannot
address arbitrary files. Logo uploads are capped at 5 MiB or the lower
configured upload limit.

Generic immutable files use `POST /api/v1/files` (`files.upload`) and
`GET /api/v1/files/{file_id}` (`files.read`). Uploads are bounded by
`TALOS_UPLOAD_MAX_BYTES`, hashed with SHA-256, and deduplicated without changing
the first canonical metadata record. Downloads are authorization-checked and
streamed as safe attachments; filesystem paths are never derived from original
filenames.

Catalog items use `/api/v1/catalog/items`. Read operations require
`catalog.read`; create, replace, and delete operations require `catalog.write`.
The list endpoint supports `purpose`, `status`, `sellable`, `tag`, and `q`
filters with bounded `limit`/`offset` pagination. Tags are normalized lowercase
strings stored as a JSONB array, and items can be archived without deletion.
Catalog supply BOM entries use `/api/v1/catalog/items/{itemID}/bom`, with
`catalog.read` for exact replacement-cost previews and `catalog.write` for
CRUD. Preview quantities and cents are exact decimal strings and are not
rounded or persisted as an official price.

Logical, non-sensitive printer records use `/api/v1/printers`. Reads require
`jobs.read`; configuration mutations require `settings.manage`. The server
stores model, nozzle, location, and costing inputs, but never accepts printer
access codes or provides a server-to-printer command path.

Manual print Jobs use `/api/v1/jobs`. They may be created for internal,
prototype, test, maintenance, personal, or production use without commercial
context. State transitions and explicit quality review remain separate, while
significant events atomically retain the acting user and source desktop device.
Per-Job filament usage is recorded at
`/api/v1/jobs/{jobID}/materials`, with a physical spool, role, exact
planned/actual quantities, visible measurement source, distinct historical and
replacement cost snapshots, and exact aggregate gram totals. The desktop Jobs
workspace supports selecting a spool, adding or editing planned/actual usage,
and reviewing totals without introducing commercial context.

The Windows desktop starts with an editable server connection screen. It
persists only a credential-free HTTP(S) base URL in the current user's config
directory and tests the public `/api/v1/meta` contract through the native Go
layer. React does not issue server HTTP requests. The connection test enforces
an eight-second timeout and reports API or minimum-desktop-version
incompatibility before login.

## Docker Compose development

Copy `.env.example` to `.env`, replace the placeholder PostgreSQL password, and
start the stack:

```powershell
Copy-Item .env.example .env
docker compose up -d --build
docker compose ps
```

PostgreSQL is internal-only and is reached as `postgres:5432`. The API binds to
the local host by default. LAN access requires both an explicit bind address
and trusted-LAN declaration; do not expose this HTTP deployment outside a
trusted LAN without TLS.

```dotenv
TALOS_SERVER_BIND_ADDRESS=0.0.0.0
TALOS_TRUSTED_LAN=true
```

Database and object data use named volumes. `docker compose down` preserves
them; deleting volumes is intentionally not part of normal development.

---

## Workflow revision

The previous workflow used:

```text
1 task = 1 branch = 1 pull request
```

This pack replaces it with:

```text
Gate
  ↓
Work Package
  ↓
related mini tasks
  ↓
1 branch
  ↓
1 pull request
```

The mini-task backlog remains granular for precision and verification, while Git/agent execution is grouped by coherent capabilities to reduce repeated context, CI, PR, and review overhead.

---

## Existing repositories

Do not manually assume the current task when adopting this pack mid-development.

Start with `CODEX_CONTINUATION.md`.

The agent must inspect the repository, reconcile actual completed/partial tasks against acceptance criteria, populate `IMPLEMENTATION_STATUS.md`, and only then select the next Work Package.

This avoids both:

- repeating already implemented work;
- falsely marking tasks complete based only on old conversation state.

---

## Release 1.1 domain clarifications retained

- Jobs may be non-commercial.
- Catalog pricing works before quote/order/sale.
- Marketplace fee profiles are manual in Release 1.
- Quotes never become orders implicitly.
- Internal labor cost and billable service price are separate.
