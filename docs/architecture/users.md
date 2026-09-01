# User persistence

Server users are stored in PostgreSQL by migration `00002_users.sql`. The
record contains a database-generated UUID, display name, login identifier,
password hash, status, creation/update timestamps, and an optional last-login
timestamp.

Login identifiers are stored without surrounding whitespace and are unique
case-insensitively. The initial statuses are `active` and `disabled`.
Timestamps use PostgreSQL `timestamptz`; the repository normalizes scanned
values to UTC.

The repository accepts only a `password_hash` field; plaintext passwords must
never cross this persistence boundary. `PasswordService` produces PHC-encoded
Argon2id hashes with a unique 16-byte cryptographic salt. The centralized
Release 1 profile is 19 MiB memory, two iterations, one thread, and a 32-byte
derived key. Each stored hash carries its parameters so the default can be
raised without invalidating existing credentials.

Verification accepts only Argon2id version 19 and validates bounded memory,
iteration, parallelism, salt, key, and encoded lengths before deriving a key.
Malformed or resource-unsafe hashes fail closed. Roles and permissions are
deliberately absent from the user record and remain RBAC-001 scope.

## First-owner bootstrap

Migration `00003_bootstrap_state.sql` adds a singleton completion record linked
to the initial owner. `CreateFirst` acquires a PostgreSQL transaction-scoped
advisory lock, rechecks both users and bootstrap state, inserts the user, and
records completion in one transaction. Concurrent requests and separate server
instances therefore cannot create multiple initial owners.

The completion record is retained independently of future account status and
prevents bootstrap from reopening. Databases that already contain users when
the migration is applied are closed automatically and record their oldest user
as the initial-owner marker. Concrete role-to-permission persistence remains
RBAC-001 scope.
