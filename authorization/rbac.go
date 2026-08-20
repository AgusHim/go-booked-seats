package authorization

import "go-ticketing/models"

type Permission string

const (
	PermissionCommunityRead   Permission = "community.read"
	PermissionCommunityManage Permission = "community.manage"
	PermissionMemberRead      Permission = "member.read"
	PermissionMemberManage    Permission = "member.manage"
	PermissionEventManage     Permission = "event.manage"
	PermissionPaymentRead     Permission = "payment.read"
	PermissionPaymentManage   Permission = "payment.manage"
	PermissionCheckinManage   Permission = "checkin.manage"
	PermissionJamaahRead      Permission = "jamaah.read"
	PermissionJamaahExport    Permission = "jamaah.export"
	PermissionContentModerate Permission = "content.moderate"
	PermissionClassroomManage Permission = "classroom.manage"
)

var permissionsByRole = map[string]map[Permission]struct{}{
	models.CommunityRoleOwner: permissionSet(
		PermissionCommunityRead,
		PermissionCommunityManage,
		PermissionMemberRead,
		PermissionMemberManage,
		PermissionEventManage,
		PermissionPaymentRead,
		PermissionPaymentManage,
		PermissionCheckinManage,
		PermissionJamaahRead,
		PermissionJamaahExport,
		PermissionContentModerate,
		PermissionClassroomManage,
	),
	models.CommunityRoleAdmin: permissionSet(
		PermissionCommunityRead,
		PermissionCommunityManage,
		PermissionMemberRead,
		PermissionMemberManage,
		PermissionEventManage,
		PermissionPaymentRead,
		PermissionCheckinManage,
		PermissionJamaahRead,
		PermissionJamaahExport,
		PermissionContentModerate,
		PermissionClassroomManage,
	),
	models.CommunityRoleEventManager: permissionSet(
		PermissionCommunityRead,
		PermissionMemberRead,
		PermissionEventManage,
		PermissionPaymentRead,
		PermissionCheckinManage,
		PermissionJamaahRead,
	),
	models.CommunityRoleCheckinStaff: permissionSet(
		PermissionCommunityRead,
		PermissionCheckinManage,
	),
	models.CommunityRoleModerator: permissionSet(
		PermissionCommunityRead,
		PermissionMemberRead,
		PermissionContentModerate,
	),
	models.CommunityRoleMentor: permissionSet(
		PermissionCommunityRead,
		PermissionMemberRead,
		PermissionClassroomManage,
	),
}

func HasPermission(role string, permission Permission) bool {
	permissions, exists := permissionsByRole[role]
	if !exists {
		return false
	}
	_, allowed := permissions[permission]
	return allowed
}

func permissionSet(permissions ...Permission) map[Permission]struct{} {
	result := make(map[Permission]struct{}, len(permissions))
	for _, permission := range permissions {
		result[permission] = struct{}{}
	}
	return result
}
