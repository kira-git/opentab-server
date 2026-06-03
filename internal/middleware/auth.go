package middleware

import (
	"errors"
	"net/http"
	"strings"

	"opentab-server/internal/models"
	"opentab-server/internal/repositories"
	"opentab-server/internal/response"

	"github.com/gin-gonic/gin"
)

const currentUserKey = "currentUser"
const currentTokenKey = "currentToken"

type UserFinder interface {
	FindUserByToken(token string) (*models.User, error)
}

func Auth(userFinder UserFinder) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if token == "" || token == authHeader {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "缺少 Bearer Token")
			c.Abort()
			return
		}

		user, err := userFinder.FindUserByToken(token)
		if err != nil {
			status, code, message := authErrorResponse(err)
			response.Error(c, status, code, message)
			c.Abort()
			return
		}

		c.Set(currentUserKey, user)
		c.Set(currentTokenKey, token)
		c.Next()
	}
}

func authErrorResponse(err error) (int, string, string) {
	if errors.Is(err, repositories.ErrTokenExpired) {
		return http.StatusUnauthorized, "TOKEN_EXPIRED", "Token 已过期"
	}
	if errors.Is(err, repositories.ErrTokenRevoked) {
		return http.StatusUnauthorized, "TOKEN_REVOKED", "Token 已退出登录"
	}
	if errors.Is(err, repositories.ErrUserDisabled) {
		return http.StatusForbidden, "USER_DISABLED", "账号已被禁用"
	}
	return http.StatusUnauthorized, "UNAUTHORIZED", "Token 无效或已过期"
}

func CurrentUser(c *gin.Context) *models.User {
	value, exists := c.Get(currentUserKey)
	if !exists {
		return nil
	}
	user, ok := value.(*models.User)
	if !ok {
		return nil
	}
	return user
}

func CurrentToken(c *gin.Context) string {
	value, exists := c.Get(currentTokenKey)
	if !exists {
		return ""
	}
	token, ok := value.(string)
	if !ok {
		return ""
	}
	return token
}
