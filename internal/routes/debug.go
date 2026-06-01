package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *Handler) debugStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"serverTime":   time.Now().Format(time.RFC3339),
		"apiVersion":   "1.1-draft",
		"mockMode":     h.status.AppMode != "postgres",
		"sseAvailable": true,
		"tabCount":     h.tabs.Count(),
		"database": gin.H{
			"enabled": h.status.DatabaseEnabled,
			"type":    h.status.DatabaseType,
		},
	})
}

func (h *Handler) debugPermissions(c *gin.Context) {
	c.JSON(http.StatusOK, h.debug.ListPermissions())
}

func (h *Handler) debugSampleTabs(c *gin.Context) {
	resp, appErr := h.tabs.ListAll()
	writeServiceResult(c, resp, appErr)
}
