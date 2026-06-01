package routes

import (
	"net/http"

	"opentab-server/internal/middleware"
	"opentab-server/internal/models"
	"opentab-server/internal/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) listTabs(c *gin.Context) {
	user := middleware.CurrentUser(c)
	tabs, appErr := h.tabs.ListUserTabs(user)
	writeServiceResult(c, tabs, appErr)
}

func (h *Handler) listTabCatalog(c *gin.Context) {
	user := middleware.CurrentUser(c)
	tabs, appErr := h.tabs.ListCatalog(user)
	writeServiceResult(c, tabs, appErr)
}

func (h *Handler) getTab(c *gin.Context) {
	user := middleware.CurrentUser(c)
	tab, appErr := h.tabs.GetTab(user, c.Param("tabId"))
	writeServiceResult(c, tab, appErr)
}

func (h *Handler) createCustomTab(c *gin.Context) {
	user := middleware.CurrentUser(c)

	var req models.CreateCustomTabRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "自定义 Tab 请求格式不正确")
		return
	}

	resp, appErr := h.tabs.CreateCustomTab(user, req)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) updateCustomTab(c *gin.Context) {
	user := middleware.CurrentUser(c)

	var req models.UpdateCustomTabRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "自定义 Tab 请求格式不正确")
		return
	}

	resp, appErr := h.tabs.UpdateCustomTab(user, c.Param("tabId"), req)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) deleteCustomTab(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.tabs.DeleteCustomTab(user, c.Param("tabId"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) enableMyTab(c *gin.Context) {
	user := middleware.CurrentUser(c)

	var req models.EnableTabRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "tabId 不可为空")
		return
	}

	resp, appErr := h.tabs.EnableTab(user, req.TabID)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) disableMyTab(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.tabs.DisableTab(user, c.Param("tabId"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) reorderMyTabs(c *gin.Context) {
	user := middleware.CurrentUser(c)

	var req models.ReorderTabsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Tab 排序请求格式不正确")
		return
	}

	resp, appErr := h.tabs.ReorderTabs(user, req)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) validateTab(c *gin.Context) {
	var req models.ValidateTabRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "Tab 校验请求格式不正确")
		return
	}

	response.OK(c, http.StatusOK, h.tabs.ValidateTab(req))
}

func (h *Handler) reportTabAction(c *gin.Context) {
	user := middleware.CurrentUser(c)

	var req models.ActionRequest
	_ = c.ShouldBindJSON(&req)

	resp, appErr := h.tabs.ReportAction(user, c.Param("tabId"), c.Param("actionId"))
	writeServiceResult(c, resp, appErr)
}
