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
never passed to the repository or persisted. Login uses the configurable
`TALOS_SESSION_TTL` policy, which defaults to 30 days. [Bearer
authentication](authentication.md) rejects expired/revoked sessions and
throttles last-used writes to one update per five-minute interval.

Authenticated users list their own session/device audit records with
`GET /api/v1/auth/sessions`. An Owner, or any future role carrying the concrete
`users.manage` permission, can add `?user_id=<uuid>` to list another user's
sessions. Responses deliberately exclude token hashes and identify the bearer
session used for the request with `current: true`.

`POST /api/v1/auth/sessions/{session_id}/revoke` revokes a session owned by the
current user. The same `users.manage` permission authorizes revoking another
user's session. Revocation is idempotent and preserves the first revocation
timestamp. A revoked token fails bearer authentication immediately. These
endpoints are device audit/session controls only and create no Server → Desktop
or Server → Printer command path.
