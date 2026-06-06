package repositories

import (
	"time"

	"opentab-server/internal/models"
)

type UserRepository interface {
	FindByAccount(account string) (*models.User, error)
	FindByID(userID string) (*models.User, error)
	FindSessionByToken(token string) (*models.AuthSession, error)
	FindByToken(token string) (*models.User, error)
	Create(user models.User, enabledTabIDs []string) error
	UpdatePasswordHash(userID string, passwordHash string) error
	CreateSession(userID string, token string, expiresAt time.Time) error
	RevokeToken(token string) error
}
