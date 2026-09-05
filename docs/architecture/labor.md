# Internal labor rates and Job time

The optional [labor rate assistant](costing-primitives.md) calculates internal
hourly suggestions from monthly assumptions. Saving or overriding a suggestion
uses the existing rate endpoints below and always requires explicit action.

Migration `00018_labor.sql` separates reusable internal labor cost rates from
immutable Job time entries. `labor_rates` contains only a name, activity type,
integer `cost_hourly_rate_cents`, and active state. It deliberately contains no
billable rate, customer price, markup, or margin field.

Supported activities are setup, material handling, support removal, finishing,
painting, assembly, packaging, modeling, customization, consulting, and other.
Rate mutations require `costing.manage`; reads require `costing.read`.

`job_labor_entries` records positive whole minutes against any Job, including
test, prototype, internal, personal, and maintenance Jobs. Creation copies the
selected active rate's activity and internal hourly cents into the entry. Later
rate edits therefore do not alter historical Job evidence. Entries are
create/list only in this package and cannot be overwritten.

`GET /api/v1/jobs/{jobID}/labor` returns total minutes and minutes grouped by
activity. It intentionally does not calculate money: deterministic
minutes-times-rate calculation belongs to COST-005. Reads require
`costing.read`; entry creation requires `costing.manage` and records the
authenticated user and UTC occurrence time.
