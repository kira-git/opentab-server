package routes

import (
	"net/http"

	"opentab-server/internal/middleware"
	"opentab-server/internal/models"
	"opentab-server/internal/response"

	"github.com/gin-gonic/gin"
)

func (h *Handler) listAdminTeams(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.ListTeams(user)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) createAdminTeam(c *gin.Context) {
	user := middleware.CurrentUser(c)
	var req models.TeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "创建团队请求格式不正确")
		return
	}
	resp, appErr := h.business.CreateTeam(user, req)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) updateAdminTeam(c *gin.Context) {
	user := middleware.CurrentUser(c)
	var req models.TeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "编辑团队请求格式不正确")
		return
	}
	resp, appErr := h.business.UpdateTeam(user, c.Param("teamId"), req)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) deleteAdminTeam(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.DisableTeam(user, c.Param("teamId"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) listAdminTeamMembers(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.ListTeamMembers(user, c.Param("teamId"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) addAdminTeamMember(c *gin.Context) {
	user := middleware.CurrentUser(c)
	var req models.TeamMemberMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "添加团队成员请求格式不正确")
		return
	}
	resp, appErr := h.business.AddTeamMember(user, c.Param("teamId"), req)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) updateAdminTeamMember(c *gin.Context) {
	user := middleware.CurrentUser(c)
	var req models.TeamMemberMutationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "修改团队成员请求格式不正确")
		return
	}
	resp, appErr := h.business.UpdateTeamMember(user, c.Param("teamId"), c.Param("userId"), req)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) deleteAdminTeamMember(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.RemoveTeamMember(user, c.Param("teamId"), c.Param("userId"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) listAdminUsers(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.ListAdminUsers(user, c.Query("teamId"), c.Query("keyword"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) getAdminUser(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.GetAdminUser(user, c.Param("userId"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) updateAdminUserGlobalRole(c *gin.Context) {
	user := middleware.CurrentUser(c)
	var req models.GlobalRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "修改全局角色请求格式不正确")
		return
	}
	resp, appErr := h.business.UpdateUserGlobalRole(user, c.Param("userId"), req)
	writeServiceResult(c, resp, appErr)
}
