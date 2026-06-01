package repositories

import "opentab-server/internal/models"

type OnCallRepository interface {
	CreateSession(userID string, title string) (*models.OnCallSession, error)
	ListSessions(userID string) ([]models.OnCallSession, error)
	FindSession(userID string, sessionID string) (*models.OnCallSession, error)
	DeleteSession(userID string, sessionID string) error
	AddMessage(userID string, sessionID string, role string, content string, contentType string) (*models.OnCallMessage, error)
	ListMessages(userID string, sessionID string) ([]models.OnCallMessage, error)
	FindMessage(userID string, sessionID string, messageID string) (*models.OnCallMessage, error)
}
