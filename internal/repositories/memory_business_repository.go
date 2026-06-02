package repositories

import (
	"fmt"
	"strings"
	"time"

	"opentab-server/internal/models"
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
			if user.GlobalRole != "admin" && !memoryHasRole(user, memoryApprovals[i].TeamID, "manager") {
				return nil, ErrForbidden
			}
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
			if memoryApprovals[i].Status != "pending" {
				return nil, ErrInvalidState
			}
			if memoryApprovals[i].ApplicantID != "" && memoryApprovals[i].ApplicantID != user.ID {
				return nil, ErrForbidden
			}
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
		if user.GlobalRole == "admin" || event.Visibility == "company" || event.TeamID == user.CurrentTeamID || event.CreatorID == user.ID || stringSliceContains(event.ParticipantIDs, user.ID) {
			result = append(result, event)
		}
	}
	return result, nil
}

func (r *MemoryBusinessRepository) FindCalendarEvent(user *models.User, eventID string) (*models.CalendarEvent, error) {
	for _, event := range memoryEvents {
		if event.ID == eventID {
			if user.GlobalRole != "admin" && event.Visibility != "company" && event.TeamID != user.CurrentTeamID && event.CreatorID != user.ID && !stringSliceContains(event.ParticipantIDs, user.ID) {
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
		if user.GlobalRole == "admin" || item.Scope == "company" || item.TeamID == user.CurrentTeamID {
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
	return &models.TeamMemberMutationResponse{Success: true, TeamID: teamID, UserID: req.UserID, TeamRole: req.TeamRole}, nil
}

func (r *MemoryBusinessRepository) UpdateTeamMember(teamID string, userID string, req models.TeamMemberMutationRequest) (*models.TeamMemberMutationResponse, error) {
	return &models.TeamMemberMutationResponse{Success: true, TeamID: teamID, UserID: userID, TeamRole: req.TeamRole}, nil
}

func (r *MemoryBusinessRepository) RemoveTeamMember(teamID string, userID string) error {
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
	now := time.Now().Format(time.RFC3339)
	memoryApprovals = []models.ApprovalItem{
		{ID: "apv-product-001", TeamID: "team-product", TeamName: "产品研发部", Type: "leave", Title: "请假申请", ApplicantID: "user-product-employee", Applicant: "产品员工", ApproverID: "user-product-manager", Approver: "产品主管", Status: "pending", Summary: "请假 1 天", CreatedAt: now, UpdatedAt: now},
		{ID: "apv-operation-001", TeamID: "team-operation", TeamName: "运营支持部", Type: "expense", Title: "活动物料报销", ApplicantID: "user-operation-employee", Applicant: "运营员工", ApproverID: "user-operation-manager", Approver: "运营主管", Status: "pending", Summary: "报销 320 元", CreatedAt: now, UpdatedAt: now},
	}
	memoryEvents = []models.CalendarEvent{
		{ID: "evt-product-001", TeamID: "team-product", TeamName: "产品研发部", Visibility: "team", CreatorID: "user-product-manager", CreatorName: "产品主管", Title: "产品研发部周会", StartTime: now, EndTime: now, ParticipantIDs: []string{"user-product-employee"}},
		{ID: "evt-operation-001", TeamID: "team-operation", TeamName: "运营支持部", Visibility: "team", CreatorID: "user-operation-manager", CreatorName: "运营主管", Title: "运营支持部周会", StartTime: now, EndTime: now, ParticipantIDs: []string{"user-operation-employee"}},
		{ID: "evt-company-001", Visibility: "company", CreatorID: "user-admin", CreatorName: "系统管理员", Title: "全公司阶段同步", StartTime: now, EndTime: now},
	}
	memoryAnnouncements = []models.Announcement{
		{ID: "ann-company-001", Scope: "company", Title: "全公司阶段同步", Content: "本周进行阶段演示。", PublisherID: "user-admin", PublisherName: "系统管理员", Pinned: true, CreatedAt: now, UpdatedAt: now},
		{ID: "ann-product-001", TeamID: "team-product", TeamName: "产品研发部", Scope: "team", Title: "产品研发部周五分享", Content: "周五 16:00 分享联调进展。", PublisherID: "user-product-manager", PublisherName: "产品主管", CreatedAt: now, UpdatedAt: now},
	}
	memoryTeams = []models.TeamAdminItem{
		{TeamID: "team-product", TeamName: "产品研发部", Description: "负责产品、客户端和服务端联调", MemberCount: 2, ManagerCount: 1, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{TeamID: "team-operation", TeamName: "运营支持部", Description: "负责运营支持和客户协同", MemberCount: 2, ManagerCount: 1, Enabled: true, CreatedAt: now, UpdatedAt: now},
	}
	memoryAdminUsers = []models.AdminUserItem{
		adminUser("user-admin", "admin", "系统管理员", "admin", nil),
		adminUser("user-product-manager", "product-manager", "产品主管", "", []models.TeamMembership{{TeamID: "team-product", TeamName: "产品研发部", TeamRole: "manager"}}),
		adminUser("user-product-employee", "product-employee", "产品员工", "", []models.TeamMembership{{TeamID: "team-product", TeamName: "产品研发部", TeamRole: "employee"}}),
		adminUser("user-operation-manager", "operation-manager", "运营主管", "", []models.TeamMembership{{TeamID: "team-operation", TeamName: "运营支持部", TeamRole: "manager"}}),
		adminUser("user-operation-employee", "operation-employee", "运营员工", "", []models.TeamMembership{{TeamID: "team-operation", TeamName: "运营支持部", TeamRole: "employee"}}),
	}
}

func canSeeMemoryApproval(user *models.User, item models.ApprovalItem, scope string) bool {
	if user.GlobalRole == "admin" {
		return true
	}
	if scope == "pending" {
		return item.TeamID == user.CurrentTeamID && item.Status == "pending" && memoryHasRole(user, item.TeamID, "manager")
	}
	return item.ApplicantID == user.ID || memoryHasRole(user, item.TeamID, "manager")
}

func memoryHasRole(user *models.User, teamID string, role string) bool {
	for _, membership := range user.Memberships {
		if membership.TeamID == teamID && membership.TeamRole == role {
			return true
		}
	}
	return false
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
		return "运营主管"
	}
	return "产品主管"
}

func valueOr(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func stringSliceContains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
