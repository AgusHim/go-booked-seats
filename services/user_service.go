package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"time"

	"go-ticketing/models"
	"go-ticketing/repositories"
	"go-ticketing/utils"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	accessTokenTTL       = 15 * time.Minute
	refreshSessionTTL    = 30 * 24 * time.Hour
	emailVerificationTTL = 24 * time.Hour
	passwordResetTTL     = time.Hour
)

type SessionMetadata struct {
	UserAgent string
	IPAddress string
}

type AuthResult struct {
	User             *models.User
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

type UserService interface {
	GetAll() ([]models.User, error)
	GetByID(id string) (*models.User, error)
	Create(user *models.User) error
	Update(id string, user *models.User) error
	Delete(id string) error
	Register(ctx context.Context, user *models.User) (string, error)
	Login(ctx context.Context, email, password string, metadata SessionMetadata) (*AuthResult, error)
	Refresh(ctx context.Context, refreshToken string, metadata SessionMetadata) (*AuthResult, error)
	Logout(ctx context.Context, refreshToken string) error
	RequestEmailVerification(ctx context.Context, email string) (string, error)
	VerifyEmail(ctx context.Context, token string) error
	RequestPasswordReset(ctx context.Context, email string) (string, error)
	ResetPassword(ctx context.Context, token, password string) error
	ListSessions(ctx context.Context, userID string) ([]models.AuthSession, error)
	RevokeSession(ctx context.Context, userID, sessionID string) error
}

type userService struct {
	userRepo repositories.UserRepository
	authRepo repositories.AuthRepository
	now      func() time.Time
}

func NewUserService(
	userRepo repositories.UserRepository,
	authRepo repositories.AuthRepository,
) UserService {
	return &userService{
		userRepo: userRepo,
		authRepo: authRepo,
		now:      time.Now,
	}
}

func (service *userService) Register(
	ctx context.Context,
	user *models.User,
) (string, error) {
	user.Name = strings.TrimSpace(user.Name)
	email, err := normalizeEmail(user.Email)
	if err != nil {
		return "", err
	}
	if user.Name == "" {
		return "", errors.New("name is required")
	}
	if err := validatePassword(user.Password); err != nil {
		return "", err
	}

	user.Email = email
	user.Role = "user"
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	user.Password = string(hash)
	if err := service.userRepo.Create(user); err != nil {
		return "", err
	}
	if err := service.authRepo.CreateEmailIdentity(ctx, user.ID, user.Email); err != nil {
		_ = service.userRepo.Delete(user.ID)
		return "", err
	}

	token, err := service.issueAuthToken(
		ctx,
		user.ID,
		models.AuthTokenPurposeEmailVerification,
		emailVerificationTTL,
	)
	if err != nil {
		return "", err
	}
	user.Password = ""
	return token, nil
}

func (service *userService) Login(
	ctx context.Context,
	email string,
	password string,
	metadata SessionMetadata,
) (*AuthResult, error) {
	user, err := service.userRepo.FindByEmail(strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, err
	}
	if strings.EqualFold(os.Getenv("REQUIRE_VERIFIED_EMAIL"), "true") {
		verified, err := service.authRepo.IsEmailVerified(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		if !verified {
			return nil, errors.New("email verification required")
		}
	}

	return service.createSession(ctx, user, metadata)
}

func (service *userService) Refresh(
	ctx context.Context,
	refreshToken string,
	metadata SessionMetadata,
) (*AuthResult, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, repositories.ErrAuthTokenUnavailable
	}

	rawReplacement, err := generateOpaqueToken()
	if err != nil {
		return nil, err
	}
	now := service.now().UTC()
	replacement := &models.AuthSession{
		ID:         uuid.NewString(),
		TokenHash:  hashOpaqueToken(rawReplacement),
		UserAgent:  truncate(metadata.UserAgent, 512),
		IPAddress:  truncate(metadata.IPAddress, 64),
		ExpiresAt:  now.Add(refreshSessionTTL),
		LastUsedAt: now,
	}
	user, err := service.authRepo.RotateSession(
		ctx,
		hashOpaqueToken(refreshToken),
		replacement,
		now,
	)
	if err != nil {
		return nil, err
	}
	user.Password = ""
	return service.buildAuthResult(user, replacement, rawReplacement, now)
}

func (service *userService) Logout(ctx context.Context, refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return nil
	}
	return service.authRepo.RevokeSession(
		ctx,
		hashOpaqueToken(refreshToken),
		service.now().UTC(),
	)
}

func (service *userService) RequestEmailVerification(
	ctx context.Context,
	email string,
) (string, error) {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return "", nil
	}
	user, err := service.userRepo.FindByEmail(normalized)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return service.issueAuthToken(
		ctx,
		user.ID,
		models.AuthTokenPurposeEmailVerification,
		emailVerificationTTL,
	)
}

func (service *userService) VerifyEmail(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return repositories.ErrAuthTokenUnavailable
	}
	return service.authRepo.VerifyEmail(
		ctx,
		hashOpaqueToken(token),
		service.now().UTC(),
	)
}

func (service *userService) RequestPasswordReset(
	ctx context.Context,
	email string,
) (string, error) {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return "", nil
	}
	user, err := service.userRepo.FindByEmail(normalized)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return service.issueAuthToken(
		ctx,
		user.ID,
		models.AuthTokenPurposePasswordReset,
		passwordResetTTL,
	)
}

func (service *userService) ResetPassword(
	ctx context.Context,
	token string,
	password string,
) error {
	if err := validatePassword(password); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return service.authRepo.ResetPassword(
		ctx,
		hashOpaqueToken(token),
		string(hash),
		service.now().UTC(),
	)
}

func (service *userService) ListSessions(
	ctx context.Context,
	userID string,
) ([]models.AuthSession, error) {
	return service.authRepo.ListActiveSessions(ctx, userID, service.now().UTC())
}

func (service *userService) RevokeSession(
	ctx context.Context,
	userID string,
	sessionID string,
) error {
	if strings.TrimSpace(sessionID) == "" {
		return repositories.ErrAuthTokenUnavailable
	}
	return service.authRepo.RevokeSessionByID(
		ctx,
		userID,
		sessionID,
		service.now().UTC(),
	)
}

func (service *userService) createSession(
	ctx context.Context,
	user *models.User,
	metadata SessionMetadata,
) (*AuthResult, error) {
	refreshToken, err := generateOpaqueToken()
	if err != nil {
		return nil, err
	}
	now := service.now().UTC()
	session := &models.AuthSession{
		ID:         uuid.NewString(),
		UserID:     user.ID,
		TokenHash:  hashOpaqueToken(refreshToken),
		UserAgent:  truncate(metadata.UserAgent, 512),
		IPAddress:  truncate(metadata.IPAddress, 64),
		ExpiresAt:  now.Add(refreshSessionTTL),
		LastUsedAt: now,
	}
	if err := service.authRepo.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	user.Password = ""
	result, err := service.buildAuthResult(user, session, refreshToken, now)
	if err != nil {
		_ = service.authRepo.RevokeSession(ctx, session.TokenHash, now)
		return nil, err
	}
	return result, nil
}

func (service *userService) buildAuthResult(
	user *models.User,
	session *models.AuthSession,
	refreshToken string,
	now time.Time,
) (*AuthResult, error) {
	accessToken, err := utils.GenerateAccessJWT(user.ID, session.ID, now, accessTokenTTL)
	if err != nil {
		return nil, err
	}
	return &AuthResult{
		User:             user,
		AccessToken:      accessToken,
		AccessExpiresAt:  now.Add(accessTokenTTL),
		RefreshToken:     refreshToken,
		RefreshExpiresAt: session.ExpiresAt,
	}, nil
}

func (service *userService) issueAuthToken(
	ctx context.Context,
	userID string,
	purpose string,
	ttl time.Duration,
) (string, error) {
	rawToken, err := generateOpaqueToken()
	if err != nil {
		return "", err
	}
	now := service.now().UTC()
	token := &models.AuthToken{
		UserID:    userID,
		Purpose:   purpose,
		TokenHash: hashOpaqueToken(rawToken),
		ExpiresAt: now.Add(ttl),
	}
	if err := service.authRepo.ReplaceAuthToken(ctx, token); err != nil {
		return "", err
	}
	return rawToken, nil
}

func generateOpaqueToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func hashOpaqueToken(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must contain at least 8 characters")
	}
	if len(password) > 128 {
		return errors.New("password must not exceed 128 characters")
	}
	return nil
}

func truncate(value string, maximum int) string {
	value = strings.TrimSpace(value)
	if len(value) <= maximum {
		return value
	}
	return value[:maximum]
}

func (service *userService) GetAll() ([]models.User, error) {
	return service.userRepo.FindAll()
}

func (service *userService) GetByID(id string) (*models.User, error) {
	return service.userRepo.FindByID(id)
}

func (service *userService) Create(user *models.User) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.ID = uuid.NewString()
	user.Password = string(hash)
	user.Role = "user"
	return service.userRepo.Create(user)
}

func (service *userService) Update(id string, user *models.User) error {
	existing, err := service.userRepo.FindByID(id)
	if err != nil {
		return err
	}
	existing.Name = user.Name
	existing.Email = user.Email
	existing.Role = user.Role
	return service.userRepo.Update(existing)
}

func (service *userService) Delete(id string) error {
	return service.userRepo.Delete(id)
}
