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

`POST /api/v1/settings/logo` requires `settings.manage` and accepts one
multipart `file`. The service reads at most the smaller of the configured
upload limit and 5 MiB, rejects path-like/control-character names, and fully
decodes PNG or JPEG content. Images must be no larger than 4096 × 4096 pixels;
the client-supplied content type is not trusted.

File metadata and the `logo_file_id` association are committed atomically after
the validated bytes enter immutable content-addressed storage. Replacing a logo
does not delete the previous file record or object; cleanup policy does not yet
exist. Identical bytes reuse the SHA-256 file record.

Both settings and metadata responses expose `/api/v1/meta/logo` only when a
logo is associated. That fixed pre-login branding route is public but
association-authorized: it joins through the singleton's current
`logo_file_id` and accepts no caller-controlled file ID or storage key. It
cannot retrieve previous logos or arbitrary files. Responses use the persisted
content type, `nosniff`, an SHA-256 ETag, and revalidation caching.
