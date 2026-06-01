package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	ai     *AIStreamClient
}

func NewOnCallService(oncall repositories.OnCallRepository, aiServiceBaseURL string) *OnCallService {
	return &OnCallService{
		oncall: oncall,
		ai:     NewAIStreamClient(aiServiceBaseURL),
	}
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

func (s *OnCallService) StreamAIChat(ctx context.Context, message string, conversationID string, emit func(OnCallEvent) error) error {
	return s.streamAI(ctx, message, conversationID, emit, convertAIEventToDocEvent)
}

func (s *OnCallService) StreamOnCallQuery(ctx context.Context, message string, emit func(OnCallEvent) error) error {
	return s.streamAI(ctx, message, "", emit, convertAIEventToClientEvent)
}

func (s *OnCallService) StreamOnCallMessage(ctx context.Context, user *models.User, sessionID string, messageID string, emit func(OnCallEvent) error) *AppError {
	message, err := s.oncall.FindMessage(user.ID, sessionID, messageID)
	if errors.Is(err, repositories.ErrNotFound) {
		return NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "AI 会话或消息不存在")
	}
	if err != nil {
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "生成 AI 回复失败")
	}

	var assistantBuilder strings.Builder
	var assistantMessageID string
	streamErr := s.streamAI(ctx, message.Content, sessionID, func(event OnCallEvent) error {
		if event.Event == "delta" {
			assistantBuilder.WriteString(readJSONText(event.Data, "text"))
		}
		if event.Event == "done" {
			assistant, err := s.oncall.AddMessage(user.ID, sessionID, "assistant", assistantBuilder.String(), "text")
			if err == nil {
				assistantMessageID = assistant.MessageID
				event.Data = `{"messageId":"` + jsonEscape(assistantMessageID) + `"}`
			}
		}
		if err := emit(event); err != nil {
			return err
		}
		return nil
	}, convertAIEventToClientEvent)
	if streamErr != nil {
		return NewAppError(http.StatusBadGateway, "AI_SERVICE_ERROR", "AI 服务调用失败: "+streamErr.Error())
	}
	if assistantMessageID == "" && assistantBuilder.Len() > 0 {
		_, _ = s.oncall.AddMessage(user.ID, sessionID, "assistant", assistantBuilder.String(), "text")
	}
	return nil
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

func (s *OnCallService) streamAI(ctx context.Context, message string, conversationID string, emit func(OnCallEvent) error, convert func(AIChatEvent) (OnCallEvent, bool)) error {
	if strings.TrimSpace(message) == "" {
		message = "如何接入一个业务 Tab？"
	}

	if !s.ai.Available() {
		return emitConvertedMockEvents(message, emit, convert)
	}

	return s.ai.Stream(ctx, AIChatRequest{
		Message:        message,
		ConversationID: conversationID,
	}, func(event AIChatEvent) error {
		converted, ok := convert(event)
		if !ok {
			return nil
		}
		return emit(converted)
	})
}

func emitConvertedMockEvents(message string, emit func(OnCallEvent) error, convert func(AIChatEvent) (OnCallEvent, bool)) error {
	events := []AIChatEvent{
		{Event: "message", Data: map[string]any{"type": "intent", "intent": "PROTOCOL_QA"}},
		{Event: "message", Data: map[string]any{"type": "tool", "tool": "search"}},
		{Event: "message", Data: map[string]any{"type": "tool", "tool": "read"}},
	}
	content := "我现在按 AI OnCall 协议返回流式回答。你可以询问 Tab 接入、协议配置或错误日志。"
	if strings.Contains(strings.ToLower(message), "tab") {
		content = "接入业务 Tab 时，先定义 TabManifest，再由容器根据权限和入口类型加载对应页面。"
	}
	events = append(events,
		AIChatEvent{Event: "message", Data: map[string]any{"type": "content", "delta": content}},
		AIChatEvent{Event: "message", Data: map[string]any{"type": "done", "messageId": "mock-ai-message"}},
	)
	for _, event := range events {
		converted, ok := convert(event)
		if !ok {
			continue
		}
		if err := emit(converted); err != nil {
			return err
		}
	}
	return nil
}

func convertAIEventToDocEvent(event AIChatEvent) (OnCallEvent, bool) {
	data, err := json.Marshal(event.Data)
	if err != nil {
		return OnCallEvent{}, false
	}
	return OnCallEvent{Event: "message", Data: string(data)}, true
}

func convertAIEventToClientEvent(event AIChatEvent) (OnCallEvent, bool) {
	eventType := readMapString(event.Data, "type")
	switch eventType {
	case "content":
		return OnCallEvent{
			Event: "delta",
			Data:  `{"text":"` + jsonEscape(readMapString(event.Data, "delta")) + `"}`,
		}, true
	case "tool":
		tool := readMapString(event.Data, "tool")
		return OnCallEvent{
			Event: "tool",
			Data: fmt.Sprintf(
				`{"name":"%s","status":"running","summary":"%s"}`,
				jsonEscape(tool),
				jsonEscape(toolSummary(tool)),
			),
		}, true
	case "done":
		messageID := readMapString(event.Data, "messageId")
		return OnCallEvent{
			Event: "done",
			Data:  `{"messageId":"` + jsonEscape(messageID) + `"}`,
		}, true
	case "error":
		code := readMapString(event.Data, "code")
		if code == "" {
			code = "AI_SERVICE_ERROR"
		}
		message := readMapString(event.Data, "delta")
		return OnCallEvent{
			Event: "error",
			Data:  `{"code":"` + jsonEscape(code) + `","message":"` + jsonEscape(message) + `"}`,
		}, true
	default:
		return OnCallEvent{}, false
	}
}

func readMapString(data map[string]any, key string) string {
	value, ok := data[key]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func readJSONText(data string, key string) string {
	var obj map[string]string
	if err := json.Unmarshal([]byte(data), &obj); err != nil {
		return ""
	}
	return obj[key]
}

func jsonEscape(value string) string {
	bytes, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	quoted := string(bytes)
	if len(quoted) < 2 {
		return ""
	}
	return quoted[1 : len(quoted)-1]
}

func toolSummary(tool string) string {
	switch tool {
	case "search":
		return "正在检索相关资料..."
	case "read":
		return "正在阅读协议文档..."
	case "analyze":
		return "正在分析问题..."
	case "generate":
		return "正在生成代码..."
	default:
		return "正在调用 AI 工具..."
	}
}
