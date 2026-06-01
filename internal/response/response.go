package response

import (
	"opentab-server/internal/models"

	"github.com/gin-gonic/gin"
)

func Error(c *gin.Context, status int, code string, message string) {
	c.JSON(status, models.ErrorResponse{
		Code:    code,
		Message: message,
	})
}

func OK(c *gin.Context, status int, body any) {
	c.JSON(status, body)
}
