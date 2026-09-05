# Logical printer registry

The server stores only non-sensitive information about workshop printers. The
registry supports Job planning and future costing without creating a network
path from the server to a printer.

## Persistence

Migration `00014_printers.sql` creates `printers` with:

- name, manufacturer, model, nozzle diameter, and location;
- acquisition and residual values as integer cents;
- exact useful-life hours and a maintenance reserve per hour in cents;
- `active`, `maintenance`, or `retired` lifecycle status;
- notes and UTC timestamps.

Names are case-insensitively unique. Nozzle diameter and useful life use exact
PostgreSQL `NUMERIC` values and API decimal strings. Residual value cannot
exceed acquisition cost. A printer referenced by a future Job will be protected
by that Job migration's foreign key rather than being silently removed.

## HTTP contract

The resource root is `/api/v1/printers`:

- list and get require `jobs.read`, because these operations provide logical
  printer choices for Job workflows;
- create, replace, and delete require `settings.manage`, because printer
  registration changes workshop configuration.

Responses never contain connection credentials. Request decoding rejects
unknown fields, so `access_code` and similar secrets cannot be accepted by this
contract.

## Security boundary

This registry performs database CRUD only. It does not discover printers,
contact Bambu devices, store LAN credentials, publish commands, or expose
start/pause/resume/cancel routes. Bambu access codes remain local to an
authorized Windows desktop in later telemetry work.
