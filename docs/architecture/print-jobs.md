# Manual print Job lifecycle

A print Job represents one physical execution and does not require a customer,
quote, order, or sale. `purpose` records why the printer was used; it never
creates revenue or implies commercial profit/loss.

## Persistence

Migration `00015_print_jobs.sql` creates `print_jobs` and append-only
`job_events`. Jobs reference a catalog item, one immutable design version, a
logical printer, and their creator. The repository verifies that the design
belongs to the selected catalog item. `order_item_id` is nullable and is not
accepted by the current API; the future commercial module may associate it
explicitly when its table exists.

Purposes are `test`, `prototype`, `production`, `maintenance`, `internal`, and
`personal`. Status progresses through:

```text
draft -> prepared -> printing -> awaiting_review -> completed
                                      \-> failed
```

Draft, prepared, and printing Jobs may also be cancelled. A detected/manual
printing failure can move `printing` directly to `failed`. Terminal Jobs cannot
transition again.

Migration `00016_job_material_usage.sql` adds N material usages per Job. Each
record identifies a physical spool and its derived material, a role (`model`,
`support`, `purge`, or `other`), exact planned/actual grams and optional meters,
and the measurement source. The same spool may appear more than once when its
role differs. Historical and replacement cost snapshot fields are separate,
nullable integer cents. Once captured, those cost snapshots are immutable even
when quantities or measurement source are corrected; recording usage never
creates revenue.

## Quality review

Printer completion moves a Job to `awaiting_review`; it does not approve the
output. An explicit review records `approved`, `partial`, or `failed`, good and
scrap quantities, result notes, and the completion time. Approved and partial
reviews complete the Job; a failed review marks it failed.

## Events and concurrency

Creation, transitions, and reviews append a significant event in the same
database transaction as the Job mutation. Events record UTC occurrence time,
actor user, source desktop device, and bounded JSON metadata. Conditional
status updates reject concurrent/stale transitions. No endpoint can mutate or
delete an event, and high-frequency telemetry is not stored here.

## HTTP contract

The resource root is `/api/v1/jobs`. List/get/events require `jobs.read`, create
requires `jobs.create`, metadata updates/transitions/deletion require
`jobs.update`, and quality review requires `jobs.evaluate`. Only draft or
prepared metadata is editable, and only draft Jobs can be deleted.

Material usage is available at `/api/v1/jobs/{jobID}/materials`. Reads require
`jobs.read`; create, replace, and delete require `jobs.update`. The list response
includes exact planned and actual gram totals. Usage may be added or corrected
through `awaiting_review`; deletion is limited to draft and prepared Jobs so
production evidence is not silently removed.
