package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-ticketing/models"
	"go-ticketing/repositories"
	"go-ticketing/utils"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserServiceRefreshRotationAndLogout(t *testing.T) {
	db, service := newAuthTestService(t, "refresh")
	ctx := context.Background()
	user := &models.User{
		Name:     "Owner",
		Email:    "owner@example.test",
		Password: "password-aman",
	}
	if _, err := service.Register(ctx, user); err != nil {
		t.Fatalf("register: %v", err)
	}

	login, err := service.Login(ctx, user.Email, "password-aman", SessionMetadata{
		UserAgent: "test-browser",
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if login.RefreshToken == "" || login.AccessToken == "" {
		t.Fatal("login must issue access and refresh tokens")
	}
	if login.AccessExpiresAt.Sub(time.Now().UTC()) > accessTokenTTL+time.Minute {
		t.Fatal("access token must remain short-lived")
	}

	var first models.AuthSession
	if err := db.First(&first, "user_id = ?", user.ID).Error; err != nil {
		t.Fatalf("load first session: %v", err)
	}
	if first.TokenHash == login.RefreshToken || first.TokenHash != hashOpaqueToken(login.RefreshToken) {
		t.Fatal("database must store only the refresh-token hash")
	}

	refreshed, err := service.Refresh(ctx, login.RefreshToken, SessionMetadata{
		UserAgent: "test-browser-rotated",
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshed.RefreshToken == login.RefreshToken {
		t.Fatal("refresh token must rotate")
	}
	if _, err := service.Refresh(ctx, login.RefreshToken, SessionMetadata{}); !errors.Is(
		err,
		repositories.ErrAuthTokenUnavailable,
	) {
		t.Fatalf("old refresh token must be rejected, got %v", err)
	}

	sessions, err := service.ListSessions(ctx, user.ID)
	if err != nil || len(sessions) != 1 {
		t.Fatalf("expected one rotated active session, sessions=%d err=%v", len(sessions), err)
	}
	if err := service.RevokeSession(ctx, "different-user", sessions[0].ID); !errors.Is(
		err,
		repositories.ErrAuthTokenUnavailable,
	) {
		t.Fatalf("another user must not revoke the session, got %v", err)
	}

	if err := service.Logout(ctx, refreshed.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := service.Refresh(ctx, refreshed.RefreshToken, SessionMetadata{}); !errors.Is(
		err,
		repositories.ErrAuthTokenUnavailable,
	) {
		t.Fatalf("logged-out refresh token must be rejected, got %v", err)
	}
}

func TestUserServiceEmailVerificationIsHashedAndSingleUse(t *testing.T) {
	db, service := newAuthTestService(t, "verify")
	ctx := context.Background()
	user := &models.User{
		Name:     "Member",
		Email:    "member@example.test",
		Password: "password-aman",
	}
	rawToken, err := service.Register(ctx, user)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if rawToken == "" {
		t.Fatal("registration must create a verification token")
	}

	var stored models.AuthToken
	if err := db.First(&stored, "user_id = ? AND purpose = ?", user.ID, models.AuthTokenPurposeEmailVerification).Error; err != nil {
		t.Fatalf("load token: %v", err)
	}
	if stored.TokenHash == rawToken || stored.TokenHash != hashOpaqueToken(rawToken) {
		t.Fatal("database must store only the verification-token hash")
	}

	if err := service.VerifyEmail(ctx, rawToken); err != nil {
		t.Fatalf("verify: %v", err)
	}
	verified, err := repositories.NewAuthRepository(db).IsEmailVerified(ctx, user.ID)
	if err != nil || !verified {
		t.Fatalf("email should be verified, verified=%v err=%v", verified, err)
	}
	if err := service.VerifyEmail(ctx, rawToken); !errors.Is(err, repositories.ErrAuthTokenUnavailable) {
		t.Fatalf("verification token must be single-use, got %v", err)
	}
}

func TestUserServicePasswordResetRevokesSessions(t *testing.T) {
	db, service := newAuthTestService(t, "password-reset")
	ctx := context.Background()
	user := &models.User{
		Name:     "Member",
		Email:    "reset@example.test",
		Password: "password-lama",
	}
	if _, err := service.Register(ctx, user); err != nil {
		t.Fatalf("register: %v", err)
	}
	login, err := service.Login(ctx, user.Email, "password-lama", SessionMetadata{})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	resetToken, err := service.RequestPasswordReset(ctx, user.Email)
	if err != nil {
		t.Fatalf("request reset: %v", err)
	}
	if err := service.ResetPassword(ctx, resetToken, "password-baru"); err != nil {
		t.Fatalf("reset password: %v", err)
	}

	if _, err := service.Refresh(ctx, login.RefreshToken, SessionMetadata{}); !errors.Is(
		err,
		repositories.ErrAuthTokenUnavailable,
	) {
		t.Fatalf("password reset must revoke sessions, got %v", err)
	}
	if _, err := service.Login(ctx, user.Email, "password-lama", SessionMetadata{}); err == nil {
		t.Fatal("old password must no longer authenticate")
	}
	if _, err := service.Login(ctx, user.Email, "password-baru", SessionMetadata{}); err != nil {
		t.Fatalf("new password should authenticate: %v", err)
	}
	if err := service.ResetPassword(ctx, resetToken, "password-lain"); !errors.Is(
		err,
		repositories.ErrAuthTokenUnavailable,
	) {
		t.Fatalf("reset token must be single-use, got %v", err)
	}

	var stored models.User
	if err := db.First(&stored, "id = ?", user.ID).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte("password-baru")) != nil {
		t.Fatal("new password hash was not persisted")
	}
}

func TestAccessTokenCarriesSessionBoundary(t *testing.T) {
	_, service := newAuthTestService(t, "access-claims")
	ctx := context.Background()
	user := &models.User{
		Name:     "Member",
		Email:    "claims@example.test",
		Password: "password-aman",
	}
	if _, err := service.Register(ctx, user); err != nil {
		t.Fatalf("register: %v", err)
	}
	result, err := service.Login(ctx, user.Email, "password-aman", SessionMetadata{})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	claims, err := utils.ParseJWT(result.AccessToken)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if tokenType, _ := utils.ClaimString(claims, "type"); tokenType != "user_session" {
		t.Fatalf("unexpected token type %q", tokenType)
	}
	if sessionID, ok := utils.ClaimString(claims, "sid"); !ok || sessionID == "" {
		t.Fatal("access token must identify its refresh session")
	}
}

func newAuthTestService(t *testing.T, name string) (*gorm.DB, UserService) {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-secret-that-is-at-least-24-characters")
	t.Setenv("REQUIRE_VERIFIED_EMAIL", "false")
	db, err := gorm.Open(
		sqlite.Open("file:"+name+"?mode=memory&cache=shared"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.AuthSession{},
		&models.AuthToken{},
		&models.UserEmailVerification{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	userRepo := repositories.NewUserRepository(db)
	authRepo := repositories.NewAuthRepository(db)
	return db, NewUserService(userRepo, authRepo)
}
