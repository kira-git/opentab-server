package services

import (
	"errors"
	"net/http"

	"opentab-server/internal/models"
	"opentab-server/internal/repositories"
)

type BusinessService struct {
	business repositories.BusinessRepository
}

func NewBusinessService(business repositories.BusinessRepository) *BusinessService {
	return &BusinessService{business: business}
}

func (s *BusinessService) ApprovalSummary(user *models.User) (*models.ApprovalSummary, *AppError) {
	if !hasPermission(user, "tab.approval.read") {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "当前账号无权查看审批数据")
	}
	summary, err := s.business.ApprovalSummary(user.ID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "获取审批数据失败")
	}
	return summary, nil
}

func (s *BusinessService) ListApprovalItems(user *models.User, status string) ([]models.ApprovalItem, *AppError) {
	if !hasPermission(user, "tab.approval.read") {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "当前账号无权查看审批数据")
	}
	items, err := s.business.ListApprovalItems(user.ID, status)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "获取审批列表失败")
	}
	return items, nil
}

func (s *BusinessService) GetApprovalItem(user *models.User, itemID string) (*models.ApprovalItem, *AppError) {
	if !hasPermission(user, "tab.approval.read") {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "当前账号无权查看审批数据")
	}
	item, err := s.business.FindApprovalItem(user.ID, itemID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "审批记录不存在")
	}
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "获取审批详情失败")
	}
	return item, nil
}

func (s *BusinessService) ApproveItem(user *models.User, itemID string, comment string) (*models.ApprovalActionResponse, *AppError) {
	return s.updateApprovalStatus(user, itemID, "approved", comment)
}

func (s *BusinessService) RejectItem(user *models.User, itemID string, comment string) (*models.ApprovalActionResponse, *AppError) {
	return s.updateApprovalStatus(user, itemID, "rejected", comment)
}

func (s *BusinessService) updateApprovalStatus(user *models.User, itemID string, status string, comment string) (*models.ApprovalActionResponse, *AppError) {
	if !hasPermission(user, "tab.approval.read") {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "当前账号无权操作审批数据")
	}
	item, err := s.business.UpdateApprovalStatus(user.ID, itemID, status, comment)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "审批记录不存在")
	}
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "更新审批状态失败")
	}
	return &models.ApprovalActionResponse{Success: true, ItemID: item.ID, Status: item.Status}, nil
}

func (s *BusinessService) CalendarSummary(user *models.User) (*models.CalendarSummary, *AppError) {
	if !hasPermission(user, "tab.calendar.read") {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "当前账号无权查看日程数据")
	}
	summary, err := s.business.CalendarSummary(user.ID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "获取日程数据失败")
	}
	return summary, nil
}

func (s *BusinessService) ListCalendarEvents(user *models.User, date string) ([]models.CalendarEvent, *AppError) {
	if !hasPermission(user, "tab.calendar.read") {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "当前账号无权查看日程数据")
	}
	events, err := s.business.ListCalendarEvents(user.ID, date)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "获取日程列表失败")
	}
	return events, nil
}

func (s *BusinessService) GetCalendarEvent(user *models.User, eventID string) (*models.CalendarEvent, *AppError) {
	if !hasPermission(user, "tab.calendar.read") {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "当前账号无权查看日程数据")
	}
	event, err := s.business.FindCalendarEvent(user.ID, eventID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "日程不存在")
	}
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "获取日程详情失败")
	}
	return event, nil
}

func (s *BusinessService) CreateCalendarEvent(user *models.User, req models.CreateCalendarEventRequest) (*models.CreateCalendarEventResponse, *AppError) {
	if !hasPermission(user, "tab.calendar.read") {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "当前账号无权新增日程")
	}
	if req.Title == "" || req.StartTime == "" || req.EndTime == "" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_REQUEST", "title、startTime、endTime 不可为空")
	}
	event, err := s.business.CreateCalendarEvent(user.ID, req)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "新增日程失败")
	}
	return &models.CreateCalendarEventResponse{Success: true, EventID: event.ID}, nil
}
