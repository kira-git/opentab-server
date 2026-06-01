package middleware

import (
	"net/http"
	"strings"

	"opentab-server/internal/models"
	"opentab-server/internal/response"

	"github.com/gin-gonic/gin"
)

const currentUserKey = "currentUser"
const currentTokenKey = "currentToken"

type UserFinder interface {
	FindUserByToken(token string) (*models.User, bool)
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

		user, ok := userFinder.FindUserByToken(token)
		if !ok {
			response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "Token 无效或已过期")
			c.Abort()
			return
		}

		c.Set(currentUserKey, user)
		c.Set(currentTokenKey, token)
		c.Next()
	}
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
