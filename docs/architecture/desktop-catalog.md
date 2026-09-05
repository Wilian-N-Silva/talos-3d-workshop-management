# Desktop catalog workspace

The authenticated shell exposes a basic catalog list/detail workspace when the
session includes `catalog.read`. Purpose, sellable state, SKU, tags, and
active/archived status are visible in the detail view.

Create and edit affordances require `catalog.write` in the safe permission list
provided to React. This client-side visibility is not authorization: native Go
sends the secure bearer token to the server, whose permission middleware remains
authoritative.

React calls only the Wails methods `ListCatalogItems`, `CreateCatalogItem`, and
`UpdateCatalogItem`. The bearer token is loaded from Windows Credential Manager
inside the native application layer and is never returned to the WebView. A 401
response removes the rejected local credential.

The first release of this workspace loads a bounded page of up to 100 items. The
server total is displayed so a later catalog navigation package can add search
and multi-page controls without changing the API contract.
