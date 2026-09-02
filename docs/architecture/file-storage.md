# Immutable file storage contract

The application port is `internal/application/storage.Store`. Files are stored
as immutable byte objects; catalog records and upload metadata refer to them but
are not part of the storage contract.

The boundary intentionally accepts only an `io.Reader` when writing. Original
filenames, content types, owners, and catalog identifiers cannot influence a
physical storage path. `Put` returns an opaque `ObjectKey`, a SHA-256 digest, and
the stored byte count.

Serialized keys must pass `ParseObjectKey` before use. The strong key type has
an unexported value and permits only bounded ASCII letters, digits, hyphens, and
underscores. Path separators, dots, drive prefixes, percent escaping, and
Unicode lookalikes therefore cannot cross the API boundary.

Objects cannot be updated through the interface. `DeleteForCleanup` exists only
to remove an unreferenced object when a creation workflow fails before its
metadata transaction completes. Product-level deletion must remove references
according to retention rules instead of treating stored objects as mutable
files.

`LocalFilesystemStorage` implements the port below the configured server data
directory. The physical layout is derived only from the SHA-256 digest:

```text
<data>/objects/<first two hex characters>/<full SHA-256 hex digest>
```

Writes are streamed to a same-filesystem staging directory, flushed, closed,
and atomically published without replacing an existing path. Concurrent writes
of identical content converge on the same object. If an existing object's bytes
do not match its content address, the write fails instead of overwriting it.

Migration `00008_files.sql` adds immutable metadata with UUID identity,
32-byte SHA-256 uniqueness, a validated opaque storage key, original name,
content type, byte size, uploader, and UTC creation time. Metadata names never
participate in physical paths. Identical bytes reuse the existing metadata and
content-addressed object; the first immutable metadata record remains canonical.

The first consumer is the [workshop logo](workshop-settings.md). Generic
authenticated file upload and file-by-ID download remain FILE-002 and FILE-003
scope.
