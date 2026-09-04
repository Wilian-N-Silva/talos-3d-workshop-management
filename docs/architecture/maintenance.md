# Printer maintenance history

Migration `00019_maintenance_events.sql` adds immutable maintenance history for
logical printers. Event types are cleaning, preventive, corrective,
replacement, upgrade, and inspection.

Every event retains its UTC performance time, description, downtime minutes,
creator, and creation time. Printer lifetime hours are optional exact decimals;
maintenance cost is optional integer cents. Neither value is inferred when it
is unavailable.

History is listed newest-first at
`GET /api/v1/printers/{printerID}/maintenance` with `jobs.read`. Creating an
event at the same path requires `settings.manage`, matching logical printer
administration. The API exposes no printer credentials, remote commands, or
server-to-printer connection.
