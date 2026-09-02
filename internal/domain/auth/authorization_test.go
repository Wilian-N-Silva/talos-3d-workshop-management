package auth

import "testing"

func TestOwnerHasEveryPermission(t *testing.T) {
	for _, permission := range AllPermissions() {
		if !RoleHasPermission(RoleOwner, permission) {
			t.Errorf("Owner missing %q", permission)
		}
	}
}

func TestRoleAndPermissionCatalogsAreDefensiveCopies(t *testing.T) {
	permissions := AllPermissions()
	permissions[0] = PermissionBackupManage
	if AllPermissions()[0] != PermissionCatalogRead {
		t.Fatal("mutating permission catalog result changed canonical order")
	}

	roles := AllRoles()
	roles[0] = RoleViewer
	if AllRoles()[0] != RoleOwner {
		t.Fatal("mutating role catalog result changed canonical order")
	}
}

func TestRepresentativeRolePermissionMatrix(t *testing.T) {
	tests := []struct {
		name       string
		role       Role
		permission Permission
		want       bool
	}{
		{name: "operator updates jobs", role: RoleOperator, permission: PermissionJobsUpdate, want: true},
		{name: "operator cannot manage settings", role: RoleOperator, permission: PermissionSettingsManage},
		{name: "designer uploads files", role: RoleDesigner, permission: PermissionFilesUpload, want: true},
		{name: "designer cannot manage inventory", role: RoleDesigner, permission: PermissionInventoryWrite},
		{name: "commercial manages pricing", role: RoleCommercial, permission: PermissionPricingManage, want: true},
		{name: "commercial cannot publish telemetry", role: RoleCommercial, permission: PermissionTelemetryPublish},
		{name: "viewer reads jobs", role: RoleViewer, permission: PermissionJobsRead, want: true},
		{name: "viewer cannot create jobs", role: RoleViewer, permission: PermissionJobsCreate},
		{name: "unknown role denied", role: Role("unknown"), permission: PermissionCatalogRead},
		{name: "unknown permission denied", role: RoleOwner, permission: Permission("unknown.permission")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := RoleHasPermission(test.role, test.permission); got != test.want {
				t.Fatalf("RoleHasPermission(%q, %q) = %t, want %t", test.role, test.permission, got, test.want)
			}
		})
	}
}

func TestPermissionsForRoleReturnsDefensiveOrderedCopy(t *testing.T) {
	permissions := PermissionsForRole(RoleDesigner)
	want := []Permission{
		PermissionCatalogRead,
		PermissionCatalogWrite,
		PermissionFilesRead,
		PermissionFilesUpload,
	}
	if len(permissions) != len(want) {
		t.Fatalf("designer permissions = %v, want %v", permissions, want)
	}
	for index := range want {
		if permissions[index] != want[index] {
			t.Fatalf("designer permissions = %v, want %v", permissions, want)
		}
	}
	permissions[0] = PermissionBackupManage
	if RoleHasPermission(RoleDesigner, PermissionBackupManage) {
		t.Fatal("mutating returned permissions changed the role mapping")
	}
}
