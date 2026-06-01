package services

import (
	"errors"
	"net/http"
	"strings"

	"opentab-server/internal/models"
	"opentab-server/internal/repositories"
)

type OnCallEvent struct {
	Event string
	Data  string
}

type OnCallService struct {
	oncall repositories.OnCallRepository
}

func NewOnCallService(oncall repositories.OnCallRepository) *OnCallService {
	return &OnCallService{oncall: oncall}
}

func (s *OnCallService) CreateSession(user *models.User, title string) (*models.OnCallSession, *AppError) {
	session, err := s.oncall.CreateSession(user.ID, title)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "创建 AI 会话失败")
	}
	return session, nil
}

func (s *OnCallService) ListSessions(user *models.User) ([]models.OnCallSession, *AppError) {
	sessions, err := s.oncall.ListSessions(user.ID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "获取 AI 会话列表失败")
	}
	return sessions, nil
}

func (s *OnCallService) AddUserMessage(user *models.User, sessionID string, req models.OnCallMessageRequest) (*models.OnCallMessage, *AppError) {
	if strings.TrimSpace(req.Content) == "" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_REQUEST", "content 不可为空")
	}
	message, err := s.oncall.AddMessage(user.ID, sessionID, "user", req.Content, req.ContentType)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "AI 会话不存在")
	}
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "保存消息失败")
	}
	return message, nil
}

func (s *OnCallService) ListMessages(user *models.User, sessionID string) ([]models.OnCallMessage, *AppError) {
	messages, err := s.oncall.ListMessages(user.ID, sessionID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "AI 会话不存在")
	}
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "获取消息记录失败")
	}
	return messages, nil
}

func (s *OnCallService) DeleteSession(user *models.User, sessionID string) (*models.DeleteOnCallSessionResponse, *AppError) {
	err := s.oncall.DeleteSession(user.ID, sessionID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "AI 会话不存在")
	}
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "删除 AI 会话失败")
	}
	return &models.DeleteOnCallSessionResponse{Success: true, SessionID: sessionID}, nil
}

func (s *OnCallService) StreamSessionReply(user *models.User, sessionID string, messageID string) ([]OnCallEvent, *AppError) {
	message, err := s.oncall.FindMessage(user.ID, sessionID, messageID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "AI 会话或消息不存在")
	}
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "生成 AI 回复失败")
	}

	events := s.MockReplyEvents(message.Content)
	assistant, err := s.oncall.AddMessage(user.ID, sessionID, "assistant", "接入业务 Tab 的第一步是定义 TabManifest。", "text")
	if err == nil && len(events) > 0 {
		events[len(events)-1] = OnCallEvent{Event: "done", Data: `{"messageId":"` + assistant.MessageID + `"}`}
	}
	return events, nil
}

func (s *OnCallService) MockReplyEvents(message string) []OnCallEvent {
	if strings.TrimSpace(message) == "" {
		message = "如何接入一个业务 Tab？"
	}

	events := []OnCallEvent{
		{Event: "delta", Data: `{"text":"我会先根据 TabManifest 协议检查入口类型、权限和容器版本。"}`},
		{Event: "delta", Data: `{"text":"如果你贴出配置或日志，我可以继续返回诊断建议。"}`},
	}

	if strings.Contains(strings.ToLower(message), "tabdefinition") || strings.Contains(message, "配置") || strings.Contains(message, "{") {
		events = append(events, OnCallEvent{Event: "tool", Data: `{"name":"validate_tab_config","status":"完成","summary":"已检查协议字段、权限声明、版本约束和入口类型。"}`})
	}

	return append(events, OnCallEvent{Event: "done", Data: `{}`})
}
