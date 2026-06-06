package cache

import (
	"context"
	"errors"
	"time"

	"opentab-server/internal/models"
)

var ErrMiss = errors.New("cache miss")

type AuthSession struct {
	UserID    string     `json:"userId"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type AuthCache interface {
	GetSession(ctx context.Context, token string) (*AuthSession, error)
	SetSession(ctx context.Context, token string, session AuthSession, ttl time.Duration) error
	DeleteSession(ctx context.Context, token string) error
	GetUserContext(ctx context.Context, userID string) (*models.User, error)
	SetUserContext(ctx context.Context, userID string, user models.User, ttl time.Duration) error
	DeleteUserContext(ctx context.Context, userID string) error
}

type NoopAuthCache struct{}

func NewNoopAuthCache() NoopAuthCache {
	return NoopAuthCache{}
}

func (NoopAuthCache) GetSession(context.Context, string) (*AuthSession, error) {
	return nil, ErrMiss
}

func (NoopAuthCache) SetSession(context.Context, string, AuthSession, time.Duration) error {
	return nil
}

func (NoopAuthCache) DeleteSession(context.Context, string) error {
	return nil
}

func (NoopAuthCache) GetUserContext(context.Context, string) (*models.User, error) {
	return nil, ErrMiss
}

func (NoopAuthCache) SetUserContext(context.Context, string, models.User, time.Duration) error {
	return nil
}

func (NoopAuthCache) DeleteUserContext(context.Context, string) error {
	return nil
}
