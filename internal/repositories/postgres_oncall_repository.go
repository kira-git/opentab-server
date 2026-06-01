package repositories

import (
	"fmt"
	"time"

	"opentab-server/internal/database"
	"opentab-server/internal/models"

	"gorm.io/gorm"
)

type PostgresOnCallRepository struct {
	db *gorm.DB
}

func NewPostgresOnCallRepository(db *gorm.DB) *PostgresOnCallRepository {
	return &PostgresOnCallRepository{db: db}
}

func (r *PostgresOnCallRepository) CreateSession(userID string, title string) (*models.OnCallSession, error) {
	if title == "" {
		title = "新的 OnCall 会话"
	}
	now := time.Now()
	record := database.OnCallSessionRecord{
		ID:        fmt.Sprintf("sess-%d", now.UnixNano()),
		UserID:    userID,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := r.db.Create(&record).Error; err != nil {
		return nil, err
	}
	session := onCallSessionRecordToModel(record, 0)
	return &session, nil
}

func (r *PostgresOnCallRepository) ListSessions(userID string) ([]models.OnCallSession, error) {
	var records []database.OnCallSessionRecord
	if err := r.db.
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("updated_at DESC").
		Find(&records).Error; err != nil {
		return nil, err
	}
	result := make([]models.OnCallSession, 0, len(records))
	for _, record := range records {
		var count int64
		_ = r.db.Model(&database.OnCallMessageRecord{}).Where("session_id = ?", record.ID).Count(&count).Error
		result = append(result, onCallSessionRecordToModel(record, int(count)))
	}
	return result, nil
}

func (r *PostgresOnCallRepository) FindSession(userID string, sessionID string) (*models.OnCallSession, error) {
	var record database.OnCallSessionRecord
	if err := r.db.Where("id = ? AND user_id = ? AND deleted_at IS NULL", sessionID, userID).First(&record).Error; err != nil {
		return nil, mapGormError(err)
	}
	var count int64
	_ = r.db.Model(&database.OnCallMessageRecord{}).Where("session_id = ?", record.ID).Count(&count).Error
	session := onCallSessionRecordToModel(record, int(count))
	return &session, nil
}

func (r *PostgresOnCallRepository) DeleteSession(userID string, sessionID string) error {
	now := time.Now()
	result := r.db.Model(&database.OnCallSessionRecord{}).
		Where("id = ? AND user_id = ?", sessionID, userID).
		Update("deleted_at", &now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresOnCallRepository) AddMessage(userID string, sessionID string, role string, content string, contentType string) (*models.OnCallMessage, error) {
	if _, err := r.FindSession(userID, sessionID); err != nil {
		return nil, err
	}
	if contentType == "" {
		contentType = "text"
	}
	now := time.Now()
	record := database.OnCallMessageRecord{
		ID:          fmt.Sprintf("msg-%d", now.UnixNano()),
		SessionID:   sessionID,
		Role:        role,
		Content:     content,
		ContentType: contentType,
		CreatedAt:   now,
	}
	if err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		return tx.Model(&database.OnCallSessionRecord{}).Where("id = ?", sessionID).Update("updated_at", now).Error
	}); err != nil {
		return nil, err
	}
	message := onCallMessageRecordToModel(record)
	return &message, nil
}

func (r *PostgresOnCallRepository) ListMessages(userID string, sessionID string) ([]models.OnCallMessage, error) {
	if _, err := r.FindSession(userID, sessionID); err != nil {
		return nil, err
	}
	var records []database.OnCallMessageRecord
	if err := r.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	result := make([]models.OnCallMessage, 0, len(records))
	for _, record := range records {
		result = append(result, onCallMessageRecordToModel(record))
	}
	return result, nil
}

func (r *PostgresOnCallRepository) FindMessage(userID string, sessionID string, messageID string) (*models.OnCallMessage, error) {
	if _, err := r.FindSession(userID, sessionID); err != nil {
		return nil, err
	}
	var record database.OnCallMessageRecord
	if err := r.db.Where("id = ? AND session_id = ?", messageID, sessionID).First(&record).Error; err != nil {
		return nil, mapGormError(err)
	}
	message := onCallMessageRecordToModel(record)
	return &message, nil
}

func onCallSessionRecordToModel(record database.OnCallSessionRecord, messageCount int) models.OnCallSession {
	return models.OnCallSession{
		SessionID:    record.ID,
		Title:        record.Title,
		CreatedAt:    formatTime(record.CreatedAt),
		UpdatedAt:    formatTime(record.UpdatedAt),
		MessageCount: messageCount,
	}
}

func onCallMessageRecordToModel(record database.OnCallMessageRecord) models.OnCallMessage {
	return models.OnCallMessage{
		MessageID:   record.ID,
		SessionID:   record.SessionID,
		Role:        record.Role,
		Content:     record.Content,
		ContentType: record.ContentType,
		CreatedAt:   formatTime(record.CreatedAt),
	}
}
