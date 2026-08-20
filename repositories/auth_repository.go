package repositories

import (
	"context"
	"errors"
	"time"

	"go-ticketing/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrAuthTokenUnavailable = errors.New("auth token is invalid, expired, or already used")

type AuthRepository interface {
	CreateEmailIdentity(ctx context.Context, userID, email string) error
	IsEmailVerified(ctx context.Context, userID string) (bool, error)
	CreateSession(ctx context.Context, session *models.AuthSession) error
	RotateSession(
		ctx context.Context,
		tokenHash string,
		replacement *models.AuthSession,
		now time.Time,
	) (*models.User, error)
	RevokeSession(ctx context.Context, tokenHash string, now time.Time) error
	RevokeAllSessions(ctx context.Context, userID string, now time.Time) error
	ListActiveSessions(ctx context.Context, userID string, now time.Time) ([]models.AuthSession, error)
	RevokeSessionByID(ctx context.Context, userID, sessionID string, now time.Time) error
	ReplaceAuthToken(ctx context.Context, token *models.AuthToken) error
	VerifyEmail(ctx context.Context, tokenHash string, now time.Time) error
	ResetPassword(
		ctx context.Context,
		tokenHash string,
		passwordHash string,
		now time.Time,
	) error
}

type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db: db}
}

func (repository *authRepository) CreateEmailIdentity(
	ctx context.Context,
	userID string,
	email string,
) error {
	identity := &models.UserEmailVerification{UserID: userID, Email: email}
	return repository.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(identity).Error
}

func (repository *authRepository) IsEmailVerified(
	ctx context.Context,
	userID string,
) (bool, error) {
	var identity models.UserEmailVerification
	err := repository.db.WithContext(ctx).First(&identity, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return identity.VerifiedAt != nil, nil
}

func (repository *authRepository) CreateSession(
	ctx context.Context,
	session *models.AuthSession,
) error {
	return repository.db.WithContext(ctx).Create(session).Error
}

func (repository *authRepository) RotateSession(
	ctx context.Context,
	tokenHash string,
	replacement *models.AuthSession,
	now time.Time,
) (*models.User, error) {
	var user models.User
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current models.AuthSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, now).
			First(&current).Error; err != nil {
			return ErrAuthTokenUnavailable
		}

		replacement.UserID = current.UserID
		if err := tx.Create(replacement).Error; err != nil {
			return err
		}
		result := tx.Model(&models.AuthSession{}).
			Where("id = ? AND revoked_at IS NULL", current.ID).
			Updates(map[string]interface{}{
				"revoked_at":     now,
				"last_used_at":   now,
				"replaced_by_id": replacement.ID,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrAuthTokenUnavailable
		}
		return tx.First(&user, "id = ?", current.UserID).Error
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (repository *authRepository) RevokeSession(
	ctx context.Context,
	tokenHash string,
	now time.Time,
) error {
	return repository.db.WithContext(ctx).
		Model(&models.AuthSession{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		Updates(map[string]interface{}{"revoked_at": now, "last_used_at": now}).Error
}

func (repository *authRepository) RevokeAllSessions(
	ctx context.Context,
	userID string,
	now time.Time,
) error {
	return repository.db.WithContext(ctx).
		Model(&models.AuthSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

func (repository *authRepository) ListActiveSessions(
	ctx context.Context,
	userID string,
	now time.Time,
) ([]models.AuthSession, error) {
	var sessions []models.AuthSession
	err := repository.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, now).
		Order("last_used_at DESC").
		Find(&sessions).Error
	return sessions, err
}

func (repository *authRepository) RevokeSessionByID(
	ctx context.Context,
	userID string,
	sessionID string,
	now time.Time,
) error {
	result := repository.db.WithContext(ctx).
		Model(&models.AuthSession{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", sessionID, userID).
		Update("revoked_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrAuthTokenUnavailable
	}
	return nil
}

func (repository *authRepository) ReplaceAuthToken(
	ctx context.Context,
	token *models.AuthToken,
) error {
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.AuthToken{}).
			Where("user_id = ? AND purpose = ? AND used_at IS NULL", token.UserID, token.Purpose).
			Update("used_at", time.Now().UTC()).Error; err != nil {
			return err
		}
		return tx.Create(token).Error
	})
}

func (repository *authRepository) VerifyEmail(
	ctx context.Context,
	tokenHash string,
	now time.Time,
) error {
	return repository.consumeToken(ctx, tokenHash, models.AuthTokenPurposeEmailVerification, now,
		func(tx *gorm.DB, token *models.AuthToken) error {
			result := tx.Model(&models.UserEmailVerification{}).
				Where("user_id = ?", token.UserID).
				Update("verified_at", now)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrAuthTokenUnavailable
			}
			return nil
		})
}

func (repository *authRepository) ResetPassword(
	ctx context.Context,
	tokenHash string,
	passwordHash string,
	now time.Time,
) error {
	return repository.consumeToken(ctx, tokenHash, models.AuthTokenPurposePasswordReset, now,
		func(tx *gorm.DB, token *models.AuthToken) error {
			if err := tx.Model(&models.User{}).
				Where("id = ?", token.UserID).
				Update("password", passwordHash).Error; err != nil {
				return err
			}
			return tx.Model(&models.AuthSession{}).
				Where("user_id = ? AND revoked_at IS NULL", token.UserID).
				Update("revoked_at", now).Error
		})
}

func (repository *authRepository) consumeToken(
	ctx context.Context,
	tokenHash string,
	purpose string,
	now time.Time,
	apply func(*gorm.DB, *models.AuthToken) error,
) error {
	return repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var token models.AuthToken
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(
				"token_hash = ? AND purpose = ? AND used_at IS NULL AND expires_at > ?",
				tokenHash,
				purpose,
				now,
			).
			First(&token).Error
		if err != nil {
			return ErrAuthTokenUnavailable
		}
		if err := apply(tx, &token); err != nil {
			return err
		}
		result := tx.Model(&models.AuthToken{}).
			Where("id = ? AND used_at IS NULL", token.ID).
			Update("used_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrAuthTokenUnavailable
		}
		return nil
	})
}
