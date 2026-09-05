# Catalog parts and immutable design versions

Catalog items own one or more printable parts. `catalog_parts` is mutable
operational structure: a designer can change a part's name, quantity, and notes.
Deleting a catalog item cascades through parts that have no design history. Once
a part has a design version, PostgreSQL restricts deletion of that part or its
item so immutable history cannot disappear. Repository integration tests make
both behaviors explicit.

Each part owns an immutable sequence of `design_versions`. A version is created
once and has no update or delete API. PostgreSQL enforces a unique
`(catalog_part_id, version)` pair. Significant design or provenance changes must
therefore create another version.

The version records origin, optional source URL and author, license name,
tri-state commercial permission, and attribution requirements. A null commercial
permission means unknown; it is different from an explicit denial.

Uploaded immutable file objects are linked through `design_version_files` with
one of these roles: `source`, `mesh`, `print`, `preview`, `documentation`, or
`other`. Links never use an original filename as a storage path. The same file
object can be linked to multiple design versions, and the version history response
includes role and digest metadata so the current print file is discoverable.

All reads require `catalog.read`; part/version/link mutations require
`catalog.write`. File content still downloads through the separately authorized
file endpoint.
