# User persistence

Server users are stored in PostgreSQL by migration `00002_users.sql`. The
record contains a database-generated UUID, display name, login identifier,
password hash, status, creation/update timestamps, and an optional last-login
timestamp.

Login identifiers are stored without surrounding whitespace and are unique
case-insensitively. The initial statuses are `active` and `disabled`.
Timestamps use PostgreSQL `timestamptz`; the repository normalizes scanned
values to UTC.

The repository accepts only a `password_hash` field. Password hashing and hash
validation are introduced by AUTH-002; plaintext passwords must never cross
this persistence boundary. Roles and permissions are deliberately absent from
the user record and remain RBAC-001 scope.
