# Catalog supply bill of materials

Catalog items can record the non-filament supplies consumed to produce one
unit. This bill of materials (BOM) is available for every catalog purpose,
including internal, prototype, test, and personal work; it does not create or
require commercial context.

## Persistence

Migration `00013_catalog_bom.sql` creates `catalog_bom_items`. Each row links
one catalog item to one supply and stores an exact positive quantity per unit,
a non-negative waste percentage, optional notes, and UTC timestamps. A supply
can appear only once in an item's BOM. Deleting the catalog item cascades to its
BOM, while a referenced supply cannot be deleted.

Filament remains in the spool/material inventory and is deliberately not
represented as a supply BOM row.

## Cost preview

The read endpoint returns a non-persisted replacement-cost preview using the
supply's current replacement unit cost:

```text
effective quantity = quantity per unit * (1 + waste percent / 100)
exact cost cents = effective quantity * replacement unit cost cents
```

The application performs this calculation with exact decimal rational
arithmetic. Values are returned as decimal strings, `rounding_applied` is
`false`, and no official price or cost snapshot is created. Batch calculation
and an approved final-cent rounding rule belong to the later costing workflow.

## HTTP contract

The resource root is `/api/v1/catalog/items/{itemID}/bom`:

- `GET /catalog/items/{itemID}/bom` and
  `GET /catalog/items/{itemID}/bom/{bomItemID}` require `catalog.read`;
- `POST /catalog/items/{itemID}/bom`,
  `PUT /catalog/items/{itemID}/bom/{bomItemID}`, and
  `DELETE /catalog/items/{itemID}/bom/{bomItemID}` require `catalog.write`.

All identifiers are server-validated and all SQL is parameterized. The desktop
bridge keeps the bearer token in the native Go layer and exposes the preview
and CRUD operations to the catalog workspace.
