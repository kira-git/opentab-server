package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"opentab-server/internal/cache"
	"opentab-server/internal/models"
	"opentab-server/internal/repositories"
	"opentab-server/internal/security"
)

type AuthService struct {
	users          repositories.UserRepository
	authCache      cache.AuthCache
	userContextTTL time.Duration
}

func NewAuthService(users repositories.UserRepository) *AuthService {
	return NewAuthServiceWithCache(users, cache.NewNoopAuthCache(), defaultUserContextCacheTTL)
}

func NewAuthServiceWithCache(users repositories.UserRepository, authCache cache.AuthCache, userContextTTL time.Duration) *AuthService {
	if authCache == nil {
		authCache = cache.NewNoopAuthCache()
	}
	if userContextTTL <= 0 {
		userContextTTL = defaultUserContextCacheTTL
	}
	return &AuthService{users: users, authCache: authCache, userContextTTL: userContextTTL}
}

func (s *AuthService) Login(account string, password string) (*models.LoginResponse, *AppError) {
	user, err := s.users.FindByAccount(account)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "账号或密码不正确")
	}
	if errors.Is(err, repositories.ErrUserDisabled) {
		return nil, NewAppError(http.StatusForbidden, "USER_DISABLED", "账号已被禁用")
	}
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "登录失败")
	}
	if !security.VerifyPassword(user.Password, password) {
		return nil, NewAppError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "账号或密码不正确")
	}
	if !security.IsBcryptHash(user.Password) {
		if passwordHash, hashErr := security.HashPassword(password); hashErr == nil {
			_ = s.users.UpdatePasswordHash(user.ID, passwordHash)
		}
	}

	token := newAccessToken()
	expiresAt := tokenExpiresAt()
	if err := s.users.CreateSession(user.ID, token, expiresAt); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "登录失败")
	}
	s.cacheSessionAndUser(token, user, expiresAt)

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
		Permissions: defaultRegisterPermissions,
	}
	passwordHash, hashErr := security.HashPassword(password)
	if hashErr != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "注册失败")
	}
	user.Password = passwordHash
	if err := s.users.Create(user, defaultRegisterTabs); err != nil {
		if errors.Is(err, repositories.ErrConflict) {
			return nil, NewAppError(http.StatusConflict, "ACCOUNT_EXISTS", "账号已存在")
		}
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "注册失败")
	}
	token := newAccessToken()
	expiresAt := tokenExpiresAt()
	if err := s.users.CreateSession(user.ID, token, expiresAt); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "注册失败")
	}
	if currentUser, findErr := s.users.FindByID(user.ID); findErr == nil {
		s.cacheSessionAndUser(token, currentUser, expiresAt)
	} else {
		s.cacheSession(token, user.ID, expiresAt)
	}

	return &models.LoginResponse{
		Token:       token,
		UserID:      user.ID,
		DisplayName: user.DisplayName,
		Permissions: user.Permissions,
	}, nil
}

func (s *AuthService) FindUserByToken(token string) (*models.User, error) {
	ctx := context.Background()
	session, err := s.authCache.GetSession(ctx, token)
	if err != nil {
		repoSession, repoErr := s.users.FindSessionByToken(token)
		if repoErr != nil {
			return nil, repoErr
		}
		session = &cache.AuthSession{UserID: repoSession.UserID, ExpiresAt: repoSession.ExpiresAt}
		_ = s.authCache.SetSession(ctx, token, *session, sessionTTL(repoSession.ExpiresAt))
	}
	if session.ExpiresAt != nil && session.ExpiresAt.Before(time.Now()) {
		_ = s.authCache.DeleteSession(ctx, token)
		return nil, repositories.ErrTokenExpired
	}

	user, err := s.authCache.GetUserContext(ctx, session.UserID)
	if err != nil {
		user, err = s.users.FindByID(session.UserID)
		if err != nil {
			return nil, err
		}
		_ = s.authCache.SetUserContext(ctx, user.ID, sanitizedUserForCache(*user), s.userContextTTL)
	}
	result := *user
	result.Token = token
	return &result, nil
}

func (s *AuthService) GetCurrentUser(user *models.User) models.MeResponse {
	var globalRole *string
	if user.GlobalRole != "" {
		globalRole = &user.GlobalRole
	}
	var currentTeamID *string
	var team *models.Team
	if user.CurrentTeamID != "" {
		currentTeamID = &user.CurrentTeamID
		for _, membership := range user.Memberships {
			if membership.TeamID == user.CurrentTeamID {
				team = &models.Team{ID: membership.TeamID, Name: membership.TeamName}
				break
			}
		}
	}
	return models.MeResponse{
		UserID:        user.ID,
		DisplayName:   user.DisplayName,
		GlobalRole:    globalRole,
		CurrentTeamID: currentTeamID,
		Memberships:   user.Memberships,
		Permissions:   user.Permissions,
		Team:          team,
	}
}

func (s *AuthService) Logout(token string) models.SuccessResponse {
	_ = s.users.RevokeToken(token)
	_ = s.authCache.DeleteSession(context.Background(), token)
	return models.SuccessResponse{Success: true}
}

func (s *AuthService) cacheSessionAndUser(token string, user *models.User, expiresAt time.Time) {
	if user == nil {
		return
	}
	s.cacheSession(token, user.ID, expiresAt)
	ctx := context.Background()
	_ = s.authCache.SetUserContext(ctx, user.ID, sanitizedUserForCache(*user), s.userContextTTL)
}

func (s *AuthService) cacheSession(token string, userID string, expiresAt time.Time) {
	ctx := context.Background()
	_ = s.authCache.SetSession(ctx, token, cache.AuthSession{UserID: userID, ExpiresAt: &expiresAt}, sessionTTL(&expiresAt))
}

func sanitizedUserForCache(user models.User) models.User {
	user.Password = ""
	user.Token = ""
	return user
}

func sessionTTL(expiresAt *time.Time) time.Duration {
	if expiresAt == nil {
		return accessTokenTTL
	}
	return time.Until(*expiresAt)
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
const defaultUserContextCacheTTL = 5 * time.Minute

var defaultRegisterPermissions = []string{
	"tab.company.read",
	"tab.announcement.read",
	"tab.fun.read",
	"tab.approval.read",
	"tab.approval.create",
	"tab.calendar.read",
	"ai.oncall",
}

var defaultRegisterTabs = []string{
	"company-intro",
	"announcements",
	"fun",
	"approval",
	"calendar",
}
