# Desktop authentication boundary

The Windows desktop authenticates through the required local boundary:

```text
React login form
  -> Wails Login binding
    -> native Go API client
      -> POST /api/v1/auth/login
```

React holds the password only while the login form is active and passes it only
to the local Go method. It does not implement server HTTP, log credentials, or
persist passwords or bearer tokens. The Wails login result contains only safe
authentication state: user identifiers, display name, login identifier, and
expiry.

The native client receives the opaque bearer token and immediately persists a
small session record as a Windows Credential Manager generic credential. Its
target is namespaced by a SHA-256 digest of the configured server URL, keeping
sessions from different workshop servers separate without putting tokens in
`connection.json`. Credential blobs are limited to the Windows maximum size and
cleared from temporary byte slices after encoding or decoding.

Startup restores a non-expired secure credential and transitions to the
application shell. Expired credentials are removed. Logout deletes the local
credential idempotently. The current package does not expose the token to React
or add authenticated business calls; server-side authorization remains
authoritative when those calls are introduced.
