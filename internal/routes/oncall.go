package routes

import (
	"fmt"
	"net/http"

	"opentab-server/internal/middleware"
	"opentab-server/internal/models"
	"opentab-server/internal/response"
	"opentab-server/internal/services"

	"github.com/gin-gonic/gin"
)

type streamAIChatRequest struct {
	Message        string `json:"message"`
	ConversationID string `json:"conversationId"`
}

func (h *Handler) streamAIChat(c *gin.Context) {
	var req streamAIChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "AI 聊天请求格式不正确")
		return
	}
	if req.Message == "" {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "message 不可为空")
		return
	}

	writeSSEHeaders(c)
	if err := h.oncall.StreamAIChatForUser(c.Request.Context(), middleware.CurrentUser(c), req.Message, req.ConversationID, func(event services.OnCallEvent) error {
		return writeEvent(c, event)
	}); err != nil {
		_ = writeEvent(c, services.OnCallEvent{
			Event: "message",
			Data:  `{"type":"error","code":"AI_SERVICE_ERROR","delta":"AI 服务调用失败"}`,
		})
	}
}

func (h *Handler) streamOnCall(c *gin.Context) {
	writeSSEHeaders(c)
	if err := h.oncall.StreamOnCallQuery(c.Request.Context(), c.Query("message"), func(event services.OnCallEvent) error {
		return writeEvent(c, event)
	}); err != nil {
		_ = writeEvent(c, services.OnCallEvent{
			Event: "error",
			Data:  `{"code":"AI_SERVICE_ERROR","message":"AI 服务调用失败"}`,
		})
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
	writeSSEHeaders(c)
	appErr := h.oncall.StreamOnCallMessage(c.Request.Context(), user, c.Param("sessionId"), c.Query("messageId"), func(event services.OnCallEvent) error {
		return writeEvent(c, event)
	})
	if appErr != nil {
		_ = writeEvent(c, services.OnCallEvent{
			Event: "error",
			Data:  fmt.Sprintf(`{"code":"%s","message":"%s"}`, appErr.Code, appErr.Message),
		})
	}
}

func (h *Handler) cancelOnCallGeneration(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.oncall.CancelSessionGeneration(user, c.Param("sessionId"))
	writeServiceResult(c, resp, appErr)
}

func (h *Handler) deleteOnCallSession(c *gin.Context) {
	user := middleware.CurrentUser(c)
	resp, appErr := h.oncall.DeleteSession(user, c.Param("sessionId"))
	writeServiceResult(c, resp, appErr)
}

func writeEvent(c *gin.Context, event services.OnCallEvent) error {
	var err error
	if event.Event == "heartbeat" {
		_, err = fmt.Fprintf(c.Writer, ": heartbeat %s\n\n", event.Data)
	} else {
		_, err = fmt.Fprintf(c.Writer, "event: %s\n", event.Event)
		if err == nil {
			_, err = fmt.Fprintf(c.Writer, "data: %s\n\n", event.Data)
		}
	}
	if err != nil {
		return err
	}
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func writeSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
}

func writeServiceResult(c *gin.Context, body any, appErr *services.AppError) {
	if appErr != nil {
		response.Error(c, appErr.Status, appErr.Code, appErr.Message)
		return
	}
	response.OK(c, http.StatusOK, body)
}
