# Job energy measurements

Migration `00017_energy_measurements.sql` stores immutable, Job-scoped energy
evidence. Quantities use exact `NUMERIC` values and the tariff used at recording
time is retained as integer `energy_rate_cents_per_kwh`. Every record includes
its UTC occurrence time and the authenticated user who recorded it.

Sources and required evidence are explicit:

- `manual_meter` requires start/end kWh readings; the service derives their
  exact non-negative difference as `measured_kwh`;
- `smart_plug` and `imported` require a direct measured kWh value;
- `estimated` requires positive average power in watts and does not invent a
  measured kWh value.

No utilization factor or energy-cost formula is hidden in this capability.
Measured-versus-estimated precedence and Job-duration calculation belong to
the later COST-004 calculation service.

`GET /api/v1/jobs/{jobID}/energy` requires `jobs.read` and returns newest-first
history. `POST /api/v1/jobs/{jobID}/energy` requires `jobs.update`. There is no
update/delete endpoint: corrections are recorded as new evidence, preserving
the tariff and audit history used by future cost snapshots.
