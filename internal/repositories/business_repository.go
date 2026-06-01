package repositories

import "opentab-server/internal/models"

type BusinessRepository interface {
	ApprovalSummary(userID string) (*models.ApprovalSummary, error)
	ListApprovalItems(userID string, status string) ([]models.ApprovalItem, error)
	FindApprovalItem(userID string, itemID string) (*models.ApprovalItem, error)
	UpdateApprovalStatus(userID string, itemID string, status string, comment string) (*models.ApprovalItem, error)
	CalendarSummary(userID string) (*models.CalendarSummary, error)
	ListCalendarEvents(userID string, date string) ([]models.CalendarEvent, error)
	FindCalendarEvent(userID string, eventID string) (*models.CalendarEvent, error)
	CreateCalendarEvent(userID string, req models.CreateCalendarEventRequest) (*models.CalendarEvent, error)
}
