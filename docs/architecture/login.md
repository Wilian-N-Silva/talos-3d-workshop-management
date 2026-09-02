# Desktop login

`POST /api/v1/auth/login` authenticates a user with a case-insensitive login
identifier and Argon2id password verification. Unknown identifiers, incorrect
passwords, and disabled accounts produce the same public error. Unknown and
structurally invalid identifiers still perform verification against a startup-
generated dummy Argon2id hash to reduce account-enumeration timing differences.

After credential verification, the service registers a client device or
refreshes the metadata and last-seen time for an existing device ID. It then
updates the user's last-login timestamp and issues an opaque session. The
response contains safe user/device metadata, expiry, and the plaintext token;
it never includes password hashes or token hashes.

Login requests are limited per direct socket-peer IP using a concurrency-safe,
bounded in-memory fixed window. Forwarding headers are deliberately ignored
because Release 1 does not define a trusted reverse-proxy boundary. The default
allows five attempts per minute and can be changed with
`TALOS_LOGIN_RATE_LIMIT_ATTEMPTS` and `TALOS_LOGIN_RATE_LIMIT_WINDOW`. Limiter
state is process-local and resets when the server restarts.
