package routes

import (
	"net/http"

	"opentab-server/internal/middleware"
	"opentab-server/internal/models"
	"opentab-server/internal/response"
	"opentab-server/internal/services"

	"github.com/gin-gonic/gin"
)

func (h *Handler) approvalSummary(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.ApprovalSummary(user)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) listApprovalItems(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.ListApprovalItems(user, c.Query("scope"), c.Query("status"), c.Query("teamId"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) createApprovalItem(c *gin.Context) {
	user := middleware.CurrentUser(c)
	var req models.CreateApprovalItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "创建审批请求格式不正确")
		return
	}
	resp, appErr := h.business.CreateApprovalItem(user, req)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) getApprovalItem(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.GetApprovalItem(user, c.Param("itemId"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) approveItem(c *gin.Context) {
	h.updateApprovalItem(c, true)
}

func (h *Handler) rejectItem(c *gin.Context) {
	h.updateApprovalItem(c, false)
}

func (h *Handler) cancelApprovalItem(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.CancelApprovalItem(user, c.Param("itemId"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) updateApprovalItem(c *gin.Context, approved bool) {
	user := middleware.CurrentUser(c)

	var req models.ApprovalActionRequest
	_ = c.ShouldBindJSON(&req)

	var resp *models.ApprovalActionResponse
	var appErr *services.AppError
	if approved {
		resp, appErr = h.business.ApproveItem(user, c.Param("itemId"), req.Comment)
	} else {
		resp, appErr = h.business.RejectItem(user, c.Param("itemId"), req.Comment)
	}
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) calendarSummary(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.CalendarSummary(user)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) listCalendarEvents(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.ListCalendarEvents(user, c.Query("scope"), c.Query("date"), c.Query("teamId"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) getCalendarEvent(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.GetCalendarEvent(user, c.Param("eventId"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) createCalendarEvent(c *gin.Context) {
	user := middleware.CurrentUser(c)

	var req models.CreateCalendarEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "新增日程请求格式不正确")
		return
	}

	resp, appErr := h.business.CreateCalendarEvent(user, req)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) updateCalendarEvent(c *gin.Context) {
	user := middleware.CurrentUser(c)
	var req models.CreateCalendarEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "编辑日程请求格式不正确")
		return
	}
	resp, appErr := h.business.UpdateCalendarEvent(user, c.Param("eventId"), req)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) deleteCalendarEvent(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.DeleteCalendarEvent(user, c.Param("eventId"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) listAnnouncements(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.ListAnnouncements(user, c.Query("scope"), c.Query("teamId"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) getAnnouncement(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.GetAnnouncement(user, c.Param("announcementId"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) createAnnouncement(c *gin.Context) {
	user := middleware.CurrentUser(c)
	var req models.AnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "发布公告请求格式不正确")
		return
	}
	resp, appErr := h.business.CreateAnnouncement(user, req)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) updateAnnouncement(c *gin.Context) {
	user := middleware.CurrentUser(c)
	var req models.AnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "编辑公告请求格式不正确")
		return
	}
	resp, appErr := h.business.UpdateAnnouncement(user, c.Param("announcementId"), req)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) deleteAnnouncement(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.business.DeleteAnnouncement(user, c.Param("announcementId"))
	writeServiceResult(c, resp, appErr)
}
