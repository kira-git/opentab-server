package services

import (
	"testing"

	"opentab-server/internal/models"
	"opentab-server/internal/repositories"
)

type fakeBusinessRepository struct {
	approval     models.ApprovalItem
	calendar     models.CalendarEvent
	announcement models.Announcement
	members      []models.TeamMemberItem
	updated      bool
	deleted      bool
}

func (r *fakeBusinessRepository) ApprovalSummary(user *models.User) (*models.ApprovalSummary, error) {
	return &models.ApprovalSummary{}, nil
}

func (r *fakeBusinessRepository) ListApprovalItems(user *models.User, scope string, status string, teamID string) ([]models.ApprovalItem, error) {
	return nil, nil
}

func (r *fakeBusinessRepository) FindApprovalItem(user *models.User, itemID string) (*models.ApprovalItem, error) {
	if r.approval.ID == itemID {
		return &r.approval, nil
	}
	return nil, repositories.ErrNotFound
}

func (r *fakeBusinessRepository) CreateApprovalItem(user *models.User, req models.CreateApprovalItemRequest) (*models.ApprovalItem, error) {
	item := models.ApprovalItem{ID: "apv-new", TeamID: req.TeamID, ApplicantID: user.ID, Status: "pending"}
	return &item, nil
}

func (r *fakeBusinessRepository) UpdateApprovalStatus(user *models.User, itemID string, status string, comment string) (*models.ApprovalItem, error) {
	r.updated = true
	r.approval.Status = status
	r.approval.Comment = comment
	return &r.approval, nil
}

func (r *fakeBusinessRepository) CancelApprovalItem(user *models.User, itemID string) (*models.ApprovalItem, error) {
	r.updated = true
	r.approval.Status = "cancelled"
	return &r.approval, nil
}

func (r *fakeBusinessRepository) CalendarSummary(user *models.User) (*models.CalendarSummary, error) {
	return &models.CalendarSummary{}, nil
}

func (r *fakeBusinessRepository) ListCalendarEvents(user *models.User, scope string, date string, teamID string) ([]models.CalendarEvent, error) {
	return nil, nil
}

func (r *fakeBusinessRepository) FindCalendarEvent(user *models.User, eventID string) (*models.CalendarEvent, error) {
	if r.calendar.ID == eventID {
		return &r.calendar, nil
	}
	return nil, repositories.ErrNotFound
}

func (r *fakeBusinessRepository) CreateCalendarEvent(user *models.User, req models.CreateCalendarEventRequest) (*models.CalendarEvent, error) {
	event := models.CalendarEvent{ID: "evt-new", TeamID: req.TeamID, Visibility: req.Visibility}
	return &event, nil
}

func (r *fakeBusinessRepository) UpdateCalendarEvent(user *models.User, eventID string, req models.CreateCalendarEventRequest) (*models.CalendarEvent, error) {
	r.updated = true
	r.calendar.TeamID = req.TeamID
	r.calendar.Visibility = req.Visibility
	return &r.calendar, nil
}

func (r *fakeBusinessRepository) DeleteCalendarEvent(user *models.User, eventID string) error {
	r.deleted = true
	return nil
}

func (r *fakeBusinessRepository) ListAnnouncements(user *models.User, scope string, teamID string) ([]models.Announcement, error) {
	return nil, nil
}

func (r *fakeBusinessRepository) FindAnnouncement(user *models.User, announcementID string) (*models.Announcement, error) {
	if r.announcement.ID == announcementID {
		return &r.announcement, nil
	}
	return nil, repositories.ErrNotFound
}

func (r *fakeBusinessRepository) CreateAnnouncement(user *models.User, req models.AnnouncementRequest) (*models.Announcement, error) {
	item := models.Announcement{ID: "ann-new", TeamID: req.TeamID, Scope: req.Scope}
	return &item, nil
}

func (r *fakeBusinessRepository) UpdateAnnouncement(user *models.User, announcementID string, req models.AnnouncementRequest) (*models.Announcement, error) {
	r.updated = true
	return &r.announcement, nil
}

func (r *fakeBusinessRepository) DeleteAnnouncement(user *models.User, announcementID string) error {
	r.deleted = true
	return nil
}

func (r *fakeBusinessRepository) ListTeams() ([]models.TeamAdminItem, error) {
	return nil, nil
}

func (r *fakeBusinessRepository) CreateTeam(req models.TeamRequest) (*models.TeamAdminItem, error) {
	return &models.TeamAdminItem{}, nil
}

func (r *fakeBusinessRepository) UpdateTeam(teamID string, req models.TeamRequest) (*models.TeamAdminItem, error) {
	return &models.TeamAdminItem{}, nil
}

func (r *fakeBusinessRepository) DisableTeam(teamID string) error {
	return nil
}

func (r *fakeBusinessRepository) ListTeamMembers(teamID string) ([]models.TeamMemberItem, error) {
	return r.members, nil
}

func (r *fakeBusinessRepository) AddTeamMember(teamID string, req models.TeamMemberMutationRequest) (*models.TeamMemberMutationResponse, error) {
	return &models.TeamMemberMutationResponse{}, nil
}

func (r *fakeBusinessRepository) UpdateTeamMember(teamID string, userID string, req models.TeamMemberMutationRequest) (*models.TeamMemberMutationResponse, error) {
	return &models.TeamMemberMutationResponse{}, nil
}

func (r *fakeBusinessRepository) RemoveTeamMember(teamID string, userID string) error {
	return nil
}

func (r *fakeBusinessRepository) ListAdminUsers(teamID string, keyword string) ([]models.AdminUserItem, error) {
	return nil, nil
}

func (r *fakeBusinessRepository) FindAdminUser(userID string) (*models.AdminUserItem, error) {
	return &models.AdminUserItem{}, nil
}

func (r *fakeBusinessRepository) UpdateUserGlobalRole(userID string, globalRole *string) (*models.AdminUserItem, error) {
	return &models.AdminUserItem{}, nil
}

func TestBusinessServiceRejectsApprovalWhenNotPendingBeforeRepositoryUpdate(t *testing.T) {
	repo := &fakeBusinessRepository{approval: models.ApprovalItem{ID: "apv-1", TeamID: "team-product", Status: "approved"}}
	service := NewBusinessService(repo)
	user := productManagerUser()

	_, err := service.ApproveItem(user, "apv-1", "ok")
	if err == nil || err.Code != "INVALID_APPROVAL_STATE" {
		t.Fatalf("expected INVALID_APPROVAL_STATE, got %+v", err)
	}
	if repo.updated {
		t.Fatalf("repository update should not be called for invalid state")
	}
}

func TestBusinessServiceRejectsCalendarCreateForEmployeeTeamEvent(t *testing.T) {
	repo := &fakeBusinessRepository{}
	service := NewBusinessService(repo)
	user := productEmployeeUser()

	_, err := service.CreateCalendarEvent(user, models.CreateCalendarEventRequest{
		Title: "团队会议", StartTime: "2026-06-03T10:00:00+08:00", EndTime: "2026-06-03T11:00:00+08:00", Visibility: "team",
	})
	if err == nil || err.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %+v", err)
	}
}

func TestBusinessServiceRejectsCompanyAnnouncementForManager(t *testing.T) {
	repo := &fakeBusinessRepository{}
	service := NewBusinessService(repo)
	user := productManagerUser()

	_, err := service.CreateAnnouncement(user, models.AnnouncementRequest{Scope: "company", Title: "公告", Content: "内容"})
	if err == nil || err.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %+v", err)
	}
}

func TestBusinessServiceInvalidatesUserContextAfterTeamMemberUpdate(t *testing.T) {
	repo := &fakeBusinessRepository{}
	authCache := newMemoryAuthCache()
	authCache.users["user-target"] = models.User{ID: "user-target", DisplayName: "Target"}
	service := NewBusinessServiceWithCache(repo, authCache)

	_, err := service.UpdateTeamMember(adminUser(), "team-product", "user-target", models.TeamMemberMutationRequest{TeamRole: "manager"})
	if err != nil {
		t.Fatalf("update team member failed: %+v", err)
	}
	if _, ok := authCache.users["user-target"]; ok {
		t.Fatalf("expected target user context cache to be invalidated")
	}
}

func TestBusinessServiceInvalidatesTeamUsersAfterTeamUpdate(t *testing.T) {
	repo := &fakeBusinessRepository{members: []models.TeamMemberItem{
		{UserID: "user-a"},
		{UserID: "user-b"},
	}}
	authCache := newMemoryAuthCache()
	authCache.users["user-a"] = models.User{ID: "user-a"}
	authCache.users["user-b"] = models.User{ID: "user-b"}
	service := NewBusinessServiceWithCache(repo, authCache)

	_, err := service.UpdateTeam(adminUser(), "team-product", models.TeamRequest{TeamName: "新团队"})
	if err != nil {
		t.Fatalf("update team failed: %+v", err)
	}
	if _, ok := authCache.users["user-a"]; ok {
		t.Fatalf("expected user-a cache to be invalidated")
	}
	if _, ok := authCache.users["user-b"]; ok {
		t.Fatalf("expected user-b cache to be invalidated")
	}
}

func productManagerUser() *models.User {
	return &models.User{
		ID:            "user-product-manager",
		DisplayName:   "产品主管",
		CurrentTeamID: "team-product",
		Memberships:   []models.TeamMembership{{TeamID: "team-product", TeamRole: "manager"}},
		Permissions:   []string{"tab.approval.read", "tab.approval.approve", "tab.calendar.create", "tab.calendar.manage", "tab.announcement.write"},
		Enabled:       true,
	}
}

func productEmployeeUser() *models.User {
	return &models.User{
		ID:            "user-product-employee",
		DisplayName:   "产品员工",
		CurrentTeamID: "team-product",
		Memberships:   []models.TeamMembership{{TeamID: "team-product", TeamRole: "employee"}},
		Permissions:   []string{"tab.calendar.create"},
		Enabled:       true,
	}
}

func adminUser() *models.User {
	return &models.User{
		ID:            "user-admin",
		DisplayName:   "管理员",
		GlobalRole:    "admin",
		CurrentTeamID: "team-product",
		Memberships:   []models.TeamMembership{{TeamID: "team-product", TeamRole: "manager"}},
		Permissions:   []string{"team.manage"},
		Enabled:       true,
	}
}
