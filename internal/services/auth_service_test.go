package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"opentab-server/internal/cache"
	"opentab-server/internal/models"
	"opentab-server/internal/repositories"
	"opentab-server/internal/security"
)

type fakeUserRepository struct {
	user                *models.User
	createdUser         *models.User
	updatedPasswordHash string
	sessionToken        string
	findByIDCalls       int
}

func (r *fakeUserRepository) FindByAccount(account string) (*models.User, error) {
	if r.user == nil || r.user.Account != account {
		return nil, repositories.ErrNotFound
	}
	return r.user, nil
}

func (r *fakeUserRepository) FindByID(userID string) (*models.User, error) {
	r.findByIDCalls++
	if r.user != nil && r.user.ID == userID {
		return r.user, nil
	}
	if r.createdUser != nil && r.createdUser.ID == userID {
		return r.createdUser, nil
	}
	return nil, repositories.ErrNotFound
}

func (r *fakeUserRepository) FindSessionByToken(token string) (*models.AuthSession, error) {
	if r.sessionToken == token && r.user != nil {
		expiresAt := time.Now().Add(time.Hour)
		return &models.AuthSession{Token: token, UserID: r.user.ID, ExpiresAt: &expiresAt}, nil
	}
	return nil, repositories.ErrNotFound
}

func (r *fakeUserRepository) FindByToken(token string) (*models.User, error) {
	return nil, repositories.ErrNotFound
}

func (r *fakeUserRepository) Create(user models.User, enabledTabIDs []string) error {
	if r.user != nil && r.user.Account == user.Account {
		return repositories.ErrConflict
	}
	r.createdUser = &user
	return nil
}

func (r *fakeUserRepository) UpdatePasswordHash(userID string, passwordHash string) error {
	if r.user == nil || r.user.ID != userID {
		return repositories.ErrNotFound
	}
	r.updatedPasswordHash = passwordHash
	r.user.Password = passwordHash
	return nil
}

func (r *fakeUserRepository) CreateSession(userID string, token string, expiresAt time.Time) error {
	r.sessionToken = token
	return nil
}

func (r *fakeUserRepository) RevokeToken(token string) error {
	return nil
}

func TestLoginAcceptsPlainPasswordAndUpgradesToHash(t *testing.T) {
	repo := &fakeUserRepository{user: &models.User{
		ID:          "user-1",
		Account:     "demo",
		DisplayName: "Demo",
		Password:    "demo123",
		Enabled:     true,
	}}
	service := NewAuthService(repo)

	resp, err := service.Login("demo", "demo123")
	if err != nil {
		t.Fatalf("login failed: %+v", err)
	}
	if resp.Token == "" || repo.sessionToken == "" {
		t.Fatalf("expected token to be created")
	}
	if !security.IsBcryptHash(repo.updatedPasswordHash) {
		t.Fatalf("expected plain password to be upgraded to bcrypt hash, got %q", repo.updatedPasswordHash)
	}
	if !security.VerifyPassword(repo.updatedPasswordHash, "demo123") {
		t.Fatalf("upgraded bcrypt hash does not verify original password")
	}
}

func TestRegisterStoresBcryptPassword(t *testing.T) {
	repo := &fakeUserRepository{}
	service := NewAuthService(repo)

	_, err := service.Register(models.RegisterRequest{Account: "new-user", Password: "newpass123", DisplayName: "New User"})
	if err != nil {
		t.Fatalf("register failed: %+v", err)
	}
	if repo.createdUser == nil {
		t.Fatalf("expected user to be created")
	}
	if !security.IsBcryptHash(repo.createdUser.Password) {
		t.Fatalf("expected registered password to be bcrypt hash, got %q", repo.createdUser.Password)
	}
	if !security.VerifyPassword(repo.createdUser.Password, "newpass123") {
		t.Fatalf("stored bcrypt hash does not verify original password")
	}
}

func TestLoginRejectsDisabledUser(t *testing.T) {
	repo := &disabledUserRepository{}
	service := NewAuthService(repo)

	_, err := service.Login("disabled", "password")
	if err == nil {
		t.Fatalf("expected disabled user login to fail")
	}
	if err.Code != "USER_DISABLED" {
		t.Fatalf("expected USER_DISABLED, got %s", err.Code)
	}
}

type disabledUserRepository struct {
	fakeUserRepository
}

func (r *disabledUserRepository) FindByAccount(account string) (*models.User, error) {
	return nil, repositories.ErrUserDisabled
}

func TestFindUserByTokenReturnsRepositoryError(t *testing.T) {
	repo := &tokenErrorRepository{}
	service := NewAuthService(repo)

	_, err := service.FindUserByToken("expired-token")
	if !errors.Is(err, repositories.ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

type tokenErrorRepository struct {
	fakeUserRepository
}

func (r *tokenErrorRepository) FindSessionByToken(token string) (*models.AuthSession, error) {
	return nil, repositories.ErrTokenExpired
}

func TestFindUserByTokenCachesSessionAndUserContext(t *testing.T) {
	repo := &fakeUserRepository{user: &models.User{
		ID:          "user-1",
		Account:     "demo",
		DisplayName: "Demo",
		Password:    "hash",
		Permissions: []string{"ai.oncall"},
		Enabled:     true,
	}}
	token := "token-cache-test"
	expiresAt := time.Now().Add(time.Hour)
	if err := repo.CreateSession(repo.user.ID, token, expiresAt); err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	authCache := newMemoryAuthCache()
	service := NewAuthServiceWithCache(repo, authCache, time.Minute)

	first, err := service.FindUserByToken(token)
	if err != nil {
		t.Fatalf("first auth failed: %v", err)
	}
	second, err := service.FindUserByToken(token)
	if err != nil {
		t.Fatalf("second auth failed: %v", err)
	}
	if first.ID != repo.user.ID || second.ID != repo.user.ID {
		t.Fatalf("unexpected cached user")
	}
	if repo.findByIDCalls != 1 {
		t.Fatalf("expected user context to be loaded from repository once, got %d", repo.findByIDCalls)
	}
}

type memoryAuthCache struct {
	sessions map[string]cache.AuthSession
	users    map[string]models.User
}

func newMemoryAuthCache() *memoryAuthCache {
	return &memoryAuthCache{
		sessions: map[string]cache.AuthSession{},
		users:    map[string]models.User{},
	}
}

func (c *memoryAuthCache) GetSession(_ context.Context, token string) (*cache.AuthSession, error) {
	session, ok := c.sessions[token]
	if !ok {
		return nil, cache.ErrMiss
	}
	return &session, nil
}

func (c *memoryAuthCache) SetSession(_ context.Context, token string, session cache.AuthSession, _ time.Duration) error {
	c.sessions[token] = session
	return nil
}

func (c *memoryAuthCache) DeleteSession(_ context.Context, token string) error {
	delete(c.sessions, token)
	return nil
}

func (c *memoryAuthCache) GetUserContext(_ context.Context, userID string) (*models.User, error) {
	user, ok := c.users[userID]
	if !ok {
		return nil, cache.ErrMiss
	}
	return &user, nil
}

func (c *memoryAuthCache) SetUserContext(_ context.Context, userID string, user models.User, _ time.Duration) error {
	c.users[userID] = user
	return nil
}

func (c *memoryAuthCache) DeleteUserContext(_ context.Context, userID string) error {
	delete(c.users, userID)
	return nil
}
