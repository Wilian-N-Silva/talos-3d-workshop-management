# Workshop settings

Migration `00007_workshop_settings.sql` defines a singleton workshop settings
record. A constrained `singleton_id` permits at most one row, and server startup
initializes that row when it is absent. The initial workshop name, locale,
currency, and display timezone come from the validated process configuration;
subsequent restarts never overwrite persisted changes. Theme initially defaults
to `system`.

The database constrains bounded non-empty names, locale and currency formats,
and the fixed `light`, `dark`, and `system` theme values. The application layer
also validates IANA timezones before persistence. Timestamps remain UTC; the
display timezone is presentation policy only.

Authenticated users read settings through `GET /api/v1/settings`. Replacing the
mutable values with `PUT /api/v1/settings` requires the concrete
`settings.manage` permission. The public `GET /api/v1/meta` response reads the
persisted workshop name on every request, so branding discovery reflects an
authorized update without restarting the server.

`logo_file_id` is nullable and read-only in this package. SET-003 will associate
it only after immutable file metadata, image validation, object storage, and an
authorized download route are available.
