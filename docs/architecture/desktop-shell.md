# Desktop permission-aware branded shell

The application shell loads only after native Go restores the Credential Manager
session and successfully calls authenticated `GET /api/v1/settings`. HTTP 401
means the bearer session is no longer usable: native Go removes the local
credential and returns the UI to login. HTTP 403 remains an error and does not
cause a client-side authorization bypass.

The secure session record includes the safe user identity, role, and concrete
permission list issued by the server at login. React receives those safe fields,
never the bearer token. `PermissionProvider` and `PermissionGate` let routes and
actions hide unavailable affordances. This is presentation behavior only; server
permission middleware remains authoritative for every protected request.

Workshop branding comes from public metadata through native Go. If a logo exists,
the client accepts only a bounded PNG/JPEG response from the same server origin
and passes a data URL to React. Login and shell headers otherwise render a Talos
monogram. The Wails window title follows the workshop name when available.

Theme mode is limited to `light`, `dark`, or `system`. CSS variables provide the
light and dark palettes, while `system` follows `prefers-color-scheme`. A user
selection is non-secret and persists in WebView local storage; without a local
selection, the authenticated workshop default applies. There is no custom CSS,
editable palette, or additional theme mode.
