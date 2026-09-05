# ADR-FIN-001 — Exact financial arithmetic and cent rounding

Status: Accepted.

Approval: the user approved the proposed exact-arithmetic and nearest-cent,
ties-away-from-zero policy in the continuation conversation, then requested
this ADR for the next tasks. The PRD itself does not choose the tie-breaking
method; that choice is recorded here under that approval.

Scope: COST-001, LABOR-003, and COST-002 in WP-COST-01.

## Context

PRD section 38 requires exact intermediate arithmetic, integer persisted cents,
and a documented rounding method. COST-001 needs deterministic rounding;
LABOR-003 divides monthly costs by estimated productive hours. No previously
approved rounding decision exists at reconciliation baseline `c78b4b2`.

Sources: [PRD](../../PRD.md), sections 4 (money and physical measures),
17.3 (machine-hour formula), 17.4 (internal labor assistant), and 38 (rounding);
[task acceptance criteria](../../IMPLEMENTATION_TASKS.md), COST-001,
LABOR-003, and COST-002.

## Decision

- Represent persisted money as signed 64-bit integer cents. Validate nonnegative
  amounts where the domain requires them.
- Represent percentage inputs as integer basis points (10,000 = 100%). Productive
  utilization must be greater than zero and at most 10,000 basis points.
- Use exact rational arithmetic from Go's standard library for intermediate
  calculations, parsing physical decimal inputs without binary floating point.
- Round final monetary amounts to the nearest cent, with exact half-cent ties
  away from zero. Thus 100.5 cents becomes 101 cents, and -100.5 becomes -101.
  Values of 100.49 and -100.49 cents become 100 and -100 respectively.
- Reject out-of-range final integer amounts and invalid/zero denominators;
  never wrap, clamp, or silently substitute defaults.

## Calculation boundaries

### COST-001 — Money and percentages

An exact amount is expressed in cents, including fractional cents during
calculation. Conversion to a persisted monetary value is explicit and returns
an integer or an error. Round the magnitude to the nearest integer, increase
it on an exact half tie, and then restore the sign. Zero remains zero.
Percentage application uses the exact ratio `basis_points / 10000`.
Do not impose utilization's 100% limit on the generic percentage primitive;
each caller validates the bounds appropriate to its domain.

### LABOR-003 — Internal hourly cost

Use the PRD formulas without changing their meaning:

```text
productive_hours = available_hours_per_month * productive_utilization_bps / 10000
internal_hourly_cost_cents =
    (target_monthly_compensation_cents + monthly_labor_overhead_cents)
    / productive_hours
```

For LABOR-003, calculate productive hours exactly as available monthly hours
times utilization. Divide the exact sum of monthly compensation and overhead
by those hours, then round the suggested hourly cost once to integer cents.
Display productive hours as an estimate without feeding display rounding back
into the calculation. Manual override accepts integer cents; saving a rate
remains an explicit, permission-protected action through the existing rate API.

Example: compensation of 300,000 cents plus overhead of 50,000 cents, with
160 available hours and 7,500 basis points utilization, gives 120 productive
hours and an exact hourly cost of 8,750/3 cents. The suggestion is 2,917 cents.

Monthly money inputs must be nonnegative; available hours must be positive
and utilization must be in `(0, 10000]`. Zero total monthly cost with a valid
denominator yields zero hourly cost. Do not invent missing productive hours or
utilization. The assistant is optional: users can configure an internal rate
manually without supplying these assumptions. When requesting a suggestion,
the monetary inputs must be explicit, including zero where appropriate.

Show the assumptions and keep the suggested value editable before explicit
save. Neither calculation nor rate edits rewrite existing Job snapshots.
This is internal human-time cost, separate from billable service pricing and
printer running time (PRD section 17.4).

### COST-002 — Machine-hour cost

Use the PRD section 17.3 formulas in cents:

```text
depreciation_per_hour = (acquisition_cost_cents - residual_value_cents)
                        / useful_life_hours
machine_hour_cost = depreciation_per_hour + maintenance_reserve_per_hour_cents
```

Keep both results exact for downstream calculations. A displayed cent value
may use the rounding primitive but must not replace the exact rate used later.
COST-002 does not persist a new rounded machine-hour rate. In particular, do
not round depreciation before adding the reserve or applying a later duration.

Validate nonnegative acquisition cost, residual value, and maintenance reserve;
residual value cannot exceed acquisition cost and useful life must be positive.
These constraints match the existing PRN-001 service. Equal acquisition and
residual values produce zero depreciation; a free machine with positive useful
life is valid. A zero or invalid lifetime is an error, even for a free machine.

Example: acquisition of 100,000 cents, residual of 10,000 cents, 7,000 hours of
useful life, and reserve of 2 cents/hour yield depreciation of 90/7 cents/hour
and a total of 104/7 cents/hour. A rounded hourly display is 15 cents; seven
hours at the exact rate are 104 cents, not 105.

Energy and human labor remain separate. No machine-rate override is introduced;
the PRD reserves that for a future documented parameter.

## Future boundaries

This ADR does not define later Job component, batch, quote, or channel
aggregation boundaries. Those packages must document their rounding boundaries
before using the primitive; intermediate values must retain precision.

## Consequences and alternatives

Exact rationals avoid intermediate precision loss and require no new dependency.
Ties away from zero are easy to explain, but repeated positive ties can introduce
upward bias. Half-to-even was considered for reducing that bias; the approved
choice is ties away from zero. Binary floating point and intermediate cent
rounding are excluded by PRD section 38.
Tests must cover positive/negative ties, non-ties, overflow, exact percentages,
and the labor formula including invalid utilization and zero productive hours.
Machine tests must cover residual value, fractional useful life, invalid/zero
lifetime, zero depreciation, and preservation of fractional cents. Inputs and
results must not be mutated unexpectedly by reusable exact-arithmetic helpers.

These are requirements for the upcoming implementation, not claims of tests
already executed. No additional dependency, migration, or separate ADR is
needed for these three tasks under the existing architecture.

No existing stored rates or historical snapshots are changed by this decision.
