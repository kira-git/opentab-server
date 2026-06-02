package repositories

import "opentab-server/internal/models"

type BusinessRepository interface {
	ApprovalSummary(user *models.User) (*models.ApprovalSummary, error)
	ListApprovalItems(user *models.User, scope string, status string, teamID string) ([]models.ApprovalItem, error)
	FindApprovalItem(user *models.User, itemID string) (*models.ApprovalItem, error)
	CreateApprovalItem(user *models.User, req models.CreateApprovalItemRequest) (*models.ApprovalItem, error)
	UpdateApprovalStatus(user *models.User, itemID string, status string, comment string) (*models.ApprovalItem, error)
	CancelApprovalItem(user *models.User, itemID string) (*models.ApprovalItem, error)
	CalendarSummary(user *models.User) (*models.CalendarSummary, error)
	ListCalendarEvents(user *models.User, scope string, date string, teamID string) ([]models.CalendarEvent, error)
	FindCalendarEvent(user *models.User, eventID string) (*models.CalendarEvent, error)
	CreateCalendarEvent(user *models.User, req models.CreateCalendarEventRequest) (*models.CalendarEvent, error)
	UpdateCalendarEvent(user *models.User, eventID string, req models.CreateCalendarEventRequest) (*models.CalendarEvent, error)
	DeleteCalendarEvent(user *models.User, eventID string) error
	ListAnnouncements(user *models.User, scope string, teamID string) ([]models.Announcement, error)
	FindAnnouncement(user *models.User, announcementID string) (*models.Announcement, error)
	CreateAnnouncement(user *models.User, req models.AnnouncementRequest) (*models.Announcement, error)
	UpdateAnnouncement(user *models.User, announcementID string, req models.AnnouncementRequest) (*models.Announcement, error)
	DeleteAnnouncement(user *models.User, announcementID string) error
	ListTeams() ([]models.TeamAdminItem, error)
	CreateTeam(req models.TeamRequest) (*models.TeamAdminItem, error)
	UpdateTeam(teamID string, req models.TeamRequest) (*models.TeamAdminItem, error)
	DisableTeam(teamID string) error
	ListTeamMembers(teamID string) ([]models.TeamMemberItem, error)
	AddTeamMember(teamID string, req models.TeamMemberMutationRequest) (*models.TeamMemberMutationResponse, error)
	UpdateTeamMember(teamID string, userID string, req models.TeamMemberMutationRequest) (*models.TeamMemberMutationResponse, error)
	RemoveTeamMember(teamID string, userID string) error
	ListAdminUsers(teamID string, keyword string) ([]models.AdminUserItem, error)
	FindAdminUser(userID string) (*models.AdminUserItem, error)
	UpdateUserGlobalRole(userID string, globalRole *string) (*models.AdminUserItem, error)
}
