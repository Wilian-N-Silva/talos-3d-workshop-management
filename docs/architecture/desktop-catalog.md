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

The item detail also loads parts and their bounded design history through native
Go. Designers can add a part, create a new immutable version with provenance and
license fields, and associate an existing immutable file object with a role. The
WebView never receives the bearer token.

For sellable items, the detail shows a non-blocking warning when any current part
has unknown commercial permission and a distinct warning when permission is
explicitly denied. Internal and prototype use remains available.

The first release of this workspace loads a bounded page of up to 100 items. The
server total is displayed so a later catalog navigation package can add search
and multi-page controls without changing the API contract.
