package repositories

import (
	"fmt"
	"strings"
	"time"

	"opentab-server/internal/mockdata"
	"opentab-server/internal/models"
)

type MemoryOnCallRepository struct{}

func NewMemoryOnCallRepository() *MemoryOnCallRepository {
	return &MemoryOnCallRepository{}
}

func (r *MemoryOnCallRepository) CreateSession(userID string, title string) (*models.OnCallSession, error) {
	if strings.TrimSpace(title) == "" {
		title = "新的 OnCall 会话"
	}
	now := time.Now().Format(time.RFC3339)
	session := models.OnCallSession{
		SessionID: fmt.Sprintf("sess-%03d", len(mockdata.OnCallSessions[userID])+1),
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}
	mockdata.OnCallSessions[userID] = append(mockdata.OnCallSessions[userID], session)
	return &session, nil
}

func (r *MemoryOnCallRepository) ListSessions(userID string) ([]models.OnCallSession, error) {
	sessions := mockdata.OnCallSessions[userID]
	result := make([]models.OnCallSession, len(sessions))
	copy(result, sessions)
	for i := range result {
		result[i].MessageCount = len(mockdata.OnCallMessages[result[i].SessionID])
	}
	return result, nil
}

func (r *MemoryOnCallRepository) FindSession(userID string, sessionID string) (*models.OnCallSession, error) {
	for i := range mockdata.OnCallSessions[userID] {
		if mockdata.OnCallSessions[userID][i].SessionID == sessionID {
			return &mockdata.OnCallSessions[userID][i], nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryOnCallRepository) DeleteSession(userID string, sessionID string) error {
	for i := range mockdata.OnCallSessions[userID] {
		if mockdata.OnCallSessions[userID][i].SessionID == sessionID {
			mockdata.OnCallSessions[userID] = append(mockdata.OnCallSessions[userID][:i], mockdata.OnCallSessions[userID][i+1:]...)
			delete(mockdata.OnCallMessages, sessionID)
			return nil
		}
	}
	return ErrNotFound
}

func (r *MemoryOnCallRepository) AddMessage(userID string, sessionID string, role string, content string, contentType string) (*models.OnCallMessage, error) {
	session, err := r.FindSession(userID, sessionID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "text"
	}
	now := time.Now().Format(time.RFC3339)
	message := models.OnCallMessage{
		MessageID:   fmt.Sprintf("msg-%03d", len(mockdata.OnCallMessages[sessionID])+1),
		SessionID:   sessionID,
		Role:        role,
		Content:     content,
		ContentType: contentType,
		CreatedAt:   now,
	}
	mockdata.OnCallMessages[sessionID] = append(mockdata.OnCallMessages[sessionID], message)
	session.UpdatedAt = now
	session.MessageCount = len(mockdata.OnCallMessages[sessionID])
	return &message, nil
}

func (r *MemoryOnCallRepository) ListMessages(userID string, sessionID string) ([]models.OnCallMessage, error) {
	if _, err := r.FindSession(userID, sessionID); err != nil {
		return nil, err
	}
	messages := mockdata.OnCallMessages[sessionID]
	result := make([]models.OnCallMessage, len(messages))
	copy(result, messages)
	return result, nil
}

func (r *MemoryOnCallRepository) FindMessage(userID string, sessionID string, messageID string) (*models.OnCallMessage, error) {
	if _, err := r.FindSession(userID, sessionID); err != nil {
		return nil, err
	}
	for i := range mockdata.OnCallMessages[sessionID] {
		if mockdata.OnCallMessages[sessionID][i].MessageID == messageID {
			return &mockdata.OnCallMessages[sessionID][i], nil
		}
	}
	return nil, ErrNotFound
}
