# Roles and permission authorization

The server authorizes concrete permissions, never role-name checks in product
handlers. Release 1 uses five fixed role profiles. Migration
`00006_user_roles.sql` adds the selected profile to each user, upgrades the
durable bootstrap identity to `owner`, and assigns least-privilege `viewer` to
other users that predate the migration.

| Capability | Owner | Operator | Designer | Commercial | Viewer |
|---|:---:|:---:|:---:|:---:|:---:|
| Catalog read | yes | yes | yes | yes | yes |
| Catalog write | yes | no | yes | no | no |
| File read | yes | yes | yes | no | yes |
| File upload | yes | no | yes | no | no |
| Inventory read/write | yes | yes | no | no | read only |
| Jobs | yes | operate/evaluate | no | no | read only |
| Costing | yes | no | no | read only | read only |
| Pricing | yes | no | no | manage | read only |
| Commercial | yes | no | no | read/write | read only |
| Telemetry | yes | read/publish | no | no | read only |
| Users/settings/backup management | yes | no | no | no | no |

The canonical permission identifiers live in
`internal/domain/auth/authorization.go`. Unknown roles and unknown permissions
are denied by default. `RoleHasPermission` is available to application
services; HTTP handlers use `RequirePermission`, which composes bearer
authentication before the permission check. A request without a valid session
receives `401 unauthenticated`; an authenticated user lacking the capability
receives `403 forbidden`.

The authenticated HTTP context contains safe identity, role, and session
metadata only. Password hashes and bearer-token hashes never enter it. Future
editable-role work can replace the fixed mapping while product handlers remain
permission-based.
