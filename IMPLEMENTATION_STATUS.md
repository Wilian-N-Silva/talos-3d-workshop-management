# Implementation Status

> Derived progress ledger for the current repository.
>
> This file is **not** a requirements source. `PRD.md` and `IMPLEMENTATION_TASKS.md` define what must exist. This file records what has been verified to exist.

---

## Reconciliation metadata

```yaml
last_reconciled_commit: 01dc256
last_reconciled_at_utc: 2026-09-04T02:16:59Z
reconciled_by: Codex
```

The first agent adopting this workflow in an already-started repository must populate these fields after inspecting the repository.

Do not assume a remembered task such as `AUTH-007` is the real frontier. Determine the frontier from repository evidence.

---

## Status vocabulary

Use only when useful:

- `verified_complete` — acceptance criteria evidenced in repository state;
- `partial` — meaningful implementation exists but one or more acceptance criteria are missing;
- `blocked` — work cannot proceed due to an explicit blocker;
- `superseded` — task no longer applies because an approved specification/ADR replaced it.

Tasks absent from this ledger are **unverified here**. Absence does not prove whether they are started or not.

---

## Verified / exceptional task states

| Task | Status | Evidence | Verified at | Notes |
|---|---|---|---|---|
| BOOT-001 | verified_complete | root specs + `cmd/server` + `desktop` + `migrations` + development README | 55e19a9 | Repository structure and entrypoints exist. |
| BOOT-002 | verified_complete | `cmd/server/main.go` + graceful-shutdown tests | 55e19a9 | Server builds and handles process shutdown. |
| BOOT-003 | verified_complete | `desktop/` Wails v2 shell + strict React/TypeScript frontend | 55e19a9 | Windows desktop build is part of CI. |
| BOOT-004 | verified_complete | README validation commands + frontend package scripts | 55e19a9 | Backend and desktop checks are documented. |
| BOOT-005 | verified_complete | `.github/workflows/ci.yml` | 55e19a9 | PR/main workflow covers backend and desktop checks without deployment. |
| CFG-001 | verified_complete | `internal/config` tests + `.env.example` | 55e19a9 | Required server settings are centralized and validated. |
| DB-001 | verified_complete | `internal/infrastructure/postgres/database.go` + tests | 55e19a9 | `database/sql` with pgx, configured pool, no ORM. |
| DB-002 | verified_complete | migration runner/tests + numbered embedded migrations | 55e19a9 | Startup migration locking and state checks are implemented. |
| DEP-001 | verified_complete | `docker-compose.yml` PostgreSQL service | 55e19a9 | Internal-only DB port, volume, and healthcheck. |
| DEP-002 | verified_complete | multi-stage `Dockerfile` + Compose server service | 55e19a9 | Non-root server image and persistent data volume. |
| OBS-001 | verified_complete | liveness handler/tests | 55e19a9 | Stable DB-independent `/health/live`. |
| OBS-002 | verified_complete | readiness service/handler tests | 55e19a9 | PostgreSQL, migrations, and storage checked; printer excluded. |
| STOR-001 | verified_complete | `internal/application/storage` contract/tests | 55e19a9 | Immutable object boundary uses validated opaque keys. |
| STOR-002 | verified_complete | local filesystem adapter/tests | 55e19a9 | Content-addressed SHA-256 storage with temp-file tests. |
| API-001 | verified_complete | API v1 router/error/request-ID tests | 55e19a9 | JSON errors and `/api/v1` routing are established. |
| API-002 | verified_complete | meta handler/tests + build metadata | 55e19a9 | API/server/minimum desktop versions and workshop name returned. |
| AUTH-001 | verified_complete | migration `00002_users` + user repository PostgreSQL tests | 55e19a9 | User status, unique login, and timestamps persisted. |
| AUTH-002 | verified_complete | Argon2id password service/tests + architecture docs | 55e19a9 | Central parameters and safe malformed-hash handling. |
| AUTH-003 | verified_complete | bootstrap state migration + race-safe service/repository/HTTP tests | 55e19a9 | First-user endpoints close permanently after creation. |
| AUTH-004 | verified_complete | migration `00004_client_devices` + repository tests | 55e19a9 | Installation metadata and last-seen audit state persisted. |
| AUTH-005 | verified_complete | migration `00005_sessions` + token/session tests | 55e19a9 | 256-bit opaque token; only SHA-256 hash is persisted. |
| AUTH-006 | verified_complete | login service/endpoint/rate-limit/integration tests | 55e19a9 | Uniform invalid credentials, last-login update, and session issuance. |
| AUTH-007 | verified_complete | authentication service + hash-only repository lookup/touch + bearer middleware/tests | a77fa45 | Expired/revoked/disabled sessions are rejected; last-used writes are throttled. |
| AUTH-008 | verified_complete | protected session list/revoke routes + ownership/`users.manage` service checks + PostgreSQL revocation/authentication test | e6e265f | Safe device metadata is listable; revocation is idempotent and immediately invalidates the bearer token. |
| RBAC-001 | verified_complete | migration `00006_user_roles` + fixed permission catalog/matrix tests | a77fa45 | Bootstrap identity is Owner; legacy non-owner users become Viewers. |
| RBAC-002 | verified_complete | permission helper + composed HTTP authorization middleware/tests | a77fa45 | Missing authentication returns 401; insufficient permission returns 403. |
| SET-001 | verified_complete | migration `00007_workshop_settings` + validated singleton service/repository + PostgreSQL tests | e93aa76 | Process defaults initialize once; persisted values and fixed theme policy survive restarts. |
| SET-002 | verified_complete | authenticated settings GET + `settings.manage` PUT + dynamic meta handler/tests | e93aa76 | All authenticated roles can read; updates require the concrete permission and immediately affect meta. |
| SET-003 | verified_complete | validated logo service + current-logo routes + association repository/HTTP/PostgreSQL tests | 27b64cb | PNG/JPEG uploads require `settings.manage`; only the current association is public, and previous immutable objects remain valid. |
| FILE-001 | verified_complete | migration `00008_files` + immutable file domain metadata + PostgreSQL repository tests | 27b64cb | UUID, unique SHA-256, safe storage key, original name, content type, size, uploader, and UTC creation time are persisted. |
| DESK-001 | verified_complete | pre-login React connection screen + validated user-scoped server configuration + Wails methods/tests | 769ea77 | Only a credential-free HTTP(S) base URL is stored; users can edit, test, and save it before login. |
| DESK-002 | verified_complete | typed native meta client + error/timeout/compatibility tests + Wails production build | 769ea77 | Native Go owns HTTP and version checks; React has no server HTTP path and no business endpoint exists yet. |
| DESK-003 | verified_complete | React login/error/shell flow + Wails Login binding + typed native login client/tests | 1c9e285 | Password is passed only to native Go; the Wails response never contains the bearer token. |
| DESK-004 | verified_complete | Windows Credential Manager session store + expiry/restore/logout tests + Wails production build | 1c9e285 | Session credentials are server-scoped, absent from plaintext config, restored at startup, and deleted on logout. |

Evidence should be concise, for example:

```text
migration 000007 + internal/auth/session.go + middleware tests
```

Do not paste large diffs or lengthy summaries into this table.

---

## Active Work Package

```yaml
id: WP-DESK-02
title: Desktop Login and Secure Session
tasks: [DESK-003, DESK-004]
branch: work/wp-desk-02-login-session
state: ready_for_pr
pull_request: null
```

Recommended `state` values:

```text
planned
in_progress
ready_for_pr
in_review
```

After a package is merged and a later reconciliation confirms it on `main`, clear this section or replace it with the next package.

---

## Recently completed Work Packages

| Work Package | Tasks | Merge/commit | Notes |
|---|---|---|---|
| WP-DESK-01 | DESK-001, DESK-002 | 01dc256 (PR #27) | Native server connection configuration and version-compatible API client. |
| WP-SET-02 | FILE-001, SET-003 | 415c986 (PR #26) | Immutable file metadata and authorized current workshop logo. |
| WP-SET-01 | SET-001, SET-002 | 8eeb208 (PR #25) | Persisted workshop settings and permission-aware API. |
| WP-AUTH-02 | AUTH-008 | f1a7fd7 (PR #24) | Session/device listing and permission-aware revocation. |
| WP-AUTH-01 | AUTH-007, RBAC-001, RBAC-002 | 6f9628d (PR #23) | Bearer authentication and permission-based access control. |

Keep this section lightweight. Older detail remains available through Git history and does not need to be duplicated forever.

---

## Follow-ups / blockers discovered

| ID | Source | Description | Disposition |
|---|---|---|---|
| FUP-CFG-001 | reconciliation at 55e19a9 | Direct `go run` listener ignores `TALOS_SERVER_BIND_ADDRESS`/`TALOS_TRUSTED_LAN` and binds all interfaces; align listener policy with trusted-LAN security docs. | Future configuration/security Work Package; not expanded into authentication packages. |
| FUP-DESK-001 | WP-DESK-02 at 1c9e285 | Desktop login does not yet persist/reuse the server-issued client device ID after local logout, so a later login registers another audit device. | Address with device/session management UI; no session-security weakening in this package. |

Use this only for concrete follow-up work discovered during implementation/reconciliation. Do not turn it into a second product backlog.

---

## Reconciliation procedure

When resuming development:

1. inspect Git status/current branch/recent history;
2. locate candidate task IDs and implementation areas;
3. verify candidate tasks against their acceptance criteria;
4. verify the contiguous dependency chain around the implementation frontier;
5. update this ledger;
6. select a coherent next Work Package;
7. continue implementation under `GIT_WORKFLOW.md`.

Prefer a bounded audit around the actual implementation frontier. Do not burn context deeply reviewing future modules that clearly do not exist yet.
