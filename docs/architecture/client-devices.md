# Client-device persistence

Authorized desktop installations are stored in PostgreSQL by migration
`00004_client_devices.sql`. Each audit record has a server-generated UUID,
display name, operating-system description, application version, creation
timestamp, and last-seen timestamp.

Text metadata must be nonblank and free of surrounding whitespace. OS and
application versions are intentionally descriptive strings rather than closed
enums so legitimate platform and pre-release version labels remain auditable.
Timestamps use PostgreSQL `timestamptz`; the repository normalizes scanned
values to UTC. A newly registered device is last seen at its creation time,
and later updates cannot move that value backward.

The record is independent of a user. AUTH-005 will relate opaque sessions to
both users and devices, enabling audit and session revocation without making a
desktop client a remotely controlled agent. The schema contains no printer
credentials, command channel, or remote-control fields.
