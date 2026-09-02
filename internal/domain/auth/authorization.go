package auth

// Permission is an operation authorized by the server. Handlers and services
// require permissions rather than checking role names directly.
type Permission string

const (
	PermissionCatalogRead      Permission = "catalog.read"
	PermissionCatalogWrite     Permission = "catalog.write"
	PermissionFilesRead        Permission = "files.read"
	PermissionFilesUpload      Permission = "files.upload"
	PermissionInventoryRead    Permission = "inventory.read"
	PermissionInventoryWrite   Permission = "inventory.write"
	PermissionJobsRead         Permission = "jobs.read"
	PermissionJobsCreate       Permission = "jobs.create"
	PermissionJobsUpdate       Permission = "jobs.update"
	PermissionJobsEvaluate     Permission = "jobs.evaluate"
	PermissionCostingRead      Permission = "costing.read"
	PermissionCostingManage    Permission = "costing.manage"
	PermissionPricingRead      Permission = "pricing.read"
	PermissionPricingManage    Permission = "pricing.manage"
	PermissionCommercialRead   Permission = "commercial.read"
	PermissionCommercialWrite  Permission = "commercial.write"
	PermissionTelemetryRead    Permission = "telemetry.read"
	PermissionTelemetryPublish Permission = "telemetry.publish"
	PermissionUsersManage      Permission = "users.manage"
	PermissionSettingsManage   Permission = "settings.manage"
	PermissionBackupManage     Permission = "backup.manage"
)

var allPermissions = [...]Permission{
	PermissionCatalogRead,
	PermissionCatalogWrite,
	PermissionFilesRead,
	PermissionFilesUpload,
	PermissionInventoryRead,
	PermissionInventoryWrite,
	PermissionJobsRead,
	PermissionJobsCreate,
	PermissionJobsUpdate,
	PermissionJobsEvaluate,
	PermissionCostingRead,
	PermissionCostingManage,
	PermissionPricingRead,
	PermissionPricingManage,
	PermissionCommercialRead,
	PermissionCommercialWrite,
	PermissionTelemetryRead,
	PermissionTelemetryPublish,
	PermissionUsersManage,
	PermissionSettingsManage,
	PermissionBackupManage,
}

// AllPermissions returns a defensive copy of the stable Release 1 catalog.
func AllPermissions() []Permission {
	return append([]Permission(nil), allPermissions[:]...)
}

// Role is a fixed Release 1 permission profile.
type Role string

const (
	RoleOwner      Role = "owner"
	RoleOperator   Role = "operator"
	RoleDesigner   Role = "designer"
	RoleCommercial Role = "commercial"
	RoleViewer     Role = "viewer"
)

var allRoles = [...]Role{RoleOwner, RoleOperator, RoleDesigner, RoleCommercial, RoleViewer}

// AllRoles returns a defensive copy of the stable Release 1 role catalog.
func AllRoles() []Role {
	return append([]Role(nil), allRoles[:]...)
}

var rolePermissions = map[Role]map[Permission]struct{}{
	RoleOwner: permissionSet(allPermissions[:]...),
	RoleOperator: permissionSet(
		PermissionCatalogRead,
		PermissionFilesRead,
		PermissionInventoryRead,
		PermissionInventoryWrite,
		PermissionJobsRead,
		PermissionJobsCreate,
		PermissionJobsUpdate,
		PermissionJobsEvaluate,
		PermissionTelemetryRead,
		PermissionTelemetryPublish,
	),
	RoleDesigner: permissionSet(
		PermissionCatalogRead,
		PermissionCatalogWrite,
		PermissionFilesRead,
		PermissionFilesUpload,
	),
	RoleCommercial: permissionSet(
		PermissionCatalogRead,
		PermissionCostingRead,
		PermissionPricingRead,
		PermissionPricingManage,
		PermissionCommercialRead,
		PermissionCommercialWrite,
	),
	RoleViewer: permissionSet(
		PermissionCatalogRead,
		PermissionFilesRead,
		PermissionInventoryRead,
		PermissionJobsRead,
		PermissionCostingRead,
		PermissionPricingRead,
		PermissionCommercialRead,
		PermissionTelemetryRead,
	),
}

// RoleHasPermission applies the fixed role-to-permission mapping. Unknown
// roles and permissions are denied by default.
func RoleHasPermission(role Role, permission Permission) bool {
	permissions, ok := rolePermissions[role]
	if !ok {
		return false
	}
	_, ok = permissions[permission]
	return ok
}

// PermissionsForRole returns a defensive, catalog-ordered permission list.
func PermissionsForRole(role Role) []Permission {
	permissions := make([]Permission, 0, len(allPermissions))
	for _, permission := range allPermissions {
		if RoleHasPermission(role, permission) {
			permissions = append(permissions, permission)
		}
	}
	return permissions
}

func permissionSet(permissions ...Permission) map[Permission]struct{} {
	set := make(map[Permission]struct{}, len(permissions))
	for _, permission := range permissions {
		set[permission] = struct{}{}
	}
	return set
}
