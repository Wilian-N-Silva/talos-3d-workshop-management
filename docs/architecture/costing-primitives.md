# Exact cost calculations and labor rate assistant

WP-COST-01 implements COST-001, LABOR-003, and COST-002 under
[ADR-FIN-001](../adr/ADR-FIN-001.md). The pure `internal/domain/costing` package
uses standard-library rational arithmetic, integer basis points, and explicit
nearest-cent rounding with half ties away from zero. It rejects overflow when
converting a result to int64 cents. Input decimal strings allow up to 15 whole
digits and six fractional digits; scientific notation and fractions are not
accepted as input. No schema changes or third-party dependencies are required.

## Labor suggestion API

`POST /api/v1/costing/labor-rates/suggestion` requires an authenticated session
with `costing.read`. It is a pure preview and never writes a rate or a Job.
Send `Content-Type: application/json` with all four assumptions:

```json
{
  "target_monthly_compensation_cents": 300000,
  "monthly_labor_overhead_cents": 50000,
  "available_hours_per_month": "160",
  "productive_utilization_bps": 7500
}
```

The response is:

```json
{
  "productive_hours": "120.0000000000",
  "internal_hourly_cost_cents": 2917
}
```

Costs must be explicit nonnegative integer cents. Hours must be positive and
utilization must be in `(0, 10000]`. Productive hours retain all ten possible
decimal places; the suggested monetary rate rounds only once, after division.
Missing assumptions, invalid values, and overflowing suggestions return 400
with `invalid_labor_assumptions`. Malformed JSON uses the existing bounded JSON
decoder error contract. All responses use the existing authorization boundary;
successful responses carry `Cache-Control: no-store`.

## Desktop flow

Users with `costing.read` can view rates and use the optional assistant. Users
with `costing.manage` can explicitly copy a suggestion into the rate form,
override it, and save a new or existing rate. They can also enter a manual rate
without assistant assumptions. Existing POST/PUT labor-rate routes enforce the
write permission. Editing assumptions clears the old suggestion.

React parses BRL decimal text to integer cents using BigInt and passes cents as
strings through Wails. Native Go converts these strings to int64 JSON numbers
for the server and returns monetary values to React as strings. Server requests
and bearer credentials remain in native Go. No customer, order, billable rate,
or machine time is involved. Existing Job labor snapshots remain immutable.

## Machine hour calculation

`CalculateMachineRate` applies PRD 17.3: `(acquisition - residual) / useful life`
plus the hourly maintenance reserve. It returns separate exact depreciation and
total hourly rates. It accepts the existing printer costing values and rejects
negative costs, residual above acquisition, or nonpositive useful life. Zero
depreciation is valid. Energy and labor remain separate. No override, endpoint,
or persisted machine rate is added by COST-002.

Downstream duration calculations must use the exact rate, not a rounded display.
For example, a rate of `104/7` cents/hour costs exactly 104 cents over seven
hours even though its rounded hourly display is 15 cents.

## Verification

Pure tests cover ties on both signs, int64 boundaries, decimal parsing, exact
percentages, invalid denominators, the labor formula, and machine fractional
cents. HTTP tests cover required assumptions, exact output, 401/403, and invalid
requests. Native tests cover explicit override requests, full int64 transport,
invalid responses, and keeping bearer tokens native. Existing PostgreSQL tests
cover saving rates and retaining historical Job snapshots after rate updates.

Browser verification exercised the actual React form with a temporary native
bridge fixture: 120 productive hours, R$ 29,17 suggestion, explicit R$ 30,25
override/save, and read-only controls. This is UI verification, not a live Wails
to PostgreSQL end-to-end test. The production Windows Wails build is validated
separately.
