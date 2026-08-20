package controllers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"go-ticketing/models"
	"go-ticketing/repositories"
	"go-ticketing/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCommunityControllerListPublicContract(t *testing.T) {
	db, err := gorm.Open(
		sqlite.Open("file:community-controller-list?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Community{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, community := range []*models.Community{
		{Name: "Active", Type: models.CommunityTypeDakwah},
		{Name: "Inactive", Type: models.CommunityTypeGeneral, Status: models.CommunityStatusInactive},
	} {
		if err := db.Create(community).Error; err != nil {
			t.Fatalf("create community: %v", err)
		}
	}

	controller := NewCommunityController(
		services.NewCommunityService(repositories.NewCommunityRepository(db), nil),
	)
	app := fiber.New()
	app.Get("/api/v1/communities", controller.ListPublic)

	response, err := app.Test(
		httptest.NewRequest("GET", "/api/v1/communities?page=1&limit=1", nil),
	)
	if err != nil {
		t.Fatalf("request public communities: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("unexpected status %d", response.StatusCode)
	}
	var payload struct {
		Data []models.Community `json:"data"`
		Meta struct {
			Page  int   `json:"page"`
			Limit int   `json:"limit"`
			Total int64 `json:"total"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data) != 1 || payload.Data[0].Name != "Active" {
		t.Fatalf("unexpected data: %#v", payload.Data)
	}
	if payload.Meta.Page != 1 || payload.Meta.Limit != 1 || payload.Meta.Total != 1 {
		t.Fatalf("unexpected meta: %#v", payload.Meta)
	}

	invalid, err := app.Test(
		httptest.NewRequest("GET", "/api/v1/communities?limit=51", nil),
	)
	if err != nil {
		t.Fatalf("request invalid pagination: %v", err)
	}
	defer invalid.Body.Close()
	if invalid.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", invalid.StatusCode)
	}
}
