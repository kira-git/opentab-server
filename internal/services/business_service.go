package services

import (
	"context"
	"errors"
	"net/http"

	"opentab-server/internal/cache"
	"opentab-server/internal/models"
	"opentab-server/internal/policies"
	"opentab-server/internal/repositories"
)

type BusinessService struct {
	business  repositories.BusinessRepository
	authCache cache.AuthCache
}

func NewBusinessService(business repositories.BusinessRepository) *BusinessService {
	return NewBusinessServiceWithCache(business, cache.NewNoopAuthCache())
}

func NewBusinessServiceWithCache(business repositories.BusinessRepository, authCache cache.AuthCache) *BusinessService {
	if authCache == nil {
		authCache = cache.NewNoopAuthCache()
	}
	return &BusinessService{business: business, authCache: authCache}
}

func (s *BusinessService) ApprovalSummary(user *models.User) (*models.ApprovalSummary, *AppError) {
	if !hasPermission(user, "tab.approval.read") {
		return nil, forbidden("当前账号无权查看审批数据")
	}
	value, err := s.business.ApprovalSummary(user)
	return wrapOK(value, err, "获取审批数据失败")
}

func (s *BusinessService) ListApprovalItems(user *models.User, scope string, status string, teamID string) ([]models.ApprovalItem, *AppError) {
	if !hasPermission(user, "tab.approval.read") {
		return nil, forbidden("当前账号无权查看审批数据")
	}
	if scope == "all" && !hasPermission(user, "tab.approval.all") {
		return nil, forbidden("当前账号无权查看全部审批")
	}
	value, err := s.business.ListApprovalItems(user, scope, status, teamID)
	return wrapOK(value, err, "获取审批列表失败")
}

func (s *BusinessService) GetApprovalItem(user *models.User, itemID string) (*models.ApprovalItem, *AppError) {
	if !hasPermission(user, "tab.approval.read") {
		return nil, forbidden("当前账号无权查看审批数据")
	}
	value, err := s.business.FindApprovalItem(user, itemID)
	return wrapBusinessResult(value, err, "审批记录不存在", "获取审批详情失败")
}

func (s *BusinessService) CreateApprovalItem(user *models.User, req models.CreateApprovalItemRequest) (*models.ApprovalItem, *AppError) {
	if !hasPermission(user, "tab.approval.create") {
		return nil, forbidden("当前账号无权发起审批")
	}
	if req.Title == "" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_REQUEST", "title 不可为空")
	}
	req.TeamID = targetTeamID(user, req.TeamID)
	if !policies.CanCreateApproval(user, req.TeamID) {
		return nil, forbidden("当前账号无权在该团队发起审批")
	}
	value, err := s.business.CreateApprovalItem(user, req)
	return wrapBusinessResult(value, err, "审批资源不存在", "创建审批失败")
}

func (s *BusinessService) ApproveItem(user *models.User, itemID string, comment string) (*models.ApprovalActionResponse, *AppError) {
	return s.updateApprovalStatus(user, itemID, "approved", comment)
}

func (s *BusinessService) RejectItem(user *models.User, itemID string, comment string) (*models.ApprovalActionResponse, *AppError) {
	return s.updateApprovalStatus(user, itemID, "rejected", comment)
}

func (s *BusinessService) updateApprovalStatus(user *models.User, itemID string, status string, comment string) (*models.ApprovalActionResponse, *AppError) {
	if !hasPermission(user, "tab.approval.approve") && !hasPermission(user, "tab.approval.all") {
		return nil, forbidden("当前账号无权操作审批数据")
	}
	current, err := s.business.FindApprovalItem(user, itemID)
	if appErr := mapRepoError(err, "审批记录不存在", "获取审批详情失败"); appErr != nil {
		return nil, appErr
	}
	if current.Status != "pending" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_APPROVAL_STATE", "当前状态不允许执行该操作")
	}
	if !policies.CanApproveTeamApproval(user, current.TeamID) {
		return nil, forbidden("当前账号无权限执行该操作")
	}
	item, err := s.business.UpdateApprovalStatus(user, itemID, status, comment)
	if appErr := mapRepoError(err, "审批记录不存在", "更新审批状态失败"); appErr != nil {
		return nil, appErr
	}
	return &models.ApprovalActionResponse{Success: true, ItemID: item.ID, Status: item.Status}, nil
}

func (s *BusinessService) CancelApprovalItem(user *models.User, itemID string) (*models.ApprovalActionResponse, *AppError) {
	if !hasPermission(user, "tab.approval.create") {
		return nil, forbidden("当前账号无权撤回审批")
	}
	current, err := s.business.FindApprovalItem(user, itemID)
	if appErr := mapRepoError(err, "审批记录不存在", "获取审批详情失败"); appErr != nil {
		return nil, appErr
	}
	if current.Status != "pending" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_APPROVAL_STATE", "当前状态不允许执行该操作")
	}
	if !policies.CanCancelApproval(user, current.ApplicantID, "") {
		return nil, forbidden("当前账号无权限执行该操作")
	}
	item, err := s.business.CancelApprovalItem(user, itemID)
	if appErr := mapRepoError(err, "审批记录不存在", "撤回审批失败"); appErr != nil {
		return nil, appErr
	}
	return &models.ApprovalActionResponse{Success: true, ItemID: item.ID, Status: item.Status}, nil
}

func (s *BusinessService) CalendarSummary(user *models.User) (*models.CalendarSummary, *AppError) {
	if !hasPermission(user, "tab.calendar.read") {
		return nil, forbidden("当前账号无权查看日程数据")
	}
	value, err := s.business.CalendarSummary(user)
	return wrapOK(value, err, "获取日程数据失败")
}

func (s *BusinessService) ListCalendarEvents(user *models.User, scope string, date string, teamID string) ([]models.CalendarEvent, *AppError) {
	if !hasPermission(user, "tab.calendar.read") {
		return nil, forbidden("当前账号无权查看日程数据")
	}
	if scope == "all" && !hasPermission(user, "tab.calendar.all") {
		return nil, forbidden("当前账号无权查看全部日程")
	}
	value, err := s.business.ListCalendarEvents(user, scope, date, teamID)
	return wrapOK(value, err, "获取日程列表失败")
}

func (s *BusinessService) GetCalendarEvent(user *models.User, eventID string) (*models.CalendarEvent, *AppError) {
	if !hasPermission(user, "tab.calendar.read") {
		return nil, forbidden("当前账号无权查看日程数据")
	}
	value, err := s.business.FindCalendarEvent(user, eventID)
	return wrapBusinessResult(value, err, "日程不存在", "获取日程详情失败")
}

func (s *BusinessService) CreateCalendarEvent(user *models.User, req models.CreateCalendarEventRequest) (*models.CreateCalendarEventResponse, *AppError) {
	if !hasPermission(user, "tab.calendar.create") && !hasPermission(user, "tab.calendar.manage") && !hasPermission(user, "tab.calendar.all") {
		return nil, forbidden("当前账号无权新增日程")
	}
	if req.Title == "" || req.StartTime == "" || req.EndTime == "" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_REQUEST", "title、startTime、endTime 不可为空")
	}
	req.TeamID = targetTeamID(user, req.TeamID)
	req.Visibility = defaultValue(req.Visibility, "team")
	if req.Visibility == "company" {
		if !policies.IsAdmin(user) {
			return nil, forbidden("当前账号无权新增全公司日程")
		}
	} else if !policies.CanManageCalendar(user, req.TeamID) {
		return nil, forbidden("当前账号无权新增该团队日程")
	}
	event, err := s.business.CreateCalendarEvent(user, req)
	if appErr := mapRepoError(err, "团队或用户不存在", "新增日程失败"); appErr != nil {
		return nil, appErr
	}
	return &models.CreateCalendarEventResponse{Success: true, EventID: event.ID}, nil
}

func (s *BusinessService) UpdateCalendarEvent(user *models.User, eventID string, req models.CreateCalendarEventRequest) (*models.CalendarEvent, *AppError) {
	if !hasPermission(user, "tab.calendar.manage") && !hasPermission(user, "tab.calendar.all") {
		return nil, forbidden("当前账号无权编辑日程")
	}
	current, err := s.business.FindCalendarEvent(user, eventID)
	if appErr := mapRepoError(err, "日程不存在", "获取日程详情失败"); appErr != nil {
		return nil, appErr
	}
	if !policies.CanManageCalendar(user, current.TeamID) {
		return nil, forbidden("当前账号无权限执行该操作")
	}
	req.TeamID = defaultValue(req.TeamID, current.TeamID)
	req.Visibility = defaultValue(req.Visibility, current.Visibility)
	if req.Visibility == "company" && !policies.IsAdmin(user) {
		return nil, forbidden("当前账号无权编辑为全公司日程")
	}
	value, err := s.business.UpdateCalendarEvent(user, eventID, req)
	return wrapBusinessResult(value, err, "日程不存在", "编辑日程失败")
}

func (s *BusinessService) DeleteCalendarEvent(user *models.User, eventID string) (models.SuccessResponse, *AppError) {
	if !hasPermission(user, "tab.calendar.manage") && !hasPermission(user, "tab.calendar.all") {
		return models.SuccessResponse{}, forbidden("当前账号无权删除日程")
	}
	current, err := s.business.FindCalendarEvent(user, eventID)
	if appErr := mapRepoError(err, "日程不存在", "获取日程详情失败"); appErr != nil {
		return models.SuccessResponse{}, appErr
	}
	if !policies.CanManageCalendar(user, current.TeamID) {
		return models.SuccessResponse{}, forbidden("当前账号无权限执行该操作")
	}
	if appErr := mapRepoError(s.business.DeleteCalendarEvent(user, eventID), "日程不存在", "删除日程失败"); appErr != nil {
		return models.SuccessResponse{}, appErr
	}
	return models.SuccessResponse{Success: true}, nil
}

func (s *BusinessService) ListAnnouncements(user *models.User, scope string, teamID string) ([]models.Announcement, *AppError) {
	if !hasPermission(user, "tab.announcement.read") {
		return nil, forbidden("当前账号无权查看公告")
	}
	value, err := s.business.ListAnnouncements(user, scope, teamID)
	return wrapOK(value, err, "获取公告列表失败")
}

func (s *BusinessService) GetAnnouncement(user *models.User, announcementID string) (*models.Announcement, *AppError) {
	if !hasPermission(user, "tab.announcement.read") {
		return nil, forbidden("当前账号无权查看公告")
	}
	value, err := s.business.FindAnnouncement(user, announcementID)
	return wrapBusinessResult(value, err, "公告不存在", "获取公告详情失败")
}

func (s *BusinessService) CreateAnnouncement(user *models.User, req models.AnnouncementRequest) (*models.Announcement, *AppError) {
	if !hasPermission(user, "tab.announcement.write") {
		return nil, forbidden("当前账号无权发布公告")
	}
	if req.Title == "" || req.Content == "" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_REQUEST", "title、content 不可为空")
	}
	req.Scope = defaultValue(req.Scope, "team")
	req.TeamID = targetTeamID(user, req.TeamID)
	if !policies.CanWriteAnnouncement(user, req.Scope, req.TeamID) {
		return nil, forbidden("当前账号无权发布该范围公告")
	}
	value, err := s.business.CreateAnnouncement(user, req)
	return wrapBusinessResult(value, err, "团队不存在", "发布公告失败")
}

func (s *BusinessService) UpdateAnnouncement(user *models.User, announcementID string, req models.AnnouncementRequest) (*models.Announcement, *AppError) {
	if !hasPermission(user, "tab.announcement.write") {
		return nil, forbidden("当前账号无权编辑公告")
	}
	current, err := s.business.FindAnnouncement(user, announcementID)
	if appErr := mapRepoError(err, "公告不存在", "获取公告详情失败"); appErr != nil {
		return nil, appErr
	}
	if !policies.CanManageAnnouncement(user, current.TeamID) {
		return nil, forbidden("当前账号无权限执行该操作")
	}
	value, err := s.business.UpdateAnnouncement(user, announcementID, req)
	return wrapBusinessResult(value, err, "公告不存在", "编辑公告失败")
}

func (s *BusinessService) DeleteAnnouncement(user *models.User, announcementID string) (models.SuccessResponse, *AppError) {
	if !hasPermission(user, "tab.announcement.write") {
		return models.SuccessResponse{}, forbidden("当前账号无权删除公告")
	}
	current, err := s.business.FindAnnouncement(user, announcementID)
	if appErr := mapRepoError(err, "公告不存在", "获取公告详情失败"); appErr != nil {
		return models.SuccessResponse{}, appErr
	}
	if !policies.CanManageAnnouncement(user, current.TeamID) {
		return models.SuccessResponse{}, forbidden("当前账号无权限执行该操作")
	}
	if appErr := mapRepoError(s.business.DeleteAnnouncement(user, announcementID), "公告不存在", "删除公告失败"); appErr != nil {
		return models.SuccessResponse{}, appErr
	}
	return models.SuccessResponse{Success: true}, nil
}

func (s *BusinessService) ListTeams(user *models.User) ([]models.TeamAdminItem, *AppError) {
	if !policies.CanManageTeam(user) {
		return nil, forbidden("当前账号无权管理团队")
	}
	value, err := s.business.ListTeams()
	return wrapOK(value, err, "获取团队列表失败")
}

func (s *BusinessService) CreateTeam(user *models.User, req models.TeamRequest) (*models.TeamAdminItem, *AppError) {
	if !policies.CanManageTeam(user) {
		return nil, forbidden("当前账号无权创建团队")
	}
	if req.TeamName == "" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_REQUEST", "teamName 不可为空")
	}
	value, err := s.business.CreateTeam(req)
	return wrapBusinessResult(value, err, "团队不存在", "创建团队失败")
}

func (s *BusinessService) UpdateTeam(user *models.User, teamID string, req models.TeamRequest) (*models.TeamAdminItem, *AppError) {
	if !policies.CanManageTeam(user) {
		return nil, forbidden("当前账号无权编辑团队")
	}
	if req.TeamName == "" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_REQUEST", "teamName 不可为空")
	}
	members := s.teamMemberUserIDs(teamID)
	value, err := s.business.UpdateTeam(teamID, req)
	if appErr := mapRepoError(err, "团队不存在", "编辑团队失败"); appErr != nil {
		return nil, appErr
	}
	s.invalidateUserContexts(members...)
	return value, nil
}

func (s *BusinessService) DisableTeam(user *models.User, teamID string) (models.SuccessResponse, *AppError) {
	if !policies.CanManageTeam(user) {
		return models.SuccessResponse{}, forbidden("当前账号无权停用团队")
	}
	members := s.teamMemberUserIDs(teamID)
	if appErr := mapRepoError(s.business.DisableTeam(teamID), "团队不存在", "停用团队失败"); appErr != nil {
		return models.SuccessResponse{}, appErr
	}
	s.invalidateUserContexts(members...)
	return models.SuccessResponse{Success: true}, nil
}

func (s *BusinessService) ListTeamMembers(user *models.User, teamID string) ([]models.TeamMemberItem, *AppError) {
	if !policies.CanReadTeamMembers(user) {
		return nil, forbidden("当前账号无权查看团队成员")
	}
	value, err := s.business.ListTeamMembers(teamID)
	return wrapOK(value, err, "获取团队成员失败")
}

func (s *BusinessService) AddTeamMember(user *models.User, teamID string, req models.TeamMemberMutationRequest) (*models.TeamMemberMutationResponse, *AppError) {
	if !policies.CanManageTeam(user) {
		return nil, forbidden("当前账号无权管理团队成员")
	}
	value, err := s.business.AddTeamMember(teamID, req)
	if appErr := mapRepoError(err, "团队或用户不存在", "添加团队成员失败"); appErr != nil {
		return nil, appErr
	}
	s.invalidateUserContexts(req.UserID)
	return value, nil
}

func (s *BusinessService) UpdateTeamMember(user *models.User, teamID string, targetUserID string, req models.TeamMemberMutationRequest) (*models.TeamMemberMutationResponse, *AppError) {
	if !policies.CanManageTeam(user) {
		return nil, forbidden("当前账号无权管理团队成员")
	}
	value, err := s.business.UpdateTeamMember(teamID, targetUserID, req)
	if appErr := mapRepoError(err, "团队成员不存在", "修改团队成员失败"); appErr != nil {
		return nil, appErr
	}
	s.invalidateUserContexts(targetUserID)
	return value, nil
}

func (s *BusinessService) RemoveTeamMember(user *models.User, teamID string, targetUserID string) (models.SuccessResponse, *AppError) {
	if !policies.CanManageTeam(user) {
		return models.SuccessResponse{}, forbidden("当前账号无权移出团队成员")
	}
	if appErr := mapRepoError(s.business.RemoveTeamMember(teamID, targetUserID), "团队成员不存在", "移出团队成员失败"); appErr != nil {
		return models.SuccessResponse{}, appErr
	}
	s.invalidateUserContexts(targetUserID)
	return models.SuccessResponse{Success: true}, nil
}

func (s *BusinessService) ListAdminUsers(user *models.User, teamID string, keyword string) ([]models.AdminUserItem, *AppError) {
	if !policies.CanManageTeam(user) {
		return nil, forbidden("当前账号无权查看用户列表")
	}
	value, err := s.business.ListAdminUsers(teamID, keyword)
	return wrapOK(value, err, "获取用户列表失败")
}

func (s *BusinessService) GetAdminUser(user *models.User, targetUserID string) (*models.AdminUserItem, *AppError) {
	if !policies.CanManageTeam(user) {
		return nil, forbidden("当前账号无权查看用户详情")
	}
	value, err := s.business.FindAdminUser(targetUserID)
	return wrapBusinessResult(value, err, "用户不存在", "获取用户详情失败")
}

func (s *BusinessService) UpdateUserGlobalRole(user *models.User, targetUserID string, req models.GlobalRoleRequest) (*models.AdminUserItem, *AppError) {
	if !policies.CanManageTeam(user) {
		return nil, forbidden("当前账号无权修改全局角色")
	}
	value, err := s.business.UpdateUserGlobalRole(targetUserID, req.GlobalRole)
	if appErr := mapRepoError(err, "用户不存在", "修改全局角色失败"); appErr != nil {
		return nil, appErr
	}
	s.invalidateUserContexts(targetUserID)
	return value, nil
}

func (s *BusinessService) teamMemberUserIDs(teamID string) []string {
	members, err := s.business.ListTeamMembers(teamID)
	if err != nil {
		return nil
	}
	userIDs := make([]string, 0, len(members))
	for _, member := range members {
		userIDs = append(userIDs, member.UserID)
	}
	return userIDs
}

func (s *BusinessService) invalidateUserContexts(userIDs ...string) {
	ctx := context.Background()
	for _, userID := range userIDs {
		if userID == "" {
			continue
		}
		_ = s.authCache.DeleteUserContext(ctx, userID)
	}
}

func forbidden(message string) *AppError {
	return NewAppError(http.StatusForbidden, "FORBIDDEN", message)
}

func targetTeamID(user *models.User, requestedTeamID string) string {
	if requestedTeamID != "" {
		return requestedTeamID
	}
	if user == nil {
		return ""
	}
	return user.CurrentTeamID
}

func defaultValue(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func wrapOK[T any](value T, err error, internalMessage string) (T, *AppError) {
	if err != nil {
		var zero T
		return zero, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", internalMessage)
	}
	return value, nil
}

func wrapBusinessResult[T any](value T, err error, notFoundMessage string, internalMessage string) (T, *AppError) {
	if appErr := mapRepoError(err, notFoundMessage, internalMessage); appErr != nil {
		var zero T
		return zero, appErr
	}
	return value, nil
}

func mapRepoError(err error, notFoundMessage string, internalMessage string) *AppError {
	if err == nil {
		return nil
	}
	if errors.Is(err, repositories.ErrNotFound) {
		return NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", notFoundMessage)
	}
	if errors.Is(err, repositories.ErrForbidden) {
		return NewAppError(http.StatusForbidden, "FORBIDDEN", "当前账号无权限执行该操作")
	}
	if errors.Is(err, repositories.ErrInvalidState) {
		return NewAppError(http.StatusBadRequest, "INVALID_APPROVAL_STATE", "当前状态不允许执行该操作")
	}
	if errors.Is(err, repositories.ErrInvalidRole) {
		return NewAppError(http.StatusBadRequest, "INVALID_TEAM_ROLE", "团队角色不合法")
	}
	return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", internalMessage)
}
