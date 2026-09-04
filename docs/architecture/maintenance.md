# Printer maintenance history

Migration `00019_maintenance_events.sql` adds immutable maintenance history for
logical printers. Event types are cleaning, preventive, corrective,
replacement, upgrade, and inspection.

Every event retains its UTC performance time, description, downtime minutes,
creator, and creation time. Printer lifetime hours are optional exact decimals;
maintenance cost is optional integer cents. Neither value is inferred when it
is unavailable.

When supplied, printer hours must contain up to 15 integer digits and up to
three fractional digits (surrounding whitespace is trimmed). Blank, malformed,
negative, and over-precision values are rejected instead of becoming zero or
being rounded by PostgreSQL. Downtime is bounded to 0–2,147,483,647 minutes,
matching the database integer column. Invalid input returns HTTP 400.

History is listed newest-first at
`GET /api/v1/printers/{printerID}/maintenance` with `jobs.read`. Creating an
event at the same path requires `settings.manage`, matching logical printer
administration. The API exposes no printer credentials, remote commands, or
server-to-printer connection.
