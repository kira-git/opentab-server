package repositories

import (
	"fmt"
	"strings"
	"time"

	"opentab-server/internal/models"
	"opentab-server/internal/policies"
)

type MemoryBusinessRepository struct{}

func NewMemoryBusinessRepository() *MemoryBusinessRepository {
	seedMemoryBusiness()
	return &MemoryBusinessRepository{}
}

var memoryApprovals []models.ApprovalItem
var memoryEvents []models.CalendarEvent
var memoryAnnouncements []models.Announcement
var memoryTeams []models.TeamAdminItem
var memoryAdminUsers []models.AdminUserItem

func (r *MemoryBusinessRepository) ApprovalSummary(user *models.User) (*models.ApprovalSummary, error) {
	items, _ := r.ListApprovalItems(user, "pending", "", "")
	return &models.ApprovalSummary{PendingCount: len(items), ApprovedToday: 0, Items: items}, nil
}

func (r *MemoryBusinessRepository) ListApprovalItems(user *models.User, scope string, status string, teamID string) ([]models.ApprovalItem, error) {
	result := []models.ApprovalItem{}
	for _, item := range memoryApprovals {
		if status != "" && status != "all" && item.Status != status {
			continue
		}
		if canSeeMemoryApproval(user, item, scope) {
			result = append(result, item)
		}
	}
	return result, nil
}

func (r *MemoryBusinessRepository) FindApprovalItem(user *models.User, itemID string) (*models.ApprovalItem, error) {
	for _, item := range memoryApprovals {
		if item.ID == itemID && canSeeMemoryApproval(user, item, "all") {
			return &item, nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryBusinessRepository) CreateApprovalItem(user *models.User, req models.CreateApprovalItemRequest) (*models.ApprovalItem, error) {
	teamID := valueOr(req.TeamID, user.CurrentTeamID)
	item := models.ApprovalItem{
		ID: fmt.Sprintf("apv-%d", time.Now().UnixNano()), TeamID: teamID, TeamName: memoryTeamName(teamID),
		Type: valueOr(req.Type, "general"), Title: req.Title, ApplicantID: user.ID, Applicant: user.DisplayName,
		ApproverID: memoryManagerID(teamID), Approver: memoryManagerName(teamID), Reason: req.Reason, Summary: req.Reason,
		Form: req.Form, Status: "pending", CreatedAt: time.Now().Format(time.RFC3339), UpdatedAt: time.Now().Format(time.RFC3339),
	}
	memoryApprovals = append(memoryApprovals, item)
	return &item, nil
}

func (r *MemoryBusinessRepository) UpdateApprovalStatus(user *models.User, itemID string, status string, comment string) (*models.ApprovalItem, error) {
	for i := range memoryApprovals {
		if memoryApprovals[i].ID == itemID {
			memoryApprovals[i].Status = status
			memoryApprovals[i].Comment = comment
			memoryApprovals[i].UpdatedAt = time.Now().Format(time.RFC3339)
			return &memoryApprovals[i], nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryBusinessRepository) CancelApprovalItem(user *models.User, itemID string) (*models.ApprovalItem, error) {
	for i := range memoryApprovals {
		if memoryApprovals[i].ID == itemID {
			memoryApprovals[i].Status = "cancelled"
			memoryApprovals[i].Comment = "发起人已撤回"
			memoryApprovals[i].UpdatedAt = time.Now().Format(time.RFC3339)
			return &memoryApprovals[i], nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryBusinessRepository) CalendarSummary(user *models.User) (*models.CalendarSummary, error) {
	events, _ := r.ListCalendarEvents(user, "visible", "", "")
	return &models.CalendarSummary{TodayCount: len(events), Events: events}, nil
}

func (r *MemoryBusinessRepository) ListCalendarEvents(user *models.User, scope string, date string, teamID string) ([]models.CalendarEvent, error) {
	result := []models.CalendarEvent{}
	for _, event := range memoryEvents {
		if date != "" && !strings.HasPrefix(event.StartTime, date) {
			continue
		}
		if policies.CanViewCalendar(user, event.Visibility, event.TeamID, event.CreatorID, event.ParticipantIDs) {
			result = append(result, event)
		}
	}
	return result, nil
}

func (r *MemoryBusinessRepository) FindCalendarEvent(user *models.User, eventID string) (*models.CalendarEvent, error) {
	for _, event := range memoryEvents {
		if event.ID == eventID {
			if !policies.CanViewCalendar(user, event.Visibility, event.TeamID, event.CreatorID, event.ParticipantIDs) {
				return nil, ErrNotFound
			}
			return &event, nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryBusinessRepository) CreateCalendarEvent(user *models.User, req models.CreateCalendarEventRequest) (*models.CalendarEvent, error) {
	event := calendarFromRequest(user, "", req)
	memoryEvents = append(memoryEvents, event)
	return &event, nil
}

func (r *MemoryBusinessRepository) UpdateCalendarEvent(user *models.User, eventID string, req models.CreateCalendarEventRequest) (*models.CalendarEvent, error) {
	for i := range memoryEvents {
		if memoryEvents[i].ID == eventID {
			memoryEvents[i] = calendarFromRequest(user, eventID, req)
			return &memoryEvents[i], nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryBusinessRepository) DeleteCalendarEvent(user *models.User, eventID string) error {
	for i := range memoryEvents {
		if memoryEvents[i].ID == eventID {
			memoryEvents = append(memoryEvents[:i], memoryEvents[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (r *MemoryBusinessRepository) ListAnnouncements(user *models.User, scope string, teamID string) ([]models.Announcement, error) {
	result := []models.Announcement{}
	for _, item := range memoryAnnouncements {
		if policies.CanViewAnnouncement(user, item.Scope, item.TeamID) {
			result = append(result, item)
		}
	}
	return result, nil
}

func (r *MemoryBusinessRepository) FindAnnouncement(user *models.User, announcementID string) (*models.Announcement, error) {
	for _, item := range memoryAnnouncements {
		if item.ID == announcementID {
			return &item, nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryBusinessRepository) CreateAnnouncement(user *models.User, req models.AnnouncementRequest) (*models.Announcement, error) {
	item := models.Announcement{
		ID: fmt.Sprintf("ann-%d", time.Now().UnixNano()), TeamID: valueOr(req.TeamID, user.CurrentTeamID), TeamName: memoryTeamName(valueOr(req.TeamID, user.CurrentTeamID)),
		Scope: valueOr(req.Scope, "team"), Title: req.Title, Content: req.Content, PublisherID: user.ID, PublisherName: user.DisplayName,
		Pinned: req.Pinned, CreatedAt: time.Now().Format(time.RFC3339), UpdatedAt: time.Now().Format(time.RFC3339),
	}
	memoryAnnouncements = append(memoryAnnouncements, item)
	return &item, nil
}

func (r *MemoryBusinessRepository) UpdateAnnouncement(user *models.User, announcementID string, req models.AnnouncementRequest) (*models.Announcement, error) {
	for i := range memoryAnnouncements {
		if memoryAnnouncements[i].ID == announcementID {
			memoryAnnouncements[i].Title = req.Title
			memoryAnnouncements[i].Content = req.Content
			memoryAnnouncements[i].Pinned = req.Pinned
			memoryAnnouncements[i].UpdatedAt = time.Now().Format(time.RFC3339)
			return &memoryAnnouncements[i], nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryBusinessRepository) DeleteAnnouncement(user *models.User, announcementID string) error {
	for i := range memoryAnnouncements {
		if memoryAnnouncements[i].ID == announcementID {
			memoryAnnouncements = append(memoryAnnouncements[:i], memoryAnnouncements[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (r *MemoryBusinessRepository) ListTeams() ([]models.TeamAdminItem, error) {
	return append([]models.TeamAdminItem{}, memoryTeams...), nil
}

func (r *MemoryBusinessRepository) CreateTeam(req models.TeamRequest) (*models.TeamAdminItem, error) {
	item := models.TeamAdminItem{
		TeamID:      fmt.Sprintf("team-%d", time.Now().UnixNano()),
		TeamName:    req.TeamName,
		Description: req.Description,
		Enabled:     true,
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}
	memoryTeams = append(memoryTeams, item)
	return &item, nil
}

func (r *MemoryBusinessRepository) UpdateTeam(teamID string, req models.TeamRequest) (*models.TeamAdminItem, error) {
	for i := range memoryTeams {
		if memoryTeams[i].TeamID == teamID {
			memoryTeams[i].TeamName = req.TeamName
			memoryTeams[i].Description = req.Description
			memoryTeams[i].UpdatedAt = time.Now().Format(time.RFC3339)
			return &memoryTeams[i], nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryBusinessRepository) DisableTeam(teamID string) error {
	for i := range memoryTeams {
		if memoryTeams[i].TeamID == teamID {
			memoryTeams[i].Enabled = false
			memoryTeams[i].UpdatedAt = time.Now().Format(time.RFC3339)
			return nil
		}
	}
	return ErrNotFound
}

func (r *MemoryBusinessRepository) ListTeamMembers(teamID string) ([]models.TeamMemberItem, error) {
	users, _ := r.ListAdminUsers(teamID, "")
	result := []models.TeamMemberItem{}
	for _, user := range users {
		for _, membership := range user.Memberships {
			if membership.TeamID == teamID {
				result = append(result, models.TeamMemberItem{UserID: user.UserID, Account: user.Account, DisplayName: user.DisplayName, TeamID: membership.TeamID, TeamName: membership.TeamName, TeamRole: membership.TeamRole, Enabled: true})
			}
		}
	}
	return result, nil
}

func (r *MemoryBusinessRepository) AddTeamMember(teamID string, req models.TeamMemberMutationRequest) (*models.TeamMemberMutationResponse, error) {
	if req.TeamRole != "manager" && req.TeamRole != "employee" {
		return nil, ErrInvalidRole
	}
	team := memoryTeamByID(teamID)
	if team == nil {
		return nil, ErrNotFound
	}
	for i := range memoryAdminUsers {
		if memoryAdminUsers[i].UserID != req.UserID {
			continue
		}
		memoryAdminUsers[i].Memberships = []models.TeamMembership{{
			TeamID:   teamID,
			TeamName: team.TeamName,
			TeamRole: req.TeamRole,
		}}
		return &models.TeamMemberMutationResponse{Success: true, TeamID: teamID, UserID: req.UserID, TeamRole: req.TeamRole}, nil
	}
	return nil, ErrNotFound
}

func (r *MemoryBusinessRepository) UpdateTeamMember(teamID string, userID string, req models.TeamMemberMutationRequest) (*models.TeamMemberMutationResponse, error) {
	if req.TeamRole != "manager" && req.TeamRole != "employee" {
		return nil, ErrInvalidRole
	}
	for i := range memoryAdminUsers {
		if memoryAdminUsers[i].UserID != userID {
			continue
		}
		for j := range memoryAdminUsers[i].Memberships {
			if memoryAdminUsers[i].Memberships[j].TeamID == teamID {
				memoryAdminUsers[i].Memberships[j].TeamRole = req.TeamRole
				return &models.TeamMemberMutationResponse{Success: true, TeamID: teamID, UserID: userID, TeamRole: req.TeamRole}, nil
			}
		}
		return nil, ErrNotFound
	}
	return nil, ErrNotFound
}

func (r *MemoryBusinessRepository) RemoveTeamMember(teamID string, userID string) error {
	for i := range memoryAdminUsers {
		if memoryAdminUsers[i].UserID != userID {
			continue
		}
		for j := range memoryAdminUsers[i].Memberships {
			if memoryAdminUsers[i].Memberships[j].TeamID == teamID {
				memoryAdminUsers[i].Memberships = append(memoryAdminUsers[i].Memberships[:j], memoryAdminUsers[i].Memberships[j+1:]...)
				return nil
			}
		}
		return ErrNotFound
	}
	return ErrNotFound
}

func memoryTeamByID(teamID string) *models.TeamAdminItem {
	for i := range memoryTeams {
		if memoryTeams[i].TeamID == teamID && memoryTeams[i].Enabled {
			return &memoryTeams[i]
		}
	}
	return nil
}

func (r *MemoryBusinessRepository) ListAdminUsers(teamID string, keyword string) ([]models.AdminUserItem, error) {
	result := append([]models.AdminUserItem{}, memoryAdminUsers...)
	if teamID == "" {
		return result, nil
	}
	filtered := []models.AdminUserItem{}
	for _, user := range result {
		for _, membership := range user.Memberships {
			if membership.TeamID == teamID {
				filtered = append(filtered, user)
			}
		}
	}
	return filtered, nil
}

func (r *MemoryBusinessRepository) FindAdminUser(userID string) (*models.AdminUserItem, error) {
	users, _ := r.ListAdminUsers("", "")
	for _, user := range users {
		if user.UserID == userID {
			return &user, nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryBusinessRepository) UpdateUserGlobalRole(userID string, globalRole *string) (*models.AdminUserItem, error) {
	for i := range memoryAdminUsers {
		if memoryAdminUsers[i].UserID == userID {
			memoryAdminUsers[i].GlobalRole = globalRole
			return &memoryAdminUsers[i], nil
		}
	}
	return nil, ErrNotFound
}

func seedMemoryBusiness() {
	now := "2026-06-03T09:00:00+08:00"
	memoryApprovals = []models.ApprovalItem{
		{ID: "apv-product-001", TeamID: "team-product", TeamName: "产品研发部", Type: "leave", Title: "周五下午请假", ApplicantID: "user-product-employee", Applicant: "陈磊", ApproverID: "user-product-manager", Approver: "刘洋", Status: "pending", Summary: "请假 0.5 天，已补充交接安排", CreatedAt: "2026-06-03T09:20:00+08:00", UpdatedAt: "2026-06-03T09:20:00+08:00"},
		{ID: "apv-operation-001", TeamID: "team-operation", TeamName: "运营支持部", Type: "expense", Title: "客户走访物料报销", ApplicantID: "user-operation-employee", Applicant: "李静", ApproverID: "user-operation-manager", Approver: "张敏", Status: "pending", Summary: "报销 320 元，客户走访物料", CreatedAt: "2026-06-03T10:05:00+08:00", UpdatedAt: "2026-06-03T10:05:00+08:00"},
		{ID: "apv-product-002", TeamID: "team-product", TeamName: "产品研发部", Type: "purchase", Title: "测试设备采购申请", ApplicantID: "user-product-employee", Applicant: "陈磊", ApproverID: "user-product-manager", Approver: "刘洋", Amount: 1299, Status: "approved", Summary: "采购一台测试机，预算 1299 元", Comment: "同意采购，注意登记资产编号", CreatedAt: "2026-06-02T15:40:00+08:00", UpdatedAt: "2026-06-02T16:10:00+08:00"},
	}
	memoryEvents = []models.CalendarEvent{
		{ID: "evt-product-001", TeamID: "team-product", TeamName: "产品研发部", Visibility: "team", CreatorID: "user-product-manager", CreatorName: "刘洋", Title: "产品研发部晨会", Description: "确认 Tab 注册、权限和 AI OnCall 联调进展", StartTime: "2026-06-03T09:30:00+08:00", EndTime: "2026-06-03T10:00:00+08:00", Location: "线上会议", Participants: []string{"刘洋", "陈磊"}, ParticipantIDs: []string{"user-product-manager", "user-product-employee"}},
		{ID: "evt-operation-001", TeamID: "team-operation", TeamName: "运营支持部", Visibility: "team", CreatorID: "user-operation-manager", CreatorName: "张敏", Title: "客户反馈整理", Description: "汇总近期客户对工作台 Tab 的反馈", StartTime: "2026-06-03T10:30:00+08:00", EndTime: "2026-06-03T11:00:00+08:00", Location: "会议室 A", Participants: []string{"张敏", "李静"}, ParticipantIDs: []string{"user-operation-manager", "user-operation-employee"}},
		{ID: "evt-product-002", TeamID: "team-product", TeamName: "产品研发部", Visibility: "team", CreatorID: "user-product-manager", CreatorName: "刘洋", Title: "Tab 容器联调复盘", Description: "检查客户端 Tab 列表、审批和日程数据展示", StartTime: "2026-06-03T14:00:00+08:00", EndTime: "2026-06-03T15:00:00+08:00", Location: "开发群语音", Participants: []string{"刘洋", "陈磊"}, ParticipantIDs: []string{"user-product-manager", "user-product-employee"}},
		{ID: "evt-operation-002", TeamID: "team-operation", TeamName: "运营支持部", Visibility: "team", CreatorID: "user-operation-manager", CreatorName: "张敏", Title: "公告发布确认", Description: "确认阶段演示公告内容和发布范围", StartTime: "2026-06-03T16:00:00+08:00", EndTime: "2026-06-03T16:40:00+08:00", Location: "会议室 B", Participants: []string{"张敏", "李静"}, ParticipantIDs: []string{"user-operation-manager", "user-operation-employee"}},
		{ID: "evt-company-001", Visibility: "company", CreatorID: "user-admin", CreatorName: "张伟", Title: "阶段演示彩排", Description: "开放式 Tab 容器与 AI OnCall 助理阶段演示", StartTime: "2026-06-04T15:30:00+08:00", EndTime: "2026-06-04T16:30:00+08:00", Location: "线上会议"},
	}
	memoryAnnouncements = []models.Announcement{
		{ID: "ann-company-001", Scope: "company", Title: "阶段演示安排", Content: "本周四 15:30 进行开放式 Tab 容器与 AI OnCall 助理阶段演示，请相关成员提前完成数据检查。", PublisherID: "user-admin", PublisherName: "张伟", Pinned: true, CreatedAt: now, UpdatedAt: now},
		{ID: "ann-product-001", TeamID: "team-product", TeamName: "产品研发部", Scope: "team", Title: "产品研发部联调提醒", Content: "请在今天 14:00 前确认 Tab 列表、审批中心和日程接口在客户端展示正常。", PublisherID: "user-product-manager", PublisherName: "刘洋", CreatedAt: now, UpdatedAt: now},
		{ID: "ann-operation-001", TeamID: "team-operation", TeamName: "运营支持部", Scope: "team", Title: "客户反馈整理", Content: "请在周三下班前整理客户反馈和常见问题，重点标注和工作台 Tab 相关的需求。", PublisherID: "user-operation-manager", PublisherName: "张敏", CreatedAt: now, UpdatedAt: now},
	}
	memoryTeams = []models.TeamAdminItem{
		{TeamID: "team-product", TeamName: "产品研发部", Description: "负责产品、客户端和服务端联调", MemberCount: 2, ManagerCount: 1, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{TeamID: "team-operation", TeamName: "运营支持部", Description: "负责运营支持和客户协同", MemberCount: 2, ManagerCount: 1, Enabled: true, CreatedAt: now, UpdatedAt: now},
	}
	memoryAdminUsers = []models.AdminUserItem{
		adminUser("user-admin", "admin", "张伟", "admin", nil),
		adminUser("user-product-manager", "product-manager", "刘洋", "", []models.TeamMembership{{TeamID: "team-product", TeamName: "产品研发部", TeamRole: "manager"}}),
		adminUser("user-product-employee", "product-employee", "陈磊", "", []models.TeamMembership{{TeamID: "team-product", TeamName: "产品研发部", TeamRole: "employee"}}),
		adminUser("user-operation-manager", "operation-manager", "张敏", "", []models.TeamMembership{{TeamID: "team-operation", TeamName: "运营支持部", TeamRole: "manager"}}),
		adminUser("user-operation-employee", "operation-employee", "李静", "", []models.TeamMembership{{TeamID: "team-operation", TeamName: "运营支持部", TeamRole: "employee"}}),
	}
}

func canSeeMemoryApproval(user *models.User, item models.ApprovalItem, scope string) bool {
	if policies.IsAdmin(user) {
		return true
	}
	if scope == "pending" {
		return item.TeamID == user.CurrentTeamID && item.Status == "pending" && policies.HasTeamRole(user, item.TeamID, "manager")
	}
	return policies.CanViewApproval(user, item.ApplicantID, "", item.TeamID)
}

func calendarFromRequest(user *models.User, id string, req models.CreateCalendarEventRequest) models.CalendarEvent {
	if id == "" {
		id = fmt.Sprintf("evt-%d", time.Now().UnixNano())
	}
	teamID := valueOr(req.TeamID, user.CurrentTeamID)
	return models.CalendarEvent{ID: id, TeamID: teamID, TeamName: memoryTeamName(teamID), Visibility: valueOr(req.Visibility, "team"), CreatorID: user.ID, CreatorName: user.DisplayName, Title: req.Title, Description: req.Description, StartTime: req.StartTime, EndTime: req.EndTime, Location: req.Location, ParticipantIDs: req.ParticipantIDs}
}

func adminUser(userID string, account string, displayName string, globalRole string, memberships []models.TeamMembership) models.AdminUserItem {
	var role *string
	if globalRole != "" {
		role = &globalRole
	}
	return models.AdminUserItem{UserID: userID, Account: account, DisplayName: displayName, GlobalRole: role, Memberships: memberships, Enabled: true}
}

func memoryTeamName(teamID string) string {
	if teamID == "team-operation" {
		return "运营支持部"
	}
	if teamID == "team-product" {
		return "产品研发部"
	}
	return ""
}

func memoryManagerID(teamID string) string {
	if teamID == "team-operation" {
		return "user-operation-manager"
	}
	return "user-product-manager"
}

func memoryManagerName(teamID string) string {
	if teamID == "team-operation" {
		return "张敏"
	}
	return "刘洋"
}

func valueOr(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
