# Catalog items

Catalog items represent any reusable workshop item, including products,
prototypes, tooling, tests, internal items, and personal items. Commercial
context is not required.

## Persistence

Migration `00009_catalog_items.sql` defines UUID identifiers, optional SKU,
purpose and status checks, sellable state, UTC timestamps, and a JSONB tag
array. Tags are normalized by the application service to trimmed lowercase
values, deduplicated, sorted, and bounded before persistence. The JSONB GIN
index supports exact tag membership filtering.

Status is one of `active` or `archived`. Archiving is the durable way to remove
an item from normal active workflows while preserving its identity. DELETE is
available while no dependent record prevents it; later catalog and operational
foreign keys may reject deletion.

## HTTP contract

The resource root is `/api/v1/catalog/items`:

- `GET /catalog/items` and `GET /catalog/items/{itemID}` require `catalog.read`;
- `POST /catalog/items`, `PUT /catalog/items/{itemID}`, and
  `DELETE /catalog/items/{itemID}` require `catalog.write`.

List filters are `purpose`, `status`, `sellable`, `tag`, and `q`. Pagination is
`limit` plus `offset`, with a default limit of 50 and maximum of 100. Responses
include the matching total. All filters are passed to parameterized SQL.
