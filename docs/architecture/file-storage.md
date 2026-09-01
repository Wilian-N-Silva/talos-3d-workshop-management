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

Filesystem layout and atomic-write behavior belong to the STOR-002
infrastructure implementation.
