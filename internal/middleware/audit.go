package middleware

import (
	"log"
	"strconv"
	"strings"
	"time"

	"opentab-server/internal/models"
	"opentab-server/internal/repositories"

	"github.com/gin-gonic/gin"
)

type AuditRecorder interface {
	Record(log models.AuditLog) error
}

func Audit(recorder AuditRecorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		action := auditAction(c.Request.Method, c.FullPath(), c.Request.URL.Path)
		if action == "" {
			return
		}

		user := CurrentUser(c)
		userID := ""
		account := ""
		if user != nil {
			userID = user.ID
			account = user.Account
		}

		statusCode := c.Writer.Status()
		result := "success"
		if statusCode >= 400 {
			result = "failure"
		}

		entry := models.AuditLog{
			ID:         "audit-" + randomID(12),
			RequestID:  RequestIDFromContext(c),
			UserID:     userID,
			Account:    account,
			Action:     action,
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			StatusCode: statusCode,
			Result:     result,
			ErrorCode:  auditErrorCode(c),
			ClientIP:   c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			DurationMS: time.Since(start).Milliseconds(),
			CreatedAt:  time.Now().Format(time.RFC3339),
		}
		if err := recorder.Record(entry); err != nil {
			log.Printf("audit log record failed requestId=%s action=%s: %v", entry.RequestID, entry.Action, err)
		}
	}
}

func SetAuditUser(c *gin.Context, userID string, account string) {
	c.Set(currentUserKey, &models.User{ID: userID, Account: account})
}

func auditErrorCode(c *gin.Context) string {
	value := c.GetString("errorCode")
	if value != "" {
		return value
	}
	statusCode := c.Writer.Status()
	if statusCode < 400 {
		return ""
	}
	return "HTTP_" + strconv.Itoa(statusCode)
}

func auditAction(method string, routePath string, rawPath string) string {
	path := routePath
	if path == "" {
		path = rawPath
	}
	if path == "/health" {
		return ""
	}
	if method == "POST" && path == "/auth/login" {
		return "auth.login"
	}
	if method == "POST" && path == "/auth/register" {
		return "auth.register"
	}
	if method == "POST" && path == "/auth/logout" {
		return "auth.logout"
	}
	if strings.HasPrefix(path, "/tabs") || strings.HasPrefix(path, "/me/tabs") {
		return "tab." + strings.ToLower(method)
	}
	if strings.HasPrefix(path, "/business/approval") {
		return "approval." + strings.ToLower(method)
	}
	if strings.HasPrefix(path, "/business/calendar") {
		return "calendar." + strings.ToLower(method)
	}
	if strings.HasPrefix(path, "/business/announcements") {
		return "announcement." + strings.ToLower(method)
	}
	if strings.HasPrefix(path, "/oncall") || path == "/api/chat/stream" {
		return "oncall." + strings.ToLower(method)
	}
	if strings.HasPrefix(path, "/admin") {
		return "admin." + strings.ToLower(method)
	}
	return ""
}

var _ AuditRecorder = repositories.AuditRepository(nil)
