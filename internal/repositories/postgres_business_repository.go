package repositories

import (
	"encoding/json"
	"fmt"
	"time"

	"opentab-server/internal/database"
	"opentab-server/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PostgresBusinessRepository struct {
	db *gorm.DB
}

func NewPostgresBusinessRepository(db *gorm.DB) *PostgresBusinessRepository {
	return &PostgresBusinessRepository{db: db}
}

func (r *PostgresBusinessRepository) ApprovalSummary(userID string) (*models.ApprovalSummary, error) {
	items, err := r.ListApprovalItems(userID, "all")
	if err != nil {
		return nil, err
	}
	pendingCount := 0
	approvedToday := 0
	today := time.Now().Format("2006-01-02")
	for _, item := range items {
		if item.Status == "pending" {
			pendingCount++
		}
		if item.Status == "approved" && len(item.UpdatedAt) >= 10 && item.UpdatedAt[:10] == today {
			approvedToday++
		}
	}
	return &models.ApprovalSummary{PendingCount: pendingCount, ApprovedToday: approvedToday, Items: items}, nil
}

func (r *PostgresBusinessRepository) ListApprovalItems(userID string, status string) ([]models.ApprovalItem, error) {
	var records []database.ApprovalItemRecord
	query := r.db.Where("user_id = ?", userID).Order("created_at DESC")
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	result := make([]models.ApprovalItem, 0, len(records))
	for _, record := range records {
		result = append(result, approvalRecordToModel(record))
	}
	return result, nil
}

func (r *PostgresBusinessRepository) FindApprovalItem(userID string, itemID string) (*models.ApprovalItem, error) {
	var record database.ApprovalItemRecord
	if err := r.db.Where("id = ? AND user_id = ?", itemID, userID).First(&record).Error; err != nil {
		return nil, mapGormError(err)
	}
	item := approvalRecordToModel(record)
	return &item, nil
}

func (r *PostgresBusinessRepository) UpdateApprovalStatus(userID string, itemID string, status string, comment string) (*models.ApprovalItem, error) {
	var record database.ApprovalItemRecord
	if err := r.db.Where("id = ? AND user_id = ?", itemID, userID).First(&record).Error; err != nil {
		return nil, mapGormError(err)
	}
	record.Status = status
	record.Comment = comment
	if err := r.db.Save(&record).Error; err != nil {
		return nil, err
	}
	item := approvalRecordToModel(record)
	return &item, nil
}

func (r *PostgresBusinessRepository) CalendarSummary(userID string) (*models.CalendarSummary, error) {
	events, err := r.ListCalendarEvents(userID, time.Now().Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	return &models.CalendarSummary{TodayCount: len(events), Events: events}, nil
}

func (r *PostgresBusinessRepository) ListCalendarEvents(userID string, date string) ([]models.CalendarEvent, error) {
	var records []database.CalendarEventRecord
	query := r.db.Where("user_id = ?", userID).Order("start_time ASC")
	if date != "" {
		start, err := time.Parse("2006-01-02", date)
		if err == nil {
			query = query.Where("start_time >= ? AND start_time < ?", start, start.AddDate(0, 0, 1))
		}
	}
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	result := make([]models.CalendarEvent, 0, len(records))
	for _, record := range records {
		event, err := calendarRecordToModel(record)
		if err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, nil
}

func (r *PostgresBusinessRepository) FindCalendarEvent(userID string, eventID string) (*models.CalendarEvent, error) {
	var record database.CalendarEventRecord
	if err := r.db.Where("id = ? AND user_id = ?", eventID, userID).First(&record).Error; err != nil {
		return nil, mapGormError(err)
	}
	event, err := calendarRecordToModel(record)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *PostgresBusinessRepository) CreateCalendarEvent(userID string, req models.CreateCalendarEventRequest) (*models.CalendarEvent, error) {
	record := database.CalendarEventRecord{
		ID:          fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		StartTime:   parseRFC3339OrNow(req.StartTime),
		EndTime:     parseRFC3339OrNow(req.EndTime),
		Location:    req.Location,
	}
	participantsJSON, _ := json.Marshal([]string{})
	record.ParticipantsJSON = datatypes.JSON(participantsJSON)
	if err := r.db.Create(&record).Error; err != nil {
		return nil, err
	}
	event, err := calendarRecordToModel(record)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func approvalRecordToModel(record database.ApprovalItemRecord) models.ApprovalItem {
	return models.ApprovalItem{
		ID:        record.ID,
		Title:     record.Title,
		Applicant: record.Applicant,
		Amount:    record.Amount,
		Reason:    record.Reason,
		Status:    record.Status,
		CreatedAt: formatTime(record.CreatedAt),
		Comment:   record.Comment,
		UpdatedAt: formatTime(record.UpdatedAt),
	}
}

func calendarRecordToModel(record database.CalendarEventRecord) (models.CalendarEvent, error) {
	participants := []string{}
	if len(record.ParticipantsJSON) > 0 {
		if err := json.Unmarshal(record.ParticipantsJSON, &participants); err != nil {
			return models.CalendarEvent{}, err
		}
	}
	return models.CalendarEvent{
		ID:           record.ID,
		Title:        record.Title,
		Description:  record.Description,
		StartTime:    formatTime(record.StartTime),
		EndTime:      formatTime(record.EndTime),
		Location:     record.Location,
		Participants: participants,
	}, nil
}

func parseRFC3339OrNow(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Now()
	}
	return parsed
}
