package controllers

import (
	"errors"
	"strconv"

	"go-ticketing/httpapi"
	"go-ticketing/models"
	"go-ticketing/repositories"
	"go-ticketing/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type CommunityController struct {
	service services.CommunityService
}

func NewCommunityController(service services.CommunityService) *CommunityController {
	return &CommunityController{service: service}
}

func (c *CommunityController) Create(ctx *fiber.Ctx) error {
	userID, ok := currentUserID(ctx)
	if !ok {
		return httpapi.Error(ctx, fiber.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required")
	}

	var community models.Community
	if err := ctx.BodyParser(&community); err != nil {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid input")
	}
	if err := c.service.Create(ctx.Context(), &community, userID); err != nil {
		status := fiber.StatusInternalServerError
		if errors.Is(err, services.ErrInvalidCommunity) ||
			errors.Is(err, services.ErrInvalidCommunityType) {
			status = fiber.StatusBadRequest
		}
		return httpapi.Error(ctx, status, "COMMUNITY_CREATE_FAILED", err.Error())
	}

	return httpapi.Data(ctx, fiber.StatusCreated, community)
}

func (c *CommunityController) GetPublic(ctx *fiber.Ctx) error {
	community, err := c.service.GetPublic(ctx.Context(), ctx.Params("slug"))
	if err != nil {
		status := fiber.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = fiber.StatusNotFound
		}
		return httpapi.Error(ctx, status, "COMMUNITY_NOT_FOUND", "Community not found")
	}
	return httpapi.Data(ctx, fiber.StatusOK, community)
}

func (c *CommunityController) ListPublic(ctx *fiber.Ctx) error {
	page, err := strconv.Atoi(ctx.Query("page", "1"))
	if err != nil || page < 1 {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "INVALID_FILTER", "page must be a positive integer")
	}
	limit, err := strconv.Atoi(ctx.Query("limit", "12"))
	if err != nil || limit < 1 || limit > 50 {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "INVALID_FILTER", "limit must be between 1 and 50")
	}
	communities, total, err := c.service.ListPublic(
		ctx.Context(),
		limit,
		(page-1)*limit,
	)
	if err != nil {
		return httpapi.Error(ctx, fiber.StatusInternalServerError, "COMMUNITY_LIST_FAILED", "Unable to load communities")
	}
	return ctx.JSON(fiber.Map{
		"data": communities,
		"meta": fiber.Map{
			"request_id": httpapi.RequestID(ctx),
			"page":       page,
			"limit":      limit,
			"total":      total,
		},
	})
}

func (c *CommunityController) ListMine(ctx *fiber.Ctx) error {
	userID, ok := currentUserID(ctx)
	if !ok {
		return httpapi.Error(ctx, fiber.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required")
	}
	communities, err := c.service.ListForUser(ctx.Context(), userID)
	if err != nil {
		return httpapi.Error(ctx, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
	}
	return httpapi.Data(ctx, fiber.StatusOK, communities)
}

func (c *CommunityController) Follow(ctx *fiber.Ctx) error {
	userID, ok := currentUserID(ctx)
	if !ok {
		return httpapi.Error(ctx, fiber.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required")
	}
	if err := c.service.Follow(ctx.Context(), ctx.Params("community_id"), userID); err != nil {
		status := fiber.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = fiber.StatusNotFound
		}
		return httpapi.Error(ctx, status, "COMMUNITY_FOLLOW_FAILED", "Unable to follow community")
	}
	return httpapi.Data(ctx, fiber.StatusOK, fiber.Map{"following": true})
}

func (c *CommunityController) Unfollow(ctx *fiber.Ctx) error {
	userID, ok := currentUserID(ctx)
	if !ok {
		return httpapi.Error(ctx, fiber.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required")
	}
	if err := c.service.Unfollow(ctx.Context(), ctx.Params("community_id"), userID); err != nil {
		return httpapi.Error(ctx, fiber.StatusInternalServerError, "COMMUNITY_UNFOLLOW_FAILED", "Unable to unfollow community")
	}
	return httpapi.Data(ctx, fiber.StatusOK, fiber.Map{"following": false})
}

func (c *CommunityController) FollowState(ctx *fiber.Ctx) error {
	userID, ok := currentUserID(ctx)
	if !ok {
		return httpapi.Error(ctx, fiber.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required")
	}
	following, err := c.service.IsFollowing(
		ctx.Context(),
		ctx.Params("community_id"),
		userID,
	)
	if err != nil {
		return httpapi.Error(ctx, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
	}
	return httpapi.Data(ctx, fiber.StatusOK, fiber.Map{"following": following})
}

func (c *CommunityController) ListFollowing(ctx *fiber.Ctx) error {
	userID, ok := currentUserID(ctx)
	if !ok {
		return httpapi.Error(ctx, fiber.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required")
	}
	communities, err := c.service.ListFollowing(ctx.Context(), userID)
	if err != nil {
		return httpapi.Error(ctx, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
	}
	return httpapi.Data(ctx, fiber.StatusOK, communities)
}

func (c *CommunityController) ListMembers(ctx *fiber.Ctx) error {
	members, err := c.service.ListMembers(
		ctx.Context(),
		ctx.Params("community_id"),
	)
	if err != nil {
		return httpapi.Error(ctx, fiber.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
	}
	return httpapi.Data(ctx, fiber.StatusOK, members)
}

func (c *CommunityController) GetPortal(ctx *fiber.Ctx) error {
	community, err := c.service.GetPortal(ctx.Context(), ctx.Params("community_id"))
	if err != nil {
		status := fiber.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = fiber.StatusNotFound
		}
		return httpapi.Error(ctx, status, "COMMUNITY_NOT_FOUND", "Community not found")
	}
	role, _ := ctx.Locals("community_role").(string)
	return httpapi.Data(ctx, fiber.StatusOK, fiber.Map{
		"community": community,
		"role":      role,
	})
}

func (c *CommunityController) UpdateProfile(ctx *fiber.Ctx) error {
	var input services.CommunityProfileInput
	if err := ctx.BodyParser(&input); err != nil {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid input")
	}
	community, err := c.service.UpdateProfile(
		ctx.Context(),
		ctx.Params("community_id"),
		input,
	)
	if err != nil {
		status := fiber.StatusInternalServerError
		if errors.Is(err, services.ErrInvalidCommunity) {
			status = fiber.StatusBadRequest
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = fiber.StatusNotFound
		}
		return httpapi.Error(ctx, status, "COMMUNITY_UPDATE_FAILED", err.Error())
	}
	return httpapi.Data(ctx, fiber.StatusOK, community)
}

func (c *CommunityController) UpdateMemberRole(ctx *fiber.Ctx) error {
	var input struct {
		Role string `json:"role"`
	}
	if err := ctx.BodyParser(&input); err != nil {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid input")
	}
	userID, _ := currentUserID(ctx)
	actorRole, _ := ctx.Locals("community_role").(string)
	err := c.service.UpdateMemberRole(
		ctx.Context(),
		ctx.Params("community_id"),
		ctx.Params("member_id"),
		input.Role,
		userID,
		actorRole,
	)
	if err != nil {
		return memberManagementError(ctx, err)
	}
	return httpapi.Data(ctx, fiber.StatusOK, fiber.Map{"role": input.Role})
}

func (c *CommunityController) RemoveMember(ctx *fiber.Ctx) error {
	userID, _ := currentUserID(ctx)
	actorRole, _ := ctx.Locals("community_role").(string)
	err := c.service.RemoveMember(
		ctx.Context(),
		ctx.Params("community_id"),
		ctx.Params("member_id"),
		userID,
		actorRole,
	)
	if err != nil {
		return memberManagementError(ctx, err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (c *CommunityController) Invite(ctx *fiber.Ctx) error {
	userID, ok := currentUserID(ctx)
	if !ok {
		return httpapi.Error(ctx, fiber.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required")
	}

	var body struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "INVALID_INPUT", "Invalid input")
	}

	result, err := c.service.Invite(
		ctx.Context(),
		ctx.Params("community_id"),
		body.Email,
		body.Role,
		userID,
	)
	if err != nil {
		status := fiber.StatusInternalServerError
		if errors.Is(err, services.ErrInvalidCommunityRole) {
			status = fiber.StatusBadRequest
		}
		return httpapi.Error(ctx, status, "INVITATION_CREATE_FAILED", err.Error())
	}

	return httpapi.Data(ctx, fiber.StatusCreated, result)
}

func (c *CommunityController) AcceptInvitation(ctx *fiber.Ctx) error {
	userID, ok := currentUserID(ctx)
	if !ok {
		return httpapi.Error(ctx, fiber.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required")
	}
	member, err := c.service.AcceptInvitation(
		ctx.Context(),
		ctx.Params("token"),
		userID,
	)
	if err != nil {
		status := fiber.StatusInternalServerError
		switch {
		case errors.Is(err, services.ErrInvitationEmail):
			status = fiber.StatusForbidden
		case errors.Is(err, repositories.ErrInvitationUnavailable),
			errors.Is(err, repositories.ErrAlreadyCommunityMember),
			errors.Is(err, gorm.ErrRecordNotFound):
			status = fiber.StatusConflict
		}
		return httpapi.Error(ctx, status, "INVITATION_ACCEPT_FAILED", err.Error())
	}

	return httpapi.Data(ctx, fiber.StatusOK, member)
}

func currentUserID(ctx *fiber.Ctx) (string, bool) {
	userID, ok := ctx.Locals("user_id").(string)
	return userID, ok && userID != ""
}

func memberManagementError(ctx *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, repositories.ErrCommunityMemberNotFound):
		return httpapi.Error(ctx, fiber.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found")
	case errors.Is(err, services.ErrInvalidCommunityRole):
		return httpapi.Error(ctx, fiber.StatusBadRequest, "INVALID_MEMBER_ROLE", err.Error())
	case errors.Is(err, services.ErrOwnerImmutable),
		errors.Is(err, services.ErrMemberRoleEscalation),
		errors.Is(err, services.ErrCannotManageSelf):
		return httpapi.Error(ctx, fiber.StatusForbidden, "MEMBER_CHANGE_FORBIDDEN", err.Error())
	default:
		return httpapi.Error(ctx, fiber.StatusInternalServerError, "MEMBER_UPDATE_FAILED", "Unable to update member")
	}
}
