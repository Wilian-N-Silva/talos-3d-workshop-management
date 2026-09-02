# Implementation Tasks — Release 1

> This file decomposes `PRD.md` into atomic implementation requirements suitable for AI coding agents.
>
> **Mini tasks are planning and verification units, not Git units.** Related tasks should normally be grouped into coherent Work Packages according to `GIT_WORKFLOW.md`.
>
> Progress is recorded in `IMPLEMENTATION_STATUS.md`, not by rewriting task definitions here.

---

# Task semantics

Each task defines:

- an atomic goal;
- dependencies;
- acceptance criteria.

Tasks remain intentionally small because they:

- reduce ambiguity for agents;
- make missing requirements visible;
- allow precise verification;
- allow a Work Package to be split when it becomes too large;
- support reliable progress reconciliation from repository evidence.

They do **not** imply:

```text
1 task = 1 branch = 1 pull request
```

The default execution model is:

```text
Gate
  ↓
Work Package
  ↓
2..N related mini tasks as checklist
  ↓
1 branch
  ↓
1 pull request
```

---

# Work Package rules

When selecting work:

```text
[ ] reconcile repository state when continuing an existing implementation
[ ] consult IMPLEMENTATION_STATUS.md, but verify stale/uncertain entries against code
[ ] select tasks with satisfied dependencies or dependencies implementable inside the same package
[ ] keep one primary capability/domain per Work Package
[ ] prefer a reviewable vertical slice over microscopic PRs
[ ] do not combine unrelated tasks merely to reduce PR count
```

Typical package sizing guidance:

- bootstrap/config/straightforward CRUD: may group several tasks;
- API + UI + tests for one capability: usually one package;
- auth/security: smaller packages;
- financial formulas: smaller packages;
- schema/domain-critical work: moderate packages;
- Bambu/protocol experimentation: very small packages/spikes.

These are heuristics, not fixed numeric limits.

---

# Task execution rules

Before starting a Work Package:

```text
[ ] read AGENTS.md
[ ] read GIT_WORKFLOW.md
[ ] read selected task sections
[ ] read relevant PRD sections
[ ] identify/verify dependencies
[ ] inspect relevant existing code
[ ] record selected Work Package in IMPLEMENTATION_STATUS.md
```

During a Work Package:

```text
[ ] execute selected tasks sequentially without intermediate PRs
[ ] use targeted tests/checks while iterating
[ ] keep scope within the package capability
[ ] report meaningful unrelated follow-ups instead of expanding scope
```

Before finishing a Work Package:

```text
[ ] all included acceptance criteria reviewed
[ ] final required tests/checks run
[ ] no unrelated broad refactor
[ ] docs updated
[ ] IMPLEMENTATION_STATUS.md reconciled/updated
[ ] follow-ups reported
```

---

# Progress status

Do not add completion marks to task headings in this file.

Use `IMPLEMENTATION_STATUS.md` so that:

- requirements remain stable;
- progress can be regenerated/reconciled from repository evidence;
- stale agent claims do not silently become specification truth;
- the backlog does not accumulate merge noise from checkbox-only edits.

A task should only be recorded as `verified_complete` when its acceptance criteria are evidenced in the repository.

---

# Gate 0 — Bootstrap

## BOOT-001 — Initialize repository structure

**Goal:** Create initial server/desktop/docs directory structure without implementing business features.

**Depends on:** none.

**Acceptance:**

- repository has server entrypoint location;
- desktop location exists;
- migrations/docs directories exist;
- `PRD.md`, `AGENTS.md`, `IMPLEMENTATION_TASKS.md` live at root;
- README explains development entrypoints;
- no database feature implemented yet.

---

## BOOT-002 — Initialize Go server module

**Goal:** Create buildable Go server executable.

**Depends on:** BOOT-001.

**Acceptance:**

- `go build ./cmd/server` succeeds;
- server starts with minimal config;
- graceful shutdown handles SIGINT/SIGTERM;
- no product routes beyond health placeholder.

---

## BOOT-003 — Initialize Wails desktop shell

**Goal:** Create Windows-targeted Wails application with React + TypeScript.

**Depends on:** BOOT-001.

**Acceptance:**

- desktop dev mode launches;
- production build command is documented;
- React renders placeholder shell;
- TypeScript strict enabled.

---

## BOOT-004 — Add lint/test/build commands

**Goal:** Standardize commands agents must run.

**Depends on:** BOOT-002, BOOT-003.

**Acceptance:**

- backend test command;
- backend lint command;
- desktop lint;
- desktop typecheck;
- desktop build;
- commands documented.

---

## BOOT-005 — Add CI baseline

**Goal:** Run baseline quality checks on pull requests.

**Depends on:** BOOT-004.

**Acceptance:**

- Go test/lint/build;
- desktop lint/typecheck/build;
- no deployment step;
- CI passes on clean repository.

---

# Gate 1 — Server Foundation

## CFG-001 — Server configuration package

**Goal:** Centralize environment configuration.

**Depends on:** BOOT-002.

**Scope:**

- port;
- database connection;
- data directory;
- trusted LAN mode;
- upload size;
- locale/currency/timezone defaults.

**Acceptance:**

- validation errors are explicit;
- secrets are not logged;
- `.env.example` has placeholders.

---

## DB-001 — Add PostgreSQL driver and connection

**Goal:** Server connects to PostgreSQL.

**Depends on:** CFG-001.

**Acceptance:**

- connection uses `database/sql` + pgx driver;
- connection pool settings configurable;
- startup fails clearly when DB unavailable;
- no ORM.

---

## DB-002 — Add migration runner

**Goal:** Add numbered PostgreSQL migrations.

**Depends on:** DB-001.

**Acceptance:**

- empty DB migrates to latest;
- migration failure blocks readiness;
- migration state test exists;
- locking behavior documented.

---

## DEP-001 — Docker Compose PostgreSQL

**Goal:** Run PostgreSQL locally through Docker Compose.

**Depends on:** DB-001.

**Acceptance:**

- DB not published to host by default;
- named volume;
- healthcheck;
- server can reach DB by service name.

---

## DEP-002 — Dockerize server

**Goal:** Add production-like server container.

**Depends on:** CFG-001, DEP-001.

**Acceptance:**

- multi-stage build;
- non-secret config via environment;
- persistent data volume;
- container starts with Compose.

---

## OBS-001 — Liveness endpoint

**Goal:** Implement `/health/live`.

**Depends on:** BOOT-002.

**Acceptance:**

- does not require DB;
- returns stable JSON;
- tested.

---

## OBS-002 — Readiness endpoint

**Goal:** Implement `/health/ready`.

**Depends on:** DB-002.

**Acceptance:**

- verifies PostgreSQL;
- verifies migration state;
- verifies storage directory;
- printer state does not affect readiness.

---

## STOR-001 — File storage interface

**Goal:** Define server-side immutable object storage contract.

**Depends on:** CFG-001.

**Acceptance:**

- put/open/delete-exclusively-for-cleanup or equivalent;
- no catalog coupling;
- path traversal impossible through API contract.

---

## STOR-002 — Local filesystem storage implementation

**Goal:** Store objects in server data volume.

**Depends on:** STOR-001.

**Acceptance:**

- content-addressed or opaque storage key;
- SHA-256 supported;
- original filename not used as physical path;
- tests use temporary directory.

---

## API-001 — API v1 router and error envelope

**Goal:** Establish `/api/v1` and consistent errors.

**Depends on:** BOOT-002.

**Acceptance:**

- standardized error envelope;
- JSON content type;
- request ID support hook;
- tests.

---

## API-002 — Meta/version endpoint

**Goal:** Add `/api/v1/meta`.

**Depends on:** API-001.

**Acceptance:**

- returns server version;
- API version;
- minimum desktop version;
- workshop name placeholder/config source.

---

# Gate 2 — Authentication & Settings

## AUTH-001 — Users schema

**Goal:** Create user persistence.

**Depends on:** DB-002.

**Acceptance:**

- migration;
- status field;
- unique login identifier;
- timestamps;
- repository tests against PostgreSQL.

---

## AUTH-002 — Argon2id password service

**Goal:** Hash and verify passwords.

**Depends on:** AUTH-001.

**Acceptance:**

- no plain passwords persisted;
- parameters centralized;
- verification tests;
- malformed hash handled safely.

---

## AUTH-003 — First admin bootstrap

**Goal:** Create first Owner when no users exist.

**Depends on:** AUTH-002, API-001.

**Acceptance:**

- setup status endpoint;
- setup endpoint works only with zero users;
- race-safe first-user creation;
- endpoint closes after creation.

---

## AUTH-004 — Devices schema

**Goal:** Track desktop installations for audit.

**Depends on:** AUTH-001.

**Acceptance:**

- migration;
- device display name;
- OS;
- app version;
- last seen.

---

## AUTH-005 — Session schema and token generation

**Goal:** Add opaque sessions.

**Depends on:** AUTH-004.

**Acceptance:**

- secure random token;
- only token hash stored;
- expiry;
- revoked timestamp;
- device relationship.

---

## AUTH-006 — Login endpoint

**Goal:** Authenticate user and create session.

**Depends on:** AUTH-002, AUTH-005.

**Acceptance:**

- invalid credentials do not reveal which field failed;
- login rate limit;
- last login updated;
- tests.

---

## AUTH-007 — Auth middleware

**Goal:** Resolve authenticated user from bearer session.

**Depends on:** AUTH-006.

**Acceptance:**

- expired/revoked token rejected;
- last-used update strategy does not write on every request unnecessarily;
- current user available to handlers.

---

## RBAC-001 — Permissions and roles

**Goal:** Define initial permissions and roles.

**Depends on:** AUTH-001.

**Acceptance:**

- permissions enumerated;
- role-to-permission mapping;
- Owner, Operator, Designer, Commercial, Viewer;
- tests for representative permission matrix.

---

## RBAC-002 — Authorization helper/middleware

**Goal:** Enforce permissions consistently.

**Depends on:** RBAC-001, AUTH-007.

**Acceptance:**

- handler/service can require permission;
- forbidden vs unauthenticated distinguished;
- tests.

---

## AUTH-008 — Session list/revoke endpoints

**Goal:** Allow user/admin to revoke devices/sessions.

**Depends on:** AUTH-007.

**Acceptance:**

- current user's sessions listable;
- revocation invalidates token;
- admin capability follows permission.

---

## SET-001 — Workshop settings schema

**Goal:** Persist workshop name, locale, currency, timezone, theme.

**Depends on:** DB-002.

**Acceptance:**

- one logical settings record;
- sensible defaults;
- validation.

---

## SET-002 — Workshop settings API

**Goal:** Read/update settings using RBAC.

**Depends on:** SET-001, RBAC-002.

**Acceptance:**

- read for authenticated users;
- update requires `settings.manage`;
- meta endpoint reflects workshop name.

---

## SET-003 — Workshop logo

**Goal:** Upload and associate workshop logo.

**Depends on:** STOR-002, SET-002.

**Acceptance:**

- image validation;
- file record persisted;
- previous logo remains valid object until cleanup policy exists;
- URL/download authorized.

---

# Gate 3 — Desktop Foundation

## DESK-001 — Server connection configuration

**Goal:** Desktop stores server base URL locally.

**Depends on:** BOOT-003.

**Acceptance:**

- editable before login;
- validation;
- connection test;
- no DB credentials.

---

## DESK-002 — API client foundation

**Goal:** Create typed Go client for API v1 in the desktop native layer.

**Depends on:** DESK-001, API-002.

**Acceptance:**

- base URL;
- common error mapping;
- version compatibility check;
- request timeout;
- React does not call authenticated server endpoints directly;
- no business endpoints yet.

---

## DESK-003 — Login UI

**Goal:** Authenticate through Wails binding → Go API client.

**Depends on:** AUTH-006, DESK-002.

**Acceptance:**

- login form;
- password is passed only to the local Go login operation;
- error state;
- successful transition to app shell;
- React does not persist session token.

---

## DESK-004 — Windows secure session storage

**Goal:** Store session token using Windows secure credential storage.

**Depends on:** DESK-003.

**Acceptance:**

- token not stored in plaintext config;
- logout removes token;
- app can restore session.

---

## DESK-005 — Current user and permission context

**Goal:** Make permissions available to UI.

**Depends on:** DESK-004, RBAC-002.

**Acceptance:**

- route/action visibility can use permissions;
- backend remains authoritative;
- unauthorized server response handled.

---

## DESK-006 — Theme light/dark/system

**Goal:** Implement the only supported theme modes.

**Depends on:** BOOT-003.

**Acceptance:**

- light;
- dark;
- system;
- preference persisted locally or from user setting decision;
- no custom theme controls.

---

## DESK-007 — Workshop branding in shell

**Goal:** Display workshop name/logo.

**Depends on:** SET-003, DESK-002.

**Acceptance:**

- header/login use branding;
- fallback when no logo;
- dynamic window title when feasible.

---

# Gate 4 — Files & Catalog

## FILE-001 — Files schema

**Goal:** Add metadata table for immutable files.

**Depends on:** DB-002, STOR-002.

**Acceptance:**

- UUID;
- SHA-256 unique/indexed appropriately;
- original filename;
- content type;
- size;
- storage key;
- uploader.

---

## FILE-002 — Authenticated upload endpoint

**Goal:** Upload file to storage.

**Depends on:** FILE-001, RBAC-002.

**Acceptance:**

- `files.upload`;
- configurable max size;
- SHA-256;
- dedup behavior documented;
- bad uploads cleaned up.

---

## FILE-003 — Authenticated download endpoint

**Goal:** Download file by ID.

**Depends on:** FILE-001, RBAC-002.

**Acceptance:**

- permission checked;
- safe content disposition;
- streaming;
- 404/403 behavior tested.

---

## CAT-001 — Catalog item schema/repository

**Goal:** Persist generic catalog items.

**Depends on:** DB-002.

**Acceptance:**

- purpose enum/check;
- sellable;
- SKU optional;
- tags representation decided;
- repository tests.

---

## CAT-002 — Catalog item CRUD API

**Goal:** CRUD catalog items.

**Depends on:** CAT-001, RBAC-002.

**Acceptance:**

- read/write permissions;
- validation;
- pagination/list filters.

---

## CAT-003 — Catalog desktop list/detail

**Goal:** Basic catalog UI.

**Depends on:** CAT-002, DESK-005.

**Acceptance:**

- list;
- create/edit when allowed;
- purpose/status visible.

---

## CAT-004 — Catalog parts schema/API

**Goal:** Allow one item to have multiple printed parts.

**Depends on:** CAT-001.

**Acceptance:**

- quantity;
- ordering if needed;
- CRUD;
- cascade behavior tested.

---

## CAT-005 — Design versions schema

**Goal:** Add immutable versions per catalog part.

**Depends on:** CAT-004.

**Acceptance:**

- unique `(part, version)`;
- created_by;
- notes;
- cannot overwrite version number.

---

## CAT-006 — Design version API

**Goal:** Create/list design versions.

**Depends on:** CAT-005, RBAC-002.

**Acceptance:**

- new version operation;
- version history;
- write permission.

---

## CAT-007 — Attach files to design versions

**Goal:** Link file objects with roles.

**Depends on:** CAT-006, FILE-003.

**Acceptance:**

- roles source/mesh/print/preview/documentation/other;
- same file can be reused safely;
- print file discoverable.

---

## CAT-008 — Design provenance/license fields

**Goal:** Track origin and commercial rights.

**Depends on:** CAT-005.

**Acceptance:**

- origin;
- URL;
- author;
- license;
- commercial allowed tri-state if needed;
- attribution fields.

---

## CAT-009 — License warning UI

**Goal:** Warn on sellable items lacking commercial permission.

**Depends on:** CAT-008, CAT-003.

**Acceptance:**

- warning only;
- does not block internal/prototype usage;
- clearly distinguishes unknown vs denied.

---

# Gate 5 — Inventory

## INV-001 — Materials schema/API

**Goal:** Material catalog.

**Depends on:** DB-002, RBAC-002.

**Acceptance:**

- manufacturer/type/color;
- replacement cost/kg;
- decimal density;
- CRUD.

---

## INV-002 — Spools schema/API

**Goal:** Physical spool records.

**Depends on:** INV-001.

**Acceptance:**

- unique human code;
- purchase cost;
- nominal/tare/opening weights;
- status/storage;
- CRUD.

---

## INV-003 — Spool measurements

**Goal:** Append-only weighing history.

**Depends on:** INV-002.

**Acceptance:**

- measurement endpoint;
- remaining weight derivation;
- current cache update in transaction;
- history UI/API.

---

## INV-004 — Supplies schema/API

**Goal:** Non-filament inventory.

**Depends on:** DB-002.

**Acceptance:**

- unit;
- quantity;
- replacement unit cost;
- minimum quantity.

---

## INV-005 — Supply movements

**Goal:** Auditable stock movement.

**Depends on:** INV-004.

**Acceptance:**

- purchase/consume/adjustment/return/discard;
- atomic current quantity update;
- movement history;
- negative stock policy explicit.

---

## BOM-001 — Catalog BOM

**Goal:** Link supplies to sellable catalog items.

**Depends on:** CAT-001, INV-004.

**Acceptance:**

- quantity per unit;
- waste percent;
- CRUD;
- cost preview.

---

## INV-006 — Low inventory queries/UI

**Goal:** Surface low spools/supplies.

**Depends on:** INV-003, INV-005.

**Acceptance:**

- configurable/defined threshold;
- derived query;
- no separate alert persistence.

---

# Gate 6 — Jobs

## PRN-001 — Printers schema/API

**Goal:** Persist logical non-sensitive printer information.

**Depends on:** DB-002.

**Acceptance:**

- model/nozzle/location;
- acquisition/residual/useful life;
- maintenance reserve;
- no access code fields.

---

## JOB-001 — Print jobs schema/repository

**Goal:** Persist job lifecycle fields.

**Depends on:** CAT-005, PRN-001.

**Acceptance:**

- statuses;
- purpose includes `internal`;
- quantities;
- design version;
- printer;
- no customer/quote/order required;
- order item nullable and added/used only when commercial link exists;
- tests cover non-commercial Job creation.

---

## JOB-002 — Print job CRUD/state API

**Goal:** Create and move commercial or non-commercial Jobs through manual lifecycle.

**Depends on:** JOB-001, RBAC-002.

**Acceptance:**

- valid transition rules;
- invalid transition rejected;
- create/update permissions;
- Job can be created without commercial context;
- purpose does not imply revenue.

---

## JOB-003 — Job material usage schema

**Goal:** Add N material usages per Job.

**Depends on:** JOB-001, INV-002.

**Acceptance:**

- roles;
- planned/actual grams;
- source;
- spool;
- historical/replacement cost snapshots fields.

---

## JOB-004 — Job material usage API/UI

**Goal:** Select bobina and record planned/actual usage.

**Depends on:** JOB-003.

**Acceptance:**

- same spool allowed for different roles;
- actual source visible;
- totals shown.

---

## JOB-005 — Job quality review

**Goal:** Separate machine/job completion from quality.

**Depends on:** JOB-002.

**Acceptance:**

- awaiting review;
- approved/partial/failed;
- notes;
- good/scrap quantity.

---

## JOB-006 — Job events

**Goal:** Persist significant lifecycle events.

**Depends on:** JOB-002, AUTH-004.

**Acceptance:**

- append-only;
- actor/source device;
- no high-frequency telemetry.

---

## ENERGY-001 — Energy measurements schema/API

**Goal:** Record manual/estimated energy.

**Depends on:** JOB-001.

**Acceptance:**

- kWh decimal;
- meter start/end;
- estimated power;
- tariff snapshot;
- source.

---

## LABOR-001 — Internal labor cost rates

**Goal:** Configure what human time costs internally.

**Depends on:** DB-002.

**Acceptance:**

- named rate;
- activity type;
- `cost_hourly_rate_cents`;
- active state;
- permission `costing.manage`;
- no billable/customer price mixed into this table.

---

## LABOR-002 — Job labor entries

**Goal:** Record human time per Job using internal cost rates.

**Depends on:** JOB-001, LABOR-001.

**Acceptance:**

- minutes;
- activity;
- internal rate snapshot or derivation strategy;
- totals;
- non-commercial Jobs supported.

---

## MAINT-001 — Maintenance events

**Goal:** Basic printer maintenance history.

**Depends on:** PRN-001.

**Acceptance:**

- types;
- cost optional;
- printer hours optional;
- history.

---

# Gate 7 — Cost Engine

## COST-001 — Pure money/percentage primitives

**Goal:** Create tested financial calculation primitives.

**Depends on:** none besides language foundation.

**Acceptance:**

- no persisted float money;
- percentage representation chosen;
- deterministic rounding;
- tests.

---

## LABOR-003 — Labor cost-rate assistant

**Goal:** Help calculate a realistic internal hourly cost from monthly assumptions.

**Depends on:** LABOR-001, COST-001.

**Acceptance:**

- target monthly compensation;
- monthly labor overhead;
- available hours;
- productive utilization;
- suggested productive hours;
- suggested internal hourly cost;
- denominator/zero validation;
- user can save/override a labor rate;
- pure formula tests.

---

## COST-002 — Machine hour calculation

**Goal:** Calculate depreciation + maintenance reserve.

**Depends on:** COST-001, PRN-001.

**Acceptance:**

- residual value;
- useful life validation;
- zero/invalid handling;
- tests.

---

## COST-003 — Material cost calculator

**Goal:** Calculate planned/actual material costs.

**Depends on:** COST-001, JOB-003.

**Acceptance:**

- historical and replacement outputs;
- multi-usage aggregation;
- tests.

---

## COST-004 — Energy cost calculator

**Goal:** Prefer measured over estimated energy.

**Depends on:** COST-001, ENERGY-001.

**Acceptance:**

- precedence rules;
- explicit utilization factor if used;
- tests.

---

## COST-005 — Labor cost calculator

**Goal:** Calculate human work cost.

**Depends on:** COST-001, LABOR-002.

**Acceptance:**

- minutes × rate;
- no machine time counted automatically;
- tests.

---

## COST-006 — BOM/supply cost calculator

**Goal:** Calculate non-filament production components.

**Depends on:** COST-001, BOM-001.

**Acceptance:**

- quantity;
- waste percent;
- batch quantity;
- tests.

---

## COST-007 — Failure risk calculator

**Goal:** Implement expected-cost failure model.

**Depends on:** COST-001.

**Acceptance:**

- vulnerable/non-vulnerable distinction;
- formula from PRD;
- p=0;
- p near 1 rejected/handled;
- f=0/1;
- tests.

---

## COST-008 — Planned Job cost service

**Goal:** Aggregate planned cost components.

**Depends on:** COST-002..007.

**Acceptance:**

- component breakdown;
- historical/replacement context explicit;
- no snapshot yet;
- API endpoint.

---

## COST-009 — Actual Job cost service

**Goal:** Aggregate real Job cost.

**Depends on:** COST-003..006, JOB-005.

**Acceptance:**

- uses good quantity;
- zero good units handled;
- component breakdown.

---

## COST-010 — Cost snapshot schema

**Goal:** Persist immutable cost breakdown.

**Depends on:** DB-002.

**Acceptance:**

- component cents;
- parameters JSON;
- calculated_at;
- immutable repository contract.

---

## COST-011 — Close Job financial snapshot

**Goal:** Generate Cost Snapshot from completed Job.

**Depends on:** COST-009, COST-010.

**Acceptance:**

- explicit action;
- previous snapshot not overwritten;
- audit user;
- UI/API shows snapshot.

---

## COST-012 — Planned vs actual comparison

**Goal:** Show variance.

**Depends on:** COST-008, COST-011.

**Acceptance:**

- absolute and percentage variance;
- material/time/energy totals where available.

---

## COST-013 — Non-commercial operational consumption classification

**Goal:** Prevent internal/test/prototype/personal Job costs from being reported as commercial losses.

**Depends on:** COST-011, JOB-001.

**Acceptance:**

- Job cost remains available;
- no sale/revenue inferred;
- cost grouped by Job purpose;
- commercial margin is not calculated for non-commercial Jobs;
- query/service tests.

---

# Gate 8 — Pricing & Commercial

## PRICE-001 — Manual sales channel profiles

**Goal:** Persist manual channel/marketplace fee assumptions.

**Depends on:** COST-001.

**Acceptance:**

- channel type;
- percent fee;
- payment fee;
- tax;
- sales commission;
- fixed per-order fee;
- fixed per-item fee;
- shipping subsidy;
- effective dates;
- source/notes;
- active;
- no marketplace API.

---

## PRICE-002 — Markup calculator

**Goal:** Implement explicit markup calculation.

**Depends on:** COST-001.

**Acceptance:**

- named markup;
- tests;
- not mislabeled as margin.

---

## PRICE-003 — Margin calculator

**Goal:** Calculate achieved margin on sale price.

**Depends on:** PRICE-001.

**Acceptance:**

- variable fees on sale price;
- fixed fees;
- tests.

---

## PRICE-004 — Target margin price solver

**Goal:** Solve minimum price for target margin.

**Depends on:** PRICE-001.

**Acceptance:**

- formula from PRD;
- invalid denominator error;
- fixed and variable fees;
- tests.

---

## PRICE-005 — Break-even and minimum price

**Goal:** Calculate no-loss and configured minimum-margin prices.

**Depends on:** PRICE-004.

**Acceptance:**

- break-even;
- minimum margin;
- clear breakdown.

---

## PRICE-006 — Batch pricing

**Goal:** Calculate cost/price at different quantities.

**Depends on:** COST-008, PRICE-005.

**Acceptance:**

- per-batch vs per-unit labor supported by available model;
- unit economics;
- total contribution;
- tests.

---

## PRICE-007 — Product price versions

**Goal:** Persist effective official price history.

**Depends on:** CAT-001, PRICE-001.

**Acceptance:**

- effective dates;
- channel profile;
- list/minimum price;
- history;
- saving a version is explicit.

---

## PRICE-008 — Market reference prices

**Goal:** Manual competitor/reference prices.

**Depends on:** CAT-001.

**Acceptance:**

- source text/URL;
- observed date;
- price;
- notes.

---

## PRICE-009 — Catalog pricing calculation API/service

**Goal:** Calculate pricing for a Catalog Item without customer, quote, order or sale.

**Depends on:** COST-008, PRICE-005, CAT-001.

**Acceptance:**

- item + quantity + cost basis + channel profile;
- manufacturing cost;
- channel fee breakdown;
- break-even;
- minimum price;
- target-margin price;
- contribution;
- achieved margin/markup;
- calculation itself does not persist a sale/quote/order.

---

## PRICE-010 — Multi-channel comparison

**Goal:** Compare the same Catalog Item across multiple manual channel profiles.

**Depends on:** PRICE-009.

**Acceptance:**

- at least two profiles per request/view;
- side-by-side required prices;
- contribution comparison;
- highlights invalid/impossible configurations;
- marketplace profiles are labeled manual assumptions.

---

## PRICE-011 — Catalog pricing workspace UI

**Goal:** Add a Precificação area directly to Catalog Item.

**Depends on:** PRICE-010, PRICE-007.

**Acceptance:**

- usable without customer;
- choose quantity/cost basis/margin;
- select one or more channels;
- show breakdown;
- explicit “save as price version”;
- no quote/order created by calculation.

---

## LABOR-004 — Billable labor pricing profiles

**Goal:** Configure what the workshop charges for human services separately from internal cost.

**Depends on:** LABOR-001, PRICE-004.

**Acceptance:**

- references internal labor rate;
- billing hourly rate;
- minimum billable minutes;
- rounding increment;
- target margin;
- active;
- supports modeling/customization/finishing/other.

---

## LABOR-005 — Labor/service pricing assistant

**Goal:** Recommend billable labor price from internal cost, target margin and selected channel.

**Depends on:** LABOR-003, LABOR-004, PRICE-004.

**Acceptance:**

- shows internal hourly cost;
- break-even hourly price;
- target-margin hourly price;
- channel-adjusted hourly price;
- minimum charge calculation;
- rounding behavior;
- formula tests.

---

## COMM-001 — Customers schema/API

**Goal:** Basic customer records.

**Depends on:** DB-002, RBAC-002.

**Acceptance:**

- person/company;
- name;
- optional contact data;
- notes.

---

## QUOTE-001 — Independent quotes schema/API

**Goal:** Quote lifecycle that does not imply order or sale.

**Depends on:** COMM-001.

**Acceptance:**

- statuses;
- validity;
- code;
- customer;
- accepted quote may remain without order;
- no revenue/order side effect.

---

## QUOTE-002 — Product/service/custom quote items

**Goal:** Add products or billable services to a quote.

**Depends on:** QUOTE-001, PRICE-005, LABOR-004.

**Acceptance:**

- item type product/service/custom;
- catalog item nullable;
- labor pricing profile nullable;
- quantity/billable minutes;
- channel profile;
- cost snapshot;
- pricing snapshot;
- discount;
- totals.

---

## QUOTE-003 — Quote pricing assistant UI

**Goal:** Reuse pricing outputs to prepare a quote without forcing conversion.

**Depends on:** QUOTE-002, PRICE-011, LABOR-005.

**Acceptance:**

- manufacturing/service cost;
- break-even;
- target price;
- minimum;
- markup;
- margin;
- operator can choose final unit/service price;
- quote remains independent.

---

## QUOTE-004 — Quote export

**Goal:** Export branded quote document/PDF.

**Depends on:** QUOTE-003, SET-003.

**Acceptance:**

- workshop branding;
- customer;
- product/service items;
- totals;
- validity;
- notes/conditions.

---

## QUOTE-005 — Quote lifecycle without implicit conversion

**Goal:** Enforce sent/accepted/rejected/expired/cancelled behavior without creating orders.

**Depends on:** QUOTE-001.

**Acceptance:**

- accepting does not create order;
- rejecting/expiring/cancelling has no revenue effect;
- transitions tested;
- audit event or equivalent.

---

## ORDER-001 — Orders schema/API

**Goal:** Manual order lifecycle.

**Depends on:** COMM-001.

**Acceptance:**

- statuses;
- customer;
- quote optional;
- codes;
- no payment/received-revenue inference.

---

## ORDER-002 — Order items

**Goal:** Persist product/service items and price snapshot.

**Depends on:** ORDER-001.

**Acceptance:**

- item type;
- quantity;
- unit price;
- totals;
- snapshot;
- service item may require no Job.

---

## ORDER-003 — Explicit quote-to-order conversion

**Goal:** Create order only from an explicit user action without re-pricing the quote.

**Depends on:** QUOTE-005, ORDER-002.

**Acceptance:**

- explicit endpoint/action;
- idempotency strategy;
- values copied from quote snapshot;
- accepted quote alone does not convert;
- duplicate conversion prevented or handled explicitly.

---

## ORDER-004 — Link Jobs to order items

**Goal:** Production traceability for commercial order items.

**Depends on:** ORDER-002, JOB-001.

**Acceptance:**

- multiple Jobs per order item;
- Job link optional;
- progress derivable;
- service-only items require no Job;
- no separate Task entity.

---

## ORDER-005 — Order profitability view

**Goal:** Show order value vs actual production costs without mixing unrelated Jobs.

**Depends on:** ORDER-004, COST-011.

**Acceptance:**

- order value;
- completed linked Job costs;
- service values;
- contribution;
- incomplete-data warning;
- non-commercial Jobs excluded.

---

# Gate 9 — Bambu Studio

## BSTUDIO-001 — Detect/configure Bambu Studio executable

**Goal:** Resolve local Bambu Studio path.

**Depends on:** DESK-001.

**Acceptance:**

- common Windows locations checked;
- manual override;
- validation.

---

## CACHE-001 — Desktop file cache

**Goal:** Cache immutable server files locally by hash.

**Depends on:** FILE-003.

**Acceptance:**

- hash-based identity;
- validation after download;
- safe temp/final write;
- cache clear action.

---

## BSTUDIO-002 — Open design print file locally

**Goal:** Download 3MF and launch Bambu Studio.

**Depends on:** BSTUDIO-001, CACHE-001, CAT-007.

**Acceptance:**

- correct design version;
- print-role file;
- hash validated;
- clear error if Studio unavailable.

---

## JOB-007 — Prepare Print workflow

**Goal:** Combine Job preparation with Bambu Studio launch.

**Depends on:** JOB-004, BSTUDIO-002.

**Acceptance:**

- create/select Job;
- select printer;
- bobina/material;
- planned usage;
- mark prepared;
- open correct 3MF;
- does not send print command.

---

# Gate 10 — Backup & Hardening

## BKP-001 — PostgreSQL backup command/service

**Goal:** Create `pg_dump` custom backup.

**Depends on:** DEP-002.

**Acceptance:**

- timestamped;
- errors surfaced;
- no password in command logs.

---

## BKP-002 — File storage backup

**Goal:** Back up object storage.

**Depends on:** STOR-002.

**Acceptance:**

- reproducible archive/snapshot;
- checksums.

---

## BKP-003 — Backup manifest

**Goal:** Describe DB + storage backup set.

**Depends on:** BKP-001, BKP-002.

**Acceptance:**

- server version;
- counts;
- checksums;
- timestamps.

---

## BKP-004 — Restore command

**Goal:** Restore DB and storage with app stopped.

**Depends on:** BKP-003.

**Acceptance:**

- refuses unsafe active restore where detectable;
- restores clean environment;
- documented.

---

## BKP-005 — Automated restore verification

**Goal:** Prove backups work.

**Depends on:** BKP-004.

**Acceptance:**

- create backup fixture;
- restore empty environment;
- check DB records;
- sample file hashes;
- reference integrity.

---

## SEC-001 — Upload security hardening

**Goal:** Validate upload boundaries.

**Depends on:** FILE-002.

**Acceptance:**

- max size;
- filename sanitization for display;
- MIME/extension handling documented;
- path traversal tests.

---

## SEC-002 — Login/session hardening

**Goal:** Final auth security review.

**Depends on:** AUTH-008.

**Acceptance:**

- rate limiting;
- token entropy;
- revocation;
- logging redaction tests.

---

## OBS-003 — Structured server logs

**Goal:** Standardize structured logging.

**Depends on:** API-001, AUTH-007.

**Acceptance:**

- request ID;
- user when available;
- no secrets.

---

## DESK-008 — Desktop logs and diagnostics

**Goal:** Provide local troubleshooting without exposing secrets.

**Depends on:** DESK-002.

**Acceptance:**

- local log file;
- open logs folder action;
- server connectivity diagnostic;
- token/access code redaction.

---

# Gate 11 — Bambu Telemetry

## BAMBUT-001 — Local printer binding model

**Goal:** Map central printer ID to local Bambu connection config.

**Depends on:** PRN-001, DESK-001.

**Acceptance:**

- binding stored locally;
- server never receives access code;
- access code in Windows secure storage.

---

## BAMBUT-002 — Telemetry adapter contract

**Goal:** Define read-only printer telemetry interface.

**Depends on:** none.

**Acceptance:**

- connect/disconnect;
- current state;
- no control methods.

---

## BAMBUT-003 — Bambu LAN telemetry spike

**Goal:** Validate A1 Mini real-device telemetry behavior.

**Depends on:** BAMBUT-001, BAMBUT-002.

**Acceptance:**

- document firmware/version;
- online/offline;
- print state;
- progress/file/layers/remaining when available;
- no control commands;
- fixtures captured for tests where safe.

---

## BAMBUT-004 — Bambu telemetry implementation

**Goal:** Implement read-only adapter from validated spike.

**Depends on:** BAMBUT-003.

**Acceptance:**

- reconnect behavior;
- sanitized state;
- tests using fixtures/mocks.

---

## TEL-001 — Server current printer state API/schema

**Goal:** Store sanitized current state.

**Depends on:** PRN-001.

**Acceptance:**

- source device;
- updated timestamp;
- no Bambu secret fields;
- upsert semantics.

---

## TEL-002 — Publish telemetry from desktop

**Goal:** Desktop periodically updates server.

**Depends on:** BAMBUT-004, TEL-001.

**Acceptance:**

- permission `telemetry.publish`;
- reasonable interval;
- network failure does not affect printer.

---

## TEL-003 — Shared telemetry UI

**Goal:** Other users see recent printer status.

**Depends on:** TEL-002.

**Acceptance:**

- polling 5–10 seconds;
- stale indicator;
- `telemetry.read`;
- no control buttons.

---

## TEL-004 — Significant printer events

**Goal:** Persist meaningful telemetry transitions only.

**Depends on:** TEL-001, JOB-006.

**Acceptance:**

- online/offline;
- print detected/finished/error where available;
- no per-second history.

---

# Gate 12 — Production Validation

## VAL-001 — Seed initial real workshop data

**Goal:** Configure actual printer/materials/bobinas/supplies.

**Depends on:** Gates 0–10.

**Acceptance:**

- no fake production data mixed with real data;
- initial parameters documented.

---

## VAL-002 — Execute first 5 tracked Jobs

**Goal:** Validate basic workflow.

**Depends on:** VAL-001.

**Acceptance:**

- planned and actual material;
- result review;
- cost snapshots;
- issues logged.

---

## VAL-003 — Validate failed Job

**Goal:** Ensure failure economics/stock handling works.

**Depends on:** VAL-002.

**Acceptance:**

- failed/partial case;
- waste recorded;
- cost remains auditable.

---

## VAL-004 — Validate batch Job

**Goal:** Test good/scrap quantity economics.

**Depends on:** VAL-002.

**Acceptance:**

- quantity > 1;
- per-good-unit cost;
- setup/per-unit effects visible.

---

## VAL-005 — Validate quote → order → Jobs

**Goal:** Prove commercial traceability.

**Depends on:** Gate 8, VAL-002.

**Acceptance:**

- customer;
- quote;
- explicitly converted order;
- Jobs;
- actual profitability.

---

## VAL-005A — Validate non-commercial Job economics

**Goal:** Prove internal/personal/test usage consumes resources without becoming commercial loss.

**Depends on:** COST-013, VAL-002.

**Acceptance:**

- one non-commercial Job;
- material/energy/machine/labor consumed;
- no customer/order/revenue;
- operational consumption report;
- no commercial margin calculated.

---

## VAL-005B — Validate catalog pricing across marketplaces/channels

**Goal:** Prove pre-sale pricing comparison works without marketplace integration.

**Depends on:** PRICE-011.

**Acceptance:**

- same item;
- direct/Pix plus at least two channel profiles;
- manual marketplace fees;
- required price/contribution compared;
- no quote/order created.

---

## VAL-005C — Validate quote without order

**Goal:** Prove quote lifecycle is independent.

**Depends on:** QUOTE-005.

**Acceptance:**

- quote created and exported;
- one rejected/expired or accepted-without-order case;
- no order/revenue side effect.

---

## VAL-005D — Validate labor/service pricing

**Goal:** Prove internal labor cost and billable labor price are separate.

**Depends on:** LABOR-005, QUOTE-002.

**Acceptance:**

- internal hourly cost calculated;
- billable hourly price calculated;
- modeling/customization service line priced;
- snapshot in quote;
- values auditable.

---

## VAL-006 — Complete 20-Job validation set

**Goal:** Meet PRD production validation.

**Depends on:** VAL-002..005D, Gate 11.

**Acceptance:**

- required scenario diversity;
- planned vs actual report;
- defects/tasks created for material issues.

---

## VAL-007 — Release 1 acceptance checklist

**Goal:** Execute every item in PRD section 34.

**Depends on:** all release tasks.

**Acceptance:**

- evidence for each item;
- known limitations documented;
- release tag only after pass.

---

# Explicit post-Release-1 backlog

Do not implement these during Release 1 tasks unless separately approved:

```text
WEB-001 web client
MARKET-ML Mercado Livre API integration / automatic fee/order sync
MARKET-SHP Shopee API integration / automatic fee/order sync
NUV-001 Nuvemshop
ENERGY-HA Home Assistant energy adapter
ENERGY-TUYA smart plug adapter
ENV-001 environment readings
BAMBU-CONTROL printer control
BAMBU-CLOUD cloud integration
PRICE-AUTO historical failure-rate pricing
PRICE-AI price recommendation
ANALYTICS advanced dashboards
FISCAL Brazilian fiscal documents
PAYMENTS payment integrations
MOBILE mobile app
```
