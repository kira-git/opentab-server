package routes

import (
	"fmt"
	"net/http"
	"time"

	"opentab-server/internal/middleware"
	"opentab-server/internal/models"
	"opentab-server/internal/response"
	"opentab-server/internal/services"

	"github.com/gin-gonic/gin"
)

func (h *Handler) streamOnCall(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	for index, event := range h.oncall.MockReplyEvents(c.Query("message")) {
		if index > 0 && h.sseDelay > 0 {
			time.Sleep(h.sseDelay)
		}
		writeEvent(c, event)
	}
}

func (h *Handler) createOnCallSession(c *gin.Context) {
	user := middleware.CurrentUser(c)

	var req models.CreateOnCallSessionRequest
	_ = c.ShouldBindJSON(&req)

	resp, appErr := h.oncall.CreateSession(user, req.Title)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) listOnCallSessions(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.oncall.ListSessions(user)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) addOnCallMessage(c *gin.Context) {
	user := middleware.CurrentUser(c)

	var req models.OnCallMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "消息请求格式不正确")
		return
	}

	resp, appErr := h.oncall.AddUserMessage(user, c.Param("sessionId"), req)
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) listOnCallMessages(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.oncall.ListMessages(user, c.Param("sessionId"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) streamOnCallSession(c *gin.Context) {
	user := middleware.CurrentUser(c)
	events, appErr := h.oncall.StreamSessionReply(user, c.Param("sessionId"), c.Query("messageId"))
	if appErr != nil {
		response.Error(c, appErr.Status, appErr.Code, appErr.Message)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	for index, event := range events {
		if index > 0 && h.sseDelay > 0 {
			time.Sleep(h.sseDelay)
		}
		writeEvent(c, event)
	}
}

func (h *Handler) deleteOnCallSession(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.oncall.DeleteSession(user, c.Param("sessionId"))
	writeServiceResult(c, resp, appErr)
}

func writeEvent(c *gin.Context, event services.OnCallEvent) {
	fmt.Fprintf(c.Writer, "event: %s\n", event.Event)
	fmt.Fprintf(c.Writer, "data: %s\n\n", event.Data)
	c.Writer.Flush()
}

func writeServiceResult(c *gin.Context, body any, appErr *services.AppError) {
	if appErr != nil {
		response.Error(c, appErr.Status, appErr.Code, appErr.Message)
		return
	}
	response.OK(c, http.StatusOK, body)
}
