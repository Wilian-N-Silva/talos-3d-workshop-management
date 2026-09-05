# Supply inventory and stock movements

Non-filament supplies store an exact current quantity, unit, replacement unit
cost in integer cents, and an exact minimum quantity. Quantity values cross the
API as decimal strings and are persisted as PostgreSQL `NUMERIC`.

The current quantity is read-only inventory state. It starts at zero and changes
only through append-only stock movements. Movement `quantity` is a signed delta:

- `purchase` and `return` require a positive quantity;
- `consume` and `discard` require a negative quantity;
- `adjustment` accepts either sign but cannot be zero.

Recording a movement locks its supply row, verifies the resulting quantity is
not negative, updates the current cache, and inserts the audit row in one
transaction. A rejected movement changes neither state nor history. Supplies
with movement history cannot be deleted.

Low inventory is derived and is never persisted as a separate alert. Supplies
with a positive configured minimum are low when
`current_quantity <= minimum_quantity`; a zero minimum disables the warning.
Active spools are low when their measured remaining weight is at or below the query threshold; the API
accepts `spool_threshold_g` and defaults it to exactly 100 grams.

Supply and low-stock reads require `inventory.read`; supply mutations and stock
movements require `inventory.write`. Lists are newest-first where historical and
bounded to 100 records.
