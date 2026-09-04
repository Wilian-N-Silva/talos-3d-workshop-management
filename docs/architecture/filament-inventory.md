# Filament inventory and weighing history

Materials describe reusable filament definitions: manufacturer, type, color,
exact nominal density, and replacement cost per kilogram in integer cents.
Physical values cross the API as decimal strings and are stored as PostgreSQL
`NUMERIC`; binary floating point is not used.

Every physical spool has a case-insensitive unique human code, material,
nominal/tare/opening weights, acquisition and replacement costs, storage
metadata, lifecycle status, and optional operational timestamps. A material
cannot be deleted while a spool refers to it.

Weighings are append-only `spool_measurements`. Recording one locks the spool in
a serializable transaction, rejects gross weight below tare, derives exact
remaining weight as `gross - tare`, inserts the history row, and updates the
spool's current remaining-weight cache and last-weighed timestamp atomically.
A spool with weighing history cannot be deleted.

Material and spool reads require `inventory.read`; mutations and weighing entry
require `inventory.write`. The desktop keeps bearer credentials in native Go and
exposes only inventory DTOs to React. History lists are newest-first and bounded
to 100 records.
