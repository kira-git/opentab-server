package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"opentab-server/internal/models"
	"opentab-server/internal/repositories"
)

type OnCallEvent struct {
	Event string
	Data  string
}

type OnCallService struct {
	oncall        repositories.OnCallRepository
	ai            *AIStreamClient
	activeStreams map[string]activeStream
	streamsMu     sync.Mutex
	aiLimiter     *AIConcurrencyLimiter
	userLimiter   *UserAIStreamLimiter
	smoother      *StreamSmoother
}

type OnCallOptions struct {
	AIConcurrentLimit     int
	AIUserConcurrentLimit int
	SmoothInterval        time.Duration
	SmoothChunkSize       int
}

func NewOnCallService(oncall repositories.OnCallRepository, aiServiceBaseURL string) *OnCallService {
	return NewOnCallServiceWithOptions(oncall, aiServiceBaseURL, OnCallOptions{
		AIConcurrentLimit:     3,
		AIUserConcurrentLimit: 1,
		SmoothInterval:        25 * time.Millisecond,
		SmoothChunkSize:       2,
	})
}

func NewOnCallServiceWithOptions(oncall repositories.OnCallRepository, aiServiceBaseURL string, opts OnCallOptions) *OnCallService {
	return &OnCallService{
		oncall:        oncall,
		ai:            NewAIStreamClient(aiServiceBaseURL),
		activeStreams: map[string]activeStream{},
		aiLimiter:     NewAIConcurrencyLimiter(opts.AIConcurrentLimit),
		userLimiter:   NewUserAIStreamLimiter(opts.AIUserConcurrentLimit),
		smoother:      NewStreamSmoother(opts.SmoothInterval, opts.SmoothChunkSize),
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
	_ = s.cancelActiveStream(user.ID, sessionID)
	err := s.oncall.DeleteSession(user.ID, sessionID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "AI 会话不存在")
	}
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "删除 AI 会话失败")
	}
	return &models.DeleteOnCallSessionResponse{Success: true, SessionID: sessionID}, nil
}

func (s *OnCallService) CancelSessionGeneration(user *models.User, sessionID string) (*models.CancelOnCallGenerationResponse, *AppError) {
	if _, err := s.oncall.FindSession(user.ID, sessionID); errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "AI 会话不存在")
	} else if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "取消生成失败")
	}
	return &models.CancelOnCallGenerationResponse{
		Success:   true,
		SessionID: sessionID,
		Cancelled: s.cancelActiveStream(user.ID, sessionID),
	}, nil
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
	return s.streamAIWithOptions(ctx, "anonymous", "", "", message, conversationID, emit, convertAIEventToDocEvent)
}

func (s *OnCallService) StreamAIChatForUser(ctx context.Context, user *models.User, message string, conversationID string, emit func(OnCallEvent) error) error {
	userID := "anonymous"
	if user != nil && user.ID != "" {
		userID = user.ID
	}
	return s.streamAIWithOptions(ctx, userID, "", "", message, conversationID, emit, convertAIEventToDocEvent)
}

func (s *OnCallService) StreamOnCallQuery(ctx context.Context, message string, emit func(OnCallEvent) error) error {
	return s.streamAIWithOptions(ctx, "anonymous", "", "", message, "", emit, convertAIEventToClientEvent)
}

func (s *OnCallService) StreamOnCallMessage(ctx context.Context, user *models.User, sessionID string, messageID string, emit func(OnCallEvent) error) *AppError {
	message, err := s.oncall.FindMessage(user.ID, sessionID, messageID)
	if errors.Is(err, repositories.ErrNotFound) {
		return NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "AI 会话或消息不存在")
	}
	if err != nil {
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "生成 AI 回复失败")
	}

	if existingAssistant := s.findExistingAssistantReply(user.ID, sessionID, messageID); existingAssistant != nil {
		if err := emit(OnCallEvent{
			Event: "done",
			Data:  `{"messageId":"` + jsonEscape(existingAssistant.MessageID) + `"}`,
		}); err != nil {
			return NewAppError(http.StatusInternalServerError, "STREAM_WRITE_FAILED", "写入 AI 流失败")
		}
		return nil
	}

	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	streamID := s.registerActiveStream(user.ID, sessionID, cancel)
	defer s.unregisterActiveStream(user.ID, sessionID, streamID)

	var assistantBuilder strings.Builder
	var assistantMessageID string
	streamErr := s.streamAIWithOptions(streamCtx, user.ID, activeStreamKey(user.ID, sessionID), streamID, message.Content, sessionID, func(event OnCallEvent) error {
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
		if errors.Is(streamErr, context.Canceled) {
			return NewAppError(http.StatusOK, "GENERATION_CANCELLED", "AI 生成已取消")
		}
		return NewAppError(http.StatusBadGateway, "AI_SERVICE_ERROR", "AI 服务调用失败: "+streamErr.Error())
	}
	if assistantMessageID == "" && assistantBuilder.Len() > 0 {
		_, _ = s.oncall.AddMessage(user.ID, sessionID, "assistant", assistantBuilder.String(), "text")
	}
	return nil
}

func (s *OnCallService) findExistingAssistantReply(userID string, sessionID string, messageID string) *models.OnCallMessage {
	messages, err := s.oncall.ListMessages(userID, sessionID)
	if err != nil {
		return nil
	}
	foundUserMessage := false
	for i := range messages {
		message := messages[i]
		if !foundUserMessage {
			foundUserMessage = message.MessageID == messageID
			continue
		}
		if message.Role == "user" {
			return nil
		}
		if message.Role == "assistant" && strings.TrimSpace(message.Content) != "" {
			return &message
		}
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

func (s *OnCallService) streamAIWithOptions(ctx context.Context, userID string, userStreamKey string, userStreamID string, message string, conversationID string, emit func(OnCallEvent) error, convert func(AIChatEvent) (OnCallEvent, bool)) error {
	if strings.TrimSpace(message) == "" {
		message = "如何接入一个业务 Tab？"
	}

	if !s.ai.Available() {
		return emitConvertedMockEvents(message, emit, convert)
	}

	if userID != "" {
		if err := s.userLimiter.TryAcquireStream(userID, userStreamKey, userStreamID); err != nil {
			return emit(OnCallEvent{
				Event: "error",
				Data:  `{"code":"AI_USER_BUSY","message":"当前用户已有 AI 生成任务，请稍后再试"}`,
			})
		}
		defer s.userLimiter.ReleaseStream(userID, userStreamKey, userStreamID)
	}

	if err := s.aiLimiter.TryAcquire(ctx); err != nil {
		return emit(OnCallEvent{
			Event: "error",
			Data:  `{"code":"AI_TOO_MANY_REQUESTS","message":"当前 AI 请求较多，请稍后再试"}`,
		})
	}
	defer s.aiLimiter.Release()

	if err := emit(OnCallEvent{
		Event: "status",
		Data:  `{"stage":"ai_processing","message":"AI 正在分析"}`,
	}); err != nil {
		return err
	}

	var lastErr error
	for attempt := 1; attempt <= aiStreamMaxAttempts; attempt++ {
		emitted, err := s.streamAIOnce(ctx, message, conversationID, emit, convert)
		if err == nil {
			return nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return err
		}
		if emitted {
			return err
		}
	}
	if lastErr != nil {
		return emitConvertedMockEvents(message, emit, convert)
	}
	return nil
}

func (s *OnCallService) streamAIOnce(ctx context.Context, message string, conversationID string, emit func(OnCallEvent) error, convert func(AIChatEvent) (OnCallEvent, bool)) (bool, error) {
	rawBuffer := make(chan RawChunk, 64)
	directEvents := make(chan OnCallEvent, 16)
	smootherOut := make(chan OnCallEvent, 64)
	aiDone := make(chan error, 1)

	go s.smoother.Run(ctx, rawBuffer, smootherOut)

	go func() {
		defer close(rawBuffer)
		defer close(directEvents)
		err := s.ai.Stream(ctx, AIChatRequest{
			Message:        message,
			ConversationID: conversationID,
		}, func(event AIChatEvent) error {
			converted, ok := convert(event)
			if !ok {
				return nil
			}
			switch converted.Event {
			case "delta":
				text := readJSONText(converted.Data, "text")
				select {
				case rawBuffer <- RawChunk{Text: text}:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			case "done":
				select {
				case rawBuffer <- RawChunk{Done: true}:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			case "error":
				select {
				case rawBuffer <- RawChunk{Err: errors.New(converted.Data)}:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			default:
				select {
				case directEvents <- converted:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		})
		aiDone <- err
	}()

	ticker := time.NewTicker(aiStreamHeartbeatInterval)
	defer ticker.Stop()

	emittedBusinessEvent := false
	smootherClosed := false
	directClosed := false
	for {
		if smootherClosed && directClosed {
			err := <-aiDone
			if err != nil {
				return emittedBusinessEvent, err
			}
			return emittedBusinessEvent, nil
		}
		select {
		case event, ok := <-directEvents:
			if !ok {
				directClosed = true
				directEvents = nil
				continue
			}
			if event.Event == "delta" || event.Event == "tool" || event.Event == "done" {
				emittedBusinessEvent = true
			}
			if err := emit(event); err != nil {
				return emittedBusinessEvent, err
			}
		case event, ok := <-smootherOut:
			if !ok {
				smootherClosed = true
				smootherOut = nil
				continue
			}
			if event.Event == "delta" || event.Event == "tool" || event.Event == "done" {
				emittedBusinessEvent = true
			}
			if err := emit(event); err != nil {
				return emittedBusinessEvent, err
			}
		case <-ticker.C:
			if err := emit(OnCallEvent{Event: "heartbeat", Data: `{"status":"running"}`}); err != nil {
				return emittedBusinessEvent, err
			}
		case <-ctx.Done():
			return emittedBusinessEvent, ctx.Err()
		}
	}
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

func activeStreamKey(userID string, sessionID string) string {
	return userID + ":" + sessionID
}

type activeStream struct {
	id     string
	cancel context.CancelFunc
}

func (s *OnCallService) registerActiveStream(userID string, sessionID string, cancel context.CancelFunc) string {
	key := activeStreamKey(userID, sessionID)
	streamID := "stream-" + randomHex(8)
	s.streamsMu.Lock()
	defer s.streamsMu.Unlock()
	if existing := s.activeStreams[key]; existing.cancel != nil {
		existing.cancel()
	}
	s.activeStreams[key] = activeStream{id: streamID, cancel: cancel}
	return streamID
}

func (s *OnCallService) unregisterActiveStream(userID string, sessionID string, streamID string) {
	key := activeStreamKey(userID, sessionID)
	s.streamsMu.Lock()
	defer s.streamsMu.Unlock()
	if current := s.activeStreams[key]; current.id == streamID {
		delete(s.activeStreams, key)
	}
}

func (s *OnCallService) cancelActiveStream(userID string, sessionID string) bool {
	key := activeStreamKey(userID, sessionID)
	s.streamsMu.Lock()
	stream := s.activeStreams[key]
	if stream.cancel != nil {
		delete(s.activeStreams, key)
	}
	s.streamsMu.Unlock()
	if stream.cancel == nil {
		return false
	}
	stream.cancel()
	return true
}

const aiStreamMaxAttempts = 2
const aiStreamHeartbeatInterval = 10 * time.Second
