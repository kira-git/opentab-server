package routes

import (
	"net/http"

	"opentab-server/internal/middleware"
	"opentab-server/internal/models"
	"opentab-server/internal/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "登录请求格式不正确")
		return
	}

	resp, appErr := h.auth.Login(req.Account, req.Password)
	if appErr != nil {
		response.Error(c, appErr.Status, appErr.Code, appErr.Message)
		return
	}

	middleware.SetAuditUser(c, resp.UserID, req.Account)
	response.OK(c, http.StatusOK, resp)
}

func (h *Handler) register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "注册请求格式不正确")
		return
	}

	resp, appErr := h.auth.Register(req)
	if appErr != nil {
		response.Error(c, appErr.Status, appErr.Code, appErr.Message)
		return
	}

	middleware.SetAuditUser(c, resp.UserID, req.Account)
	response.OK(c, http.StatusOK, resp)
}

func (h *Handler) logout(c *gin.Context) {
	response.OK(c, http.StatusOK, h.auth.Logout(middleware.CurrentToken(c)))
}

func (h *Handler) me(c *gin.Context) {
	user := middleware.CurrentUser(c)
	response.OK(c, http.StatusOK, h.auth.GetCurrentUser(user))
}
