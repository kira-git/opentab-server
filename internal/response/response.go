package response

import (
	"opentab-server/internal/models"

	"github.com/gin-gonic/gin"
)

func Error(c *gin.Context, status int, code string, message string) {
	c.Set("errorCode", code)
	c.JSON(status, models.ErrorResponse{
		Code:    code,
		Message: message,
		TraceID: requestID(c),
	})
}

func OK(c *gin.Context, status int, body any) {
	c.JSON(status, body)
}

func requestID(c *gin.Context) string {
	value, exists := c.Get("requestId")
	if !exists {
		return ""
	}
	requestID, ok := value.(string)
	if !ok {
		return ""
	}
	return requestID
}
