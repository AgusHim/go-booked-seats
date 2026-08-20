package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-ticketing/authorization"
	"go-ticketing/models"
	"go-ticketing/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCommunityPermissionEnforcesTenantRole(t *testing.T) {
	t.Setenv("JWT_SECRET", "community-middleware-test-secret")
	t.Setenv("REQUIRE_JWT_SECRET", "true")

	db, err := gorm.Open(
		sqlite.Open("file:community-middleware?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Community{},
		&models.CommunityMember{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	owner := createPermissionTestUser(t, db, "owner@example.test", "user")
	staff := createPermissionTestUser(t, db, "staff@example.test", "user")
	outsider := createPermissionTestUser(t, db, "outsider@example.test", "user")
	platformAdmin := createPermissionTestUser(t, db, "admin@example.test", "admin")
	community := &models.Community{Name: "RBAC Test", Type: models.CommunityTypeGeneral}
	if err := db.Create(community).Error; err != nil {
		t.Fatalf("create community: %v", err)
	}
	otherCommunity := &models.Community{Name: "Other Tenant", Type: models.CommunityTypeGeneral}
	if err := db.Create(otherCommunity).Error; err != nil {
		t.Fatalf("create other community: %v", err)
	}
	for _, member := range []*models.CommunityMember{
		{
			CommunityID: community.ID,
			UserID:      owner.ID,
			Role:        models.CommunityRoleOwner,
			Status:      models.CommunityMemberStatusActive,
		},
		{
			CommunityID: community.ID,
			UserID:      staff.ID,
			Role:        models.CommunityRoleCheckinStaff,
			Status:      models.CommunityMemberStatusActive,
		},
		{
			CommunityID: otherCommunity.ID,
			UserID:      outsider.ID,
			Role:        models.CommunityRoleOwner,
			Status:      models.CommunityMemberStatusActive,
		},
	} {
		if err := db.Create(member).Error; err != nil {
			t.Fatalf("create member: %v", err)
		}
	}

	app := fiber.New()
	app.Get(
		"/portal/:community_id/payments",
		AuthProtected(db),
		CommunityPermission(db, authorization.PermissionPaymentRead),
		func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"allowed": true})
		},
	)

	tests := []struct {
		name       string
		user       *models.User
		wantStatus int
	}{
		{"owner allowed", owner, fiber.StatusOK},
		{"checkin staff denied", staff, fiber.StatusForbidden},
		{"outsider denied", outsider, fiber.StatusForbidden},
		{"platform admin allowed", platformAdmin, fiber.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := utils.GenerateJWT(test.user.ID)
			if err != nil {
				t.Fatalf("generate token: %v", err)
			}
			req := httptest.NewRequest(
				http.MethodGet,
				"/portal/"+community.ID+"/payments",
				nil,
			)
			req.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(req)
			if err != nil {
				t.Fatalf("app test: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != test.wantStatus {
				var body map[string]interface{}
				_ = json.NewDecoder(response.Body).Decode(&body)
				t.Fatalf("status=%d want=%d body=%v", response.StatusCode, test.wantStatus, body)
			}
		})
	}
}

func createPermissionTestUser(
	t *testing.T,
	db *gorm.DB,
	email string,
	role string,
) *models.User {
	t.Helper()
	user := &models.User{Name: email, Email: email, Password: "hash", Role: role}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	return user
}
