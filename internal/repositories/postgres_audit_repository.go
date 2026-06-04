package repositories

import (
	"opentab-server/internal/database"
	"opentab-server/internal/models"

	"gorm.io/gorm"
)

type PostgresAuditRepository struct {
	db *gorm.DB
}

func NewPostgresAuditRepository(db *gorm.DB) *PostgresAuditRepository {
	return &PostgresAuditRepository{db: db}
}

func (r *PostgresAuditRepository) Record(log models.AuditLog) error {
	record := database.AuditLogRecord{
		ID:         log.ID,
		RequestID:  log.RequestID,
		UserID:     log.UserID,
		Account:    log.Account,
		Action:     log.Action,
		Method:     log.Method,
		Path:       log.Path,
		StatusCode: log.StatusCode,
		Result:     log.Result,
		ErrorCode:  log.ErrorCode,
		ClientIP:   log.ClientIP,
		UserAgent:  log.UserAgent,
		DurationMS: log.DurationMS,
	}
	return r.db.Create(&record).Error
}
