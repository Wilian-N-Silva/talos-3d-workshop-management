# Desktop sessions

Desktop bearer sessions are stored in PostgreSQL by migration
`00005_sessions.sql`. Each session relates one user to one registered client
device and records creation, expiry, optional last-use, and optional revocation
timestamps. User and device deletion is restricted while related audit records
exist.

`SessionService` generates 32 cryptographically random bytes and encodes them
as an unpadded base64url bearer token. This provides 256 bits of entropy. The
plaintext token is returned only after its session record is created and must
be placed in Windows secure credential storage by the desktop client.

The server derives a SHA-256 digest from the encoded token. PostgreSQL stores
only that fixed 32-byte digest and enforces uniqueness; the plaintext token is
never passed to the repository or persisted. Callers provide an explicit
future expiry because Release 1 has not yet defined a global session-lifetime
policy. Authentication and last-used throttling remain AUTH-006 and AUTH-007
scope, while session listing and revocation behavior remain AUTH-008 scope.
