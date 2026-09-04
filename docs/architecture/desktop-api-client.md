# Desktop server connection and API client

The Windows desktop stores one non-secret server base URL in the current
user's configuration directory. On Windows the file is
`%AppData%\TalosWorkshopManagement\connection.json`. Only absolute `http` and
`https` URLs are accepted; embedded credentials, query strings, fragments, and
non-HTTP schemes such as PostgreSQL are rejected.

The connection screen is available before login and exposes separate actions
to test or save the address. Saving does not require the server to be online,
so a temporarily unavailable LAN service does not prevent configuration.

The request path remains:

```text
React
  -> Wails binding
    -> desktop Go application
      -> typed Go API client
        -> workshop server
```

React contains no HTTP client for the workshop server. The native client owns
the base URL, an eight-second request timeout, bounded response reads, and the
common mapping for network, timeout, API-envelope, and invalid-response
failures. The client exposes public metadata and login operations. The shell
uses authenticated `GET /api/v1/settings` to validate its bearer session and
obtain the workshop default theme; other business endpoints remain later scope.

Compatibility requires API version `v1` and a desktop semantic version at
least as new as `minimum_desktop_version`. The desktop build version defaults
to `0.0.0` for development and can be replaced through Go linker flags for a
release build. An incompatible server is identified without persisting or
exposing a session token.

Login credentials follow the same native path through
`POST /api/v1/auth/login`. The password exists only for that local operation,
and the resulting token is handed directly to Windows secure credential storage
without crossing the Wails response boundary.

Public workshop branding follows the same native path. Logo URLs returned by
metadata must resolve to the configured server origin, responses are
size-bounded, and only PNG/JPEG media types become data URLs for React.
