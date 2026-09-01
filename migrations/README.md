# Database migrations

Numbered, one-way PostgreSQL migrations belong in this directory and are
embedded into the server binary.

## Convention

- tool: Goose v3;
- filenames: five-digit sequence plus description, for example
  `00002_users.sql`;
- migrations contain an `Up` section only;
- a migration merged into `main` is immutable;
- later corrections use a new migration number.

## Startup and locking

The server acquires a PostgreSQL session advisory lock before running pending
migrations. The lock is held on a dedicated pool connection, so the configured
maximum open connection count must be at least two. Other server instances wait
for the same lock and then re-check migration state.

Migration execution happens after the database ping and before the HTTP
listener starts. A migration or lock failure prevents the server from becoming
available. The readiness endpoint introduced by `OBS-002` will also inspect the
exposed migration state.
