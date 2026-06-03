package services

import (
	"errors"
	"testing"
	"time"

	"opentab-server/internal/models"
	"opentab-server/internal/repositories"
	"opentab-server/internal/security"
)

type fakeUserRepository struct {
	user                *models.User
	createdUser         *models.User
	updatedPasswordHash string
	sessionToken        string
}

func (r *fakeUserRepository) FindByAccount(account string) (*models.User, error) {
	if r.user == nil || r.user.Account != account {
		return nil, repositories.ErrNotFound
	}
	return r.user, nil
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

func (r *tokenErrorRepository) FindByToken(token string) (*models.User, error) {
	return nil, repositories.ErrTokenExpired
}
