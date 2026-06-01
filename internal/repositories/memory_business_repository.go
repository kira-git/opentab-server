package repositories

import (
	"fmt"
	"strings"
	"time"

	"opentab-server/internal/mockdata"
	"opentab-server/internal/models"
)

type MemoryBusinessRepository struct{}

func NewMemoryBusinessRepository() *MemoryBusinessRepository {
	return &MemoryBusinessRepository{}
}

var memoryApprovalItemsByUser = map[string][]models.ApprovalItem{}
var memoryCalendarEventsByUser = map[string][]models.CalendarEvent{}

func (r *MemoryBusinessRepository) ApprovalSummary(userID string) (*models.ApprovalSummary, error) {
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
		if item.Status == "approved" && strings.HasPrefix(item.UpdatedAt, today) {
			approvedToday++
		}
	}
	return &models.ApprovalSummary{PendingCount: pendingCount, ApprovedToday: approvedToday, Items: items}, nil
}

func (r *MemoryBusinessRepository) ListApprovalItems(userID string, status string) ([]models.ApprovalItem, error) {
	if status == "" {
		status = "all"
	}
	items := memoryApprovalItems(userID)
	result := make([]models.ApprovalItem, 0)
	for _, item := range items {
		if status == "all" || item.Status == status {
			result = append(result, item)
		}
	}
	return result, nil
}

func (r *MemoryBusinessRepository) FindApprovalItem(userID string, itemID string) (*models.ApprovalItem, error) {
	items := memoryApprovalItems(userID)
	for i := range items {
		if items[i].ID == itemID {
			return &items[i], nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryBusinessRepository) UpdateApprovalStatus(userID string, itemID string, status string, comment string) (*models.ApprovalItem, error) {
	items := memoryApprovalItems(userID)
	for i := range items {
		if items[i].ID == itemID {
			items[i].Status = status
			items[i].Comment = comment
			items[i].UpdatedAt = time.Now().Format(time.RFC3339)
			memoryApprovalItemsByUser[userID] = items
			return &items[i], nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryBusinessRepository) CalendarSummary(userID string) (*models.CalendarSummary, error) {
	events, err := r.ListCalendarEvents(userID, time.Now().Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	return &models.CalendarSummary{TodayCount: len(events), Events: events}, nil
}

func (r *MemoryBusinessRepository) ListCalendarEvents(userID string, date string) ([]models.CalendarEvent, error) {
	events := memoryCalendarEvents(userID)
	result := make([]models.CalendarEvent, 0)
	for _, event := range events {
		if date == "" || strings.HasPrefix(event.StartTime, date) {
			result = append(result, event)
		}
	}
	return result, nil
}

func (r *MemoryBusinessRepository) FindCalendarEvent(userID string, eventID string) (*models.CalendarEvent, error) {
	events := memoryCalendarEvents(userID)
	for i := range events {
		if events[i].ID == eventID {
			return &events[i], nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryBusinessRepository) CreateCalendarEvent(userID string, req models.CreateCalendarEventRequest) (*models.CalendarEvent, error) {
	events := memoryCalendarEvents(userID)
	event := models.CalendarEvent{
		ID:          fmt.Sprintf("evt-%03d", len(events)+1),
		Title:       req.Title,
		Description: req.Description,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Location:    req.Location,
	}
	memoryCalendarEventsByUser[userID] = append(events, event)
	return &event, nil
}

func memoryApprovalItems(userID string) []models.ApprovalItem {
	if items, ok := memoryApprovalItemsByUser[userID]; ok {
		return items
	}
	items := make([]models.ApprovalItem, len(mockdata.ApprovalSummary.Items))
	copy(items, mockdata.ApprovalSummary.Items)
	memoryApprovalItemsByUser[userID] = items
	return items
}

func memoryCalendarEvents(userID string) []models.CalendarEvent {
	if events, ok := memoryCalendarEventsByUser[userID]; ok {
		return events
	}
	events := make([]models.CalendarEvent, len(mockdata.CalendarSummary.Events))
	copy(events, mockdata.CalendarSummary.Events)
	memoryCalendarEventsByUser[userID] = events
	return events
}
