package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"opentab-server/internal/models"
	"opentab-server/internal/repositories"
)

type AuthService struct {
	users repositories.UserRepository
}

func NewAuthService(users repositories.UserRepository) *AuthService {
	return &AuthService{users: users}
}

func (s *AuthService) Login(account string, password string) (*models.LoginResponse, *AppError) {
	user, err := s.users.FindByAccount(account)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "账号或密码不正确")
	}
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "登录失败")
	}
	if user.Password != password {
		return nil, NewAppError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "账号或密码不正确")
	}

	token := newAccessToken()
	if err := s.users.CreateSession(user.ID, token, tokenExpiresAt()); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "登录失败")
	}

	return &models.LoginResponse{
		Token:       token,
		UserID:      user.ID,
		DisplayName: user.DisplayName,
		Permissions: user.Permissions,
	}, nil
}

func (s *AuthService) Register(req models.RegisterRequest) (*models.LoginResponse, *AppError) {
	account := strings.TrimSpace(req.Account)
	password := strings.TrimSpace(req.Password)
	displayName := strings.TrimSpace(req.DisplayName)
	if account == "" || password == "" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_REQUEST", "账号和密码不可为空")
	}
	if len(account) > 64 {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_REQUEST", "账号长度不能超过 64")
	}
	if len(password) < 6 {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_REQUEST", "密码长度不能少于 6")
	}
	if displayName == "" {
		displayName = account
	}

	if _, err := s.users.FindByAccount(account); err == nil {
		return nil, NewAppError(http.StatusConflict, "ACCOUNT_EXISTS", "账号已存在")
	} else if !errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "注册失败")
	}

	user := models.User{
		ID:          "user-" + randomHex(8),
		Account:     account,
		DisplayName: displayName,
		Password:    password,
		Permissions: defaultRegisterPermissions,
	}
	if err := s.users.Create(user, defaultRegisterTabs); err != nil {
		if errors.Is(err, repositories.ErrConflict) {
			return nil, NewAppError(http.StatusConflict, "ACCOUNT_EXISTS", "账号已存在")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "注册失败")
	}
	token := newAccessToken()
	if err := s.users.CreateSession(user.ID, token, tokenExpiresAt()); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "注册失败")
	}

	return &models.LoginResponse{
		Token:       token,
		UserID:      user.ID,
		DisplayName: user.DisplayName,
		Permissions: user.Permissions,
	}, nil
}

func (s *AuthService) FindUserByToken(token string) (*models.User, bool) {
	user, err := s.users.FindByToken(token)
	return user, err == nil
}

func (s *AuthService) GetCurrentUser(user *models.User) models.MeResponse {
	return models.MeResponse{
		UserID:      user.ID,
		DisplayName: user.DisplayName,
		Permissions: user.Permissions,
		Team: models.Team{
			ID:   "team-demo",
			Name: "演示团队",
		},
	}
}

func (s *AuthService) Logout(token string) models.SuccessResponse {
	_ = s.users.RevokeToken(token)
	return models.SuccessResponse{Success: true}
}

func randomHex(byteCount int) string {
	data := make([]byte, byteCount)
	if _, err := rand.Read(data); err != nil {
		return "fallback"
	}
	return hex.EncodeToString(data)
}

func newAccessToken() string {
	return "token-" + randomHex(32)
}

func tokenExpiresAt() time.Time {
	return time.Now().Add(accessTokenTTL)
}

const accessTokenTTL = 7 * 24 * time.Hour

var defaultRegisterPermissions = []string{
	"tab.approval.read",
	"tab.calendar.read",
	"ai.oncall",
}

var defaultRegisterTabs = []string{
	"approval",
	"calendar",
}
