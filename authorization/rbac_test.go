package authorization

import (
	"testing"

	"go-ticketing/models"
)

func TestRolePermissions(t *testing.T) {
	tests := []struct {
		name       string
		role       string
		permission Permission
		want       bool
	}{
		{"owner can manage members", models.CommunityRoleOwner, PermissionMemberManage, true},
		{"admin can manage public community profile", models.CommunityRoleAdmin, PermissionCommunityManage, true},
		{"admin cannot manage payment", models.CommunityRoleAdmin, PermissionPaymentManage, false},
		{"event manager can manage event", models.CommunityRoleEventManager, PermissionEventManage, true},
		{"checkin staff cannot read payment", models.CommunityRoleCheckinStaff, PermissionPaymentRead, false},
		{"mentor can manage classroom", models.CommunityRoleMentor, PermissionClassroomManage, true},
		{"unknown role denied", "unknown", PermissionCommunityRead, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := HasPermission(test.role, test.permission); got != test.want {
				t.Fatalf("HasPermission(%q, %q) = %v, want %v", test.role, test.permission, got, test.want)
			}
		})
	}
}
