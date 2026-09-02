# Bearer authentication

Protected API handlers use `AuthenticationMiddleware` with the opaque token in
one `Authorization: Bearer <token>` header. Missing, malformed, unknown,
expired, and revoked tokens all return the same HTTP `401` response. Sessions
for disabled users are rejected the same way. Authentication errors use
`WWW-Authenticate: Bearer` and `Cache-Control: no-store` without including the
token or the reason a session is inactive.

The application layer accepts only the canonical 43-character unpadded
base64url form produced by session issuance, hashes it with SHA-256, and sends
only the hash to PostgreSQL. The repository resolves the session and user in a
parameterized query. The HTTP context receives safe `CurrentUser` and
`CurrentSession` values; password hashes and token hashes are deliberately
excluded. `CurrentUser` includes the persisted fixed role so authorization can
resolve concrete permissions without querying on every handler invocation.

`last_used_at` is refreshed at most once every five minutes. Requests inside
that interval perform no update. When concurrent requests observe a stale
value, a conditional PostgreSQL update permits only one of them to write. This
keeps the audit timestamp useful without writing on every authenticated
request. Authorization remains a separate layer and can be composed through
`RequirePermission`, as documented in [Roles and permission
authorization](authorization.md).

Session listing and revocation are bearer-protected. Ownership permits users
to manage their own sessions; management of another user's sessions requires
the concrete `users.manage` permission. See [Desktop sessions](sessions.md) for
the endpoint contract and response safety boundary.
