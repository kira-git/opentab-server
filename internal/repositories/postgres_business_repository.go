package repositories

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"opentab-server/internal/database"
	"opentab-server/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostgresBusinessRepository struct {
	db *gorm.DB
}

func NewPostgresBusinessRepository(db *gorm.DB) *PostgresBusinessRepository {
	return &PostgresBusinessRepository{db: db}
}

func (r *PostgresBusinessRepository) ApprovalSummary(user *models.User) (*models.ApprovalSummary, error) {
	items, err := r.ListApprovalItems(user, "pending", "", "")
	if err != nil {
		return nil, err
	}
	allVisible, err := r.ListApprovalItems(user, "mine", "", "")
	if err != nil {
		return nil, err
	}
	return &models.ApprovalSummary{PendingCount: len(items), ApprovedToday: countApprovedToday(allVisible), Items: items}, nil
}

func (r *PostgresBusinessRepository) ListApprovalItems(user *models.User, scope string, status string, teamID string) ([]models.ApprovalItem, error) {
	if scope == "" {
		scope = approvalScopeFromStatus(status)
	}
	query := r.db.Model(&database.ApprovalItemRecord{}).Order("created_at DESC")
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	query = r.applyApprovalVisibility(query, user, scope, teamID)
	var records []database.ApprovalItemRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	return r.approvalRecordsToModels(records)
}

func (r *PostgresBusinessRepository) FindApprovalItem(user *models.User, itemID string) (*models.ApprovalItem, error) {
	var record database.ApprovalItemRecord
	if err := r.db.Where("id = ?", itemID).First(&record).Error; err != nil {
		return nil, mapGormError(err)
	}
	if !canViewApproval(user, record) {
		return nil, ErrNotFound
	}
	item, err := r.approvalRecordToModel(record)
	return &item, err
}

func (r *PostgresBusinessRepository) CreateApprovalItem(user *models.User, req models.CreateApprovalItemRequest) (*models.ApprovalItem, error) {
	teamID := req.TeamID
	if teamID == "" {
		teamID = user.CurrentTeamID
	}
	if teamID == "" || (!isAdmin(user) && !userInTeam(user, teamID)) {
		return nil, ErrForbidden
	}
	approverID, approverName, err := r.firstTeamManager(teamID)
	if err != nil {
		return nil, err
	}
	formJSON, _ := json.Marshal(req.Form)
	record := database.ApprovalItemRecord{
		ID:          fmt.Sprintf("apv-%d", time.Now().UnixNano()),
		UserID:      user.ID,
		TeamID:      teamID,
		Type:        valueOrDefault(req.Type, "general"),
		Title:       strings.TrimSpace(req.Title),
		ApplicantID: user.ID,
		Applicant:   user.DisplayName,
		ApproverID:  approverID,
		Approver:    approverName,
		Reason:      req.Reason,
		Summary:     req.Reason,
		FormJSON:    datatypes.JSON(formJSON),
		Status:      "pending",
	}
	if err := r.db.Create(&record).Error; err != nil {
		return nil, err
	}
	item, err := r.approvalRecordToModel(record)
	return &item, err
}

func (r *PostgresBusinessRepository) UpdateApprovalStatus(user *models.User, itemID string, status string, comment string) (*models.ApprovalItem, error) {
	var record database.ApprovalItemRecord
	if err := r.db.Where("id = ?", itemID).First(&record).Error; err != nil {
		return nil, mapGormError(err)
	}
	if record.Status != "pending" {
		return nil, ErrInvalidState
	}
	if !isAdmin(user) && !userHasTeamRole(user, record.TeamID, "manager") {
		return nil, ErrForbidden
	}
	record.Status = status
	record.Comment = comment
	if err := r.db.Save(&record).Error; err != nil {
		return nil, err
	}
	item, err := r.approvalRecordToModel(record)
	return &item, err
}

func (r *PostgresBusinessRepository) CancelApprovalItem(user *models.User, itemID string) (*models.ApprovalItem, error) {
	var record database.ApprovalItemRecord
	if err := r.db.Where("id = ?", itemID).First(&record).Error; err != nil {
		return nil, mapGormError(err)
	}
	if record.Status != "pending" {
		return nil, ErrInvalidState
	}
	if record.ApplicantID != user.ID && record.UserID != user.ID {
		return nil, ErrForbidden
	}
	record.Status = "cancelled"
	record.Comment = "发起人已撤回"
	if err := r.db.Save(&record).Error; err != nil {
		return nil, err
	}
	item, err := r.approvalRecordToModel(record)
	return &item, err
}

func (r *PostgresBusinessRepository) CalendarSummary(user *models.User) (*models.CalendarSummary, error) {
	events, err := r.ListCalendarEvents(user, "visible", time.Now().Format("2006-01-02"), "")
	if err != nil {
		return nil, err
	}
	return &models.CalendarSummary{TodayCount: len(events), Events: events}, nil
}

func (r *PostgresBusinessRepository) ListCalendarEvents(user *models.User, scope string, date string, teamID string) ([]models.CalendarEvent, error) {
	if scope == "" {
		scope = "visible"
	}
	query := r.db.Model(&database.CalendarEventRecord{}).Order("start_time ASC")
	if date != "" {
		if start, err := time.Parse("2006-01-02", date); err == nil {
			query = query.Where("start_time >= ? AND start_time < ?", start, start.AddDate(0, 0, 1))
		}
	}
	query = r.applyCalendarVisibility(query, user, scope, teamID)
	var records []database.CalendarEventRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	return r.calendarRecordsToModels(records)
}

func (r *PostgresBusinessRepository) FindCalendarEvent(user *models.User, eventID string) (*models.CalendarEvent, error) {
	var record database.CalendarEventRecord
	if err := r.db.Where("id = ?", eventID).First(&record).Error; err != nil {
		return nil, mapGormError(err)
	}
	if !canViewCalendar(user, record) {
		return nil, ErrNotFound
	}
	event, err := r.calendarRecordToModel(record)
	return &event, err
}

func (r *PostgresBusinessRepository) CreateCalendarEvent(user *models.User, req models.CreateCalendarEventRequest) (*models.CalendarEvent, error) {
	teamID := req.TeamID
	if teamID == "" {
		teamID = user.CurrentTeamID
	}
	visibility := valueOrDefault(req.Visibility, "team")
	if visibility == "company" && !isAdmin(user) {
		return nil, ErrForbidden
	}
	if visibility != "company" && (!isAdmin(user) && !userHasTeamRole(user, teamID, "manager")) {
		return nil, ErrForbidden
	}
	record := calendarRequestToRecord(user, "", teamID, visibility, req)
	if err := r.db.Create(&record).Error; err != nil {
		return nil, err
	}
	event, err := r.calendarRecordToModel(record)
	return &event, err
}

func (r *PostgresBusinessRepository) UpdateCalendarEvent(user *models.User, eventID string, req models.CreateCalendarEventRequest) (*models.CalendarEvent, error) {
	var record database.CalendarEventRecord
	if err := r.db.Where("id = ?", eventID).First(&record).Error; err != nil {
		return nil, mapGormError(err)
	}
	if !canManageCalendar(user, record) {
		return nil, ErrForbidden
	}
	updated := calendarRequestToRecord(user, record.ID, valueOrDefault(req.TeamID, record.TeamID), valueOrDefault(req.Visibility, record.Visibility), req)
	record.Title = updated.Title
	record.Description = updated.Description
	record.StartTime = updated.StartTime
	record.EndTime = updated.EndTime
	record.Location = updated.Location
	record.TeamID = updated.TeamID
	record.Visibility = updated.Visibility
	record.ParticipantsJSON = updated.ParticipantsJSON
	record.ParticipantIDsJSON = updated.ParticipantIDsJSON
	if err := r.db.Save(&record).Error; err != nil {
		return nil, err
	}
	event, err := r.calendarRecordToModel(record)
	return &event, err
}

func (r *PostgresBusinessRepository) DeleteCalendarEvent(user *models.User, eventID string) error {
	var record database.CalendarEventRecord
	if err := r.db.Where("id = ?", eventID).First(&record).Error; err != nil {
		return mapGormError(err)
	}
	if !canManageCalendar(user, record) {
		return ErrForbidden
	}
	return r.db.Delete(&record).Error
}

func (r *PostgresBusinessRepository) ListAnnouncements(user *models.User, scope string, teamID string) ([]models.Announcement, error) {
	if scope == "" {
		scope = "visible"
	}
	query := r.db.Model(&database.AnnouncementRecord{}).Where("deleted_at IS NULL").Order("pinned DESC, created_at DESC")
	if scope == "all" {
		if !isAdmin(user) {
			return nil, ErrForbidden
		}
	} else if scope == "team" {
		targetTeamID := valueOrDefault(teamID, user.CurrentTeamID)
		if targetTeamID == "" || (!isAdmin(user) && !userInTeam(user, targetTeamID)) {
			return nil, ErrForbidden
		}
		query = query.Where("scope = 'team' AND team_id = ?", targetTeamID)
	} else {
		if isAdmin(user) {
			query = query.Where("scope = 'company' OR scope = 'team'")
		} else {
			query = query.Where("scope = 'company' OR (scope = 'team' AND team_id = ?)", user.CurrentTeamID)
		}
	}
	var records []database.AnnouncementRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	return r.announcementRecordsToModels(records)
}

func (r *PostgresBusinessRepository) FindAnnouncement(user *models.User, announcementID string) (*models.Announcement, error) {
	var record database.AnnouncementRecord
	if err := r.db.Where("id = ? AND deleted_at IS NULL", announcementID).First(&record).Error; err != nil {
		return nil, mapGormError(err)
	}
	if !canViewAnnouncement(user, record) {
		return nil, ErrNotFound
	}
	item, err := r.announcementRecordToModel(record)
	return &item, err
}

func (r *PostgresBusinessRepository) CreateAnnouncement(user *models.User, req models.AnnouncementRequest) (*models.Announcement, error) {
	if !canWriteAnnouncement(user, req.Scope, req.TeamID) {
		return nil, ErrForbidden
	}
	record := database.AnnouncementRecord{
		ID:            fmt.Sprintf("ann-%d", time.Now().UnixNano()),
		TeamID:        req.TeamID,
		Scope:         valueOrDefault(req.Scope, "team"),
		Title:         strings.TrimSpace(req.Title),
		Content:       strings.TrimSpace(req.Content),
		PublisherID:   user.ID,
		PublisherName: user.DisplayName,
		Pinned:        req.Pinned,
	}
	if record.Scope == "team" && record.TeamID == "" {
		record.TeamID = user.CurrentTeamID
	}
	if err := r.db.Create(&record).Error; err != nil {
		return nil, err
	}
	item, err := r.announcementRecordToModel(record)
	return &item, err
}

func (r *PostgresBusinessRepository) UpdateAnnouncement(user *models.User, announcementID string, req models.AnnouncementRequest) (*models.Announcement, error) {
	var record database.AnnouncementRecord
	if err := r.db.Where("id = ? AND deleted_at IS NULL", announcementID).First(&record).Error; err != nil {
		return nil, mapGormError(err)
	}
	if !canManageAnnouncement(user, record) {
		return nil, ErrForbidden
	}
	record.Title = strings.TrimSpace(req.Title)
	record.Content = strings.TrimSpace(req.Content)
	record.Pinned = req.Pinned
	if err := r.db.Save(&record).Error; err != nil {
		return nil, err
	}
	item, err := r.announcementRecordToModel(record)
	return &item, err
}

func (r *PostgresBusinessRepository) DeleteAnnouncement(user *models.User, announcementID string) error {
	var record database.AnnouncementRecord
	if err := r.db.Where("id = ? AND deleted_at IS NULL", announcementID).First(&record).Error; err != nil {
		return mapGormError(err)
	}
	if !canManageAnnouncement(user, record) {
		return ErrForbidden
	}
	now := time.Now()
	return r.db.Model(&record).Update("deleted_at", &now).Error
}

func (r *PostgresBusinessRepository) ListTeams() ([]models.TeamAdminItem, error) {
	var teams []database.TeamRecord
	if err := r.db.Order("created_at ASC").Find(&teams).Error; err != nil {
		return nil, err
	}
	result := make([]models.TeamAdminItem, 0, len(teams))
	for _, team := range teams {
		var memberCount int64
		var managerCount int64
		r.db.Model(&database.TeamMemberRecord{}).Where("team_id = ? AND enabled = true", team.ID).Count(&memberCount)
		r.db.Model(&database.TeamMemberRecord{}).Where("team_id = ? AND team_role = 'manager' AND enabled = true", team.ID).Count(&managerCount)
		result = append(result, models.TeamAdminItem{
			TeamID: team.ID, TeamName: team.Name, Description: team.Description,
			MemberCount: int(memberCount), ManagerCount: int(managerCount), Enabled: team.Enabled,
			CreatedAt: formatTime(team.CreatedAt), UpdatedAt: formatTime(team.UpdatedAt),
		})
	}
	return result, nil
}

func (r *PostgresBusinessRepository) CreateTeam(req models.TeamRequest) (*models.TeamAdminItem, error) {
	record := database.TeamRecord{
		ID:          fmt.Sprintf("team-%d", time.Now().UnixNano()),
		Name:        strings.TrimSpace(req.TeamName),
		Description: strings.TrimSpace(req.Description),
		Enabled:     true,
	}
	if err := r.db.Create(&record).Error; err != nil {
		return nil, err
	}
	items, err := r.ListTeams()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.TeamID == record.ID {
			return &item, nil
		}
	}
	return nil, ErrNotFound
}

func (r *PostgresBusinessRepository) UpdateTeam(teamID string, req models.TeamRequest) (*models.TeamAdminItem, error) {
	updates := map[string]any{
		"name":        strings.TrimSpace(req.TeamName),
		"description": strings.TrimSpace(req.Description),
	}
	result := r.db.Model(&database.TeamRecord{}).Where("id = ?", teamID).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	items, err := r.ListTeams()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.TeamID == teamID {
			return &item, nil
		}
	}
	return nil, ErrNotFound
}

func (r *PostgresBusinessRepository) DisableTeam(teamID string) error {
	result := r.db.Model(&database.TeamRecord{}).Where("id = ?", teamID).Update("enabled", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresBusinessRepository) ListTeamMembers(teamID string) ([]models.TeamMemberItem, error) {
	var rows []teamMemberRow
	if err := r.db.Table("team_members").
		Select("team_members.team_id, teams.name AS team_name, team_members.user_id, users.account, users.display_name, team_members.team_role, team_members.joined_at, team_members.enabled").
		Joins("JOIN users ON users.id = team_members.user_id").
		Joins("JOIN teams ON teams.id = team_members.team_id").
		Where("team_members.team_id = ?", teamID).
		Order("team_members.joined_at ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return teamMemberRowsToModels(rows), nil
}

func (r *PostgresBusinessRepository) AddTeamMember(teamID string, req models.TeamMemberMutationRequest) (*models.TeamMemberMutationResponse, error) {
	if req.TeamRole != "manager" && req.TeamRole != "employee" {
		return nil, ErrInvalidRole
	}
	record := database.TeamMemberRecord{TeamID: teamID, UserID: req.UserID, TeamRole: req.TeamRole, Enabled: true, JoinedAt: time.Now()}
	if err := r.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&record).Error; err != nil {
		return nil, err
	}
	return &models.TeamMemberMutationResponse{Success: true, TeamID: teamID, UserID: req.UserID, TeamRole: req.TeamRole}, nil
}

func (r *PostgresBusinessRepository) UpdateTeamMember(teamID string, userID string, req models.TeamMemberMutationRequest) (*models.TeamMemberMutationResponse, error) {
	if req.TeamRole != "manager" && req.TeamRole != "employee" {
		return nil, ErrInvalidRole
	}
	result := r.db.Model(&database.TeamMemberRecord{}).Where("team_id = ? AND user_id = ?", teamID, userID).Updates(map[string]any{"team_role": req.TeamRole, "enabled": true})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return &models.TeamMemberMutationResponse{Success: true, TeamID: teamID, UserID: userID, TeamRole: req.TeamRole}, nil
}

func (r *PostgresBusinessRepository) RemoveTeamMember(teamID string, userID string) error {
	result := r.db.Model(&database.TeamMemberRecord{}).Where("team_id = ? AND user_id = ?", teamID, userID).Update("enabled", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresBusinessRepository) ListAdminUsers(teamID string, keyword string) ([]models.AdminUserItem, error) {
	var users []database.UserRecord
	query := r.db.Model(&database.UserRecord{}).Order("created_at ASC")
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("account LIKE ? OR display_name LIKE ?", like, like)
	}
	if teamID != "" {
		query = query.Joins("JOIN team_members ON team_members.user_id = users.id AND team_members.team_id = ?", teamID)
	}
	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	return r.userRecordsToAdminItems(users)
}

func (r *PostgresBusinessRepository) FindAdminUser(userID string) (*models.AdminUserItem, error) {
	var user database.UserRecord
	if err := r.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, mapGormError(err)
	}
	items, err := r.userRecordsToAdminItems([]database.UserRecord{user})
	if err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (r *PostgresBusinessRepository) UpdateUserGlobalRole(userID string, globalRole *string) (*models.AdminUserItem, error) {
	role := ""
	if globalRole != nil {
		role = strings.TrimSpace(*globalRole)
		if role != "" && role != "admin" {
			return nil, ErrInvalidRole
		}
	}
	result := r.db.Model(&database.UserRecord{}).Where("id = ?", userID).Update("global_role", role)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return r.FindAdminUser(userID)
}

func (r *PostgresBusinessRepository) applyApprovalVisibility(query *gorm.DB, user *models.User, scope string, teamID string) *gorm.DB {
	if isAdmin(user) {
		if teamID != "" {
			query = query.Where("team_id = ?", teamID)
		}
		if scope == "pending" {
			query = query.Where("status = 'pending'")
		}
		return query
	}
	if scope == "pending" {
		return query.Where("team_id = ? AND status = 'pending'", user.CurrentTeamID)
	}
	return query.Where("applicant_id = ? OR user_id = ?", user.ID, user.ID)
}

func (r *PostgresBusinessRepository) applyCalendarVisibility(query *gorm.DB, user *models.User, scope string, teamID string) *gorm.DB {
	if isAdmin(user) {
		if teamID != "" {
			return query.Where("team_id = ?", teamID)
		}
		return query
	}
	targetTeamID := valueOrDefault(teamID, user.CurrentTeamID)
	switch scope {
	case "team":
		return query.Where("visibility = 'team' AND team_id = ?", targetTeamID)
	case "mine":
		return query.Where("creator_id = ? OR participant_ids_json::text LIKE ?", user.ID, "%"+user.ID+"%")
	default:
		return query.Where("visibility = 'company' OR (visibility = 'team' AND team_id = ?) OR creator_id = ? OR participant_ids_json::text LIKE ?", user.CurrentTeamID, user.ID, "%"+user.ID+"%")
	}
}

func (r *PostgresBusinessRepository) approvalRecordsToModels(records []database.ApprovalItemRecord) ([]models.ApprovalItem, error) {
	result := make([]models.ApprovalItem, 0, len(records))
	for _, record := range records {
		item, err := r.approvalRecordToModel(record)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *PostgresBusinessRepository) approvalRecordToModel(record database.ApprovalItemRecord) (models.ApprovalItem, error) {
	form := map[string]any{}
	if len(record.FormJSON) > 0 {
		_ = json.Unmarshal(record.FormJSON, &form)
	}
	return models.ApprovalItem{
		ID: record.ID, TeamID: record.TeamID, TeamName: r.teamName(record.TeamID), Type: record.Type,
		Title: record.Title, ApplicantID: record.ApplicantID, Applicant: record.Applicant, ApproverID: record.ApproverID, Approver: record.Approver,
		Amount: record.Amount, Reason: record.Reason, Summary: record.Summary, Form: form,
		Status: record.Status, CreatedAt: formatTime(record.CreatedAt), Comment: record.Comment, UpdatedAt: formatTime(record.UpdatedAt),
	}, nil
}

func (r *PostgresBusinessRepository) calendarRecordsToModels(records []database.CalendarEventRecord) ([]models.CalendarEvent, error) {
	result := make([]models.CalendarEvent, 0, len(records))
	for _, record := range records {
		event, err := r.calendarRecordToModel(record)
		if err != nil {
			return nil, err
		}
		result = append(result, event)
	}
	return result, nil
}

func (r *PostgresBusinessRepository) calendarRecordToModel(record database.CalendarEventRecord) (models.CalendarEvent, error) {
	participants := []string{}
	participantIDs := []string{}
	_ = json.Unmarshal(record.ParticipantsJSON, &participants)
	_ = json.Unmarshal(record.ParticipantIDsJSON, &participantIDs)
	return models.CalendarEvent{
		ID: record.ID, TeamID: record.TeamID, TeamName: r.teamName(record.TeamID), Visibility: record.Visibility,
		CreatorID: record.CreatorID, CreatorName: record.CreatorName,
		Title: record.Title, Description: record.Description, StartTime: formatTime(record.StartTime), EndTime: formatTime(record.EndTime),
		Location: record.Location, Participants: participants, ParticipantIDs: participantIDs, UpdatedAt: formatTime(record.UpdatedAt),
	}, nil
}

func (r *PostgresBusinessRepository) announcementRecordsToModels(records []database.AnnouncementRecord) ([]models.Announcement, error) {
	result := make([]models.Announcement, 0, len(records))
	for _, record := range records {
		item, err := r.announcementRecordToModel(record)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *PostgresBusinessRepository) announcementRecordToModel(record database.AnnouncementRecord) (models.Announcement, error) {
	return models.Announcement{
		ID: record.ID, TeamID: record.TeamID, TeamName: r.teamName(record.TeamID), Scope: record.Scope, Title: record.Title, Content: record.Content,
		PublisherID: record.PublisherID, PublisherName: record.PublisherName, Pinned: record.Pinned,
		CreatedAt: formatTime(record.CreatedAt), UpdatedAt: formatTime(record.UpdatedAt),
	}, nil
}

func calendarRequestToRecord(user *models.User, id string, teamID string, visibility string, req models.CreateCalendarEventRequest) database.CalendarEventRecord {
	participantsJSON, _ := json.Marshal(req.ParticipantIDs)
	participantIDsJSON, _ := json.Marshal(req.ParticipantIDs)
	return database.CalendarEventRecord{
		ID: idOrGenerated(id, "evt"), UserID: user.ID, TeamID: teamID, Visibility: visibility,
		CreatorID: user.ID, CreatorName: user.DisplayName,
		Title: req.Title, Description: req.Description, StartTime: parseRFC3339OrNow(req.StartTime), EndTime: parseRFC3339OrNow(req.EndTime),
		Location: req.Location, ParticipantsJSON: datatypes.JSON(participantsJSON), ParticipantIDsJSON: datatypes.JSON(participantIDsJSON),
	}
}

func (r *PostgresBusinessRepository) firstTeamManager(teamID string) (string, string, error) {
	var row struct {
		UserID      string
		DisplayName string
	}
	if err := r.db.Table("team_members").Select("team_members.user_id, users.display_name").
		Joins("JOIN users ON users.id = team_members.user_id").
		Where("team_members.team_id = ? AND team_members.team_role = 'manager' AND team_members.enabled = true", teamID).
		Limit(1).Scan(&row).Error; err != nil {
		return "", "", err
	}
	if row.UserID == "" {
		return "", "", ErrNotFound
	}
	return row.UserID, row.DisplayName, nil
}

func (r *PostgresBusinessRepository) teamName(teamID string) string {
	if teamID == "" {
		return ""
	}
	var team database.TeamRecord
	if err := r.db.Where("id = ?", teamID).First(&team).Error; err != nil {
		return ""
	}
	return team.Name
}

type teamMemberRow struct {
	TeamID      string
	TeamName    string
	UserID      string
	Account     string
	DisplayName string
	TeamRole    string
	JoinedAt    time.Time
	Enabled     bool
}

func teamMemberRowsToModels(rows []teamMemberRow) []models.TeamMemberItem {
	result := make([]models.TeamMemberItem, 0, len(rows))
	for _, row := range rows {
		result = append(result, models.TeamMemberItem{
			UserID: row.UserID, Account: row.Account, DisplayName: row.DisplayName,
			TeamID: row.TeamID, TeamName: row.TeamName, TeamRole: row.TeamRole,
			JoinedAt: formatTime(row.JoinedAt), Enabled: row.Enabled,
		})
	}
	return result
}

func (r *PostgresBusinessRepository) userRecordsToAdminItems(users []database.UserRecord) ([]models.AdminUserItem, error) {
	result := make([]models.AdminUserItem, 0, len(users))
	for _, user := range users {
		memberships, err := r.membershipsForUser(user.ID)
		if err != nil {
			return nil, err
		}
		var globalRole *string
		if user.GlobalRole != "" {
			globalRole = &user.GlobalRole
		}
		result = append(result, models.AdminUserItem{UserID: user.ID, Account: user.Account, DisplayName: user.DisplayName, GlobalRole: globalRole, Memberships: memberships, Enabled: user.Enabled})
	}
	return result, nil
}

func (r *PostgresBusinessRepository) membershipsForUser(userID string) ([]models.TeamMembership, error) {
	var rows []teamMemberRow
	if err := r.db.Table("team_members").
		Select("team_members.team_id, teams.name AS team_name, team_members.team_role").
		Joins("JOIN teams ON teams.id = team_members.team_id").
		Where("team_members.user_id = ? AND team_members.enabled = true", userID).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]models.TeamMembership, 0, len(rows))
	for _, row := range rows {
		result = append(result, models.TeamMembership{TeamID: row.TeamID, TeamName: row.TeamName, TeamRole: row.TeamRole})
	}
	return result, nil
}

func canViewApproval(user *models.User, record database.ApprovalItemRecord) bool {
	return isAdmin(user) || record.ApplicantID == user.ID || record.UserID == user.ID || userHasTeamRole(user, record.TeamID, "manager")
}

func canViewCalendar(user *models.User, record database.CalendarEventRecord) bool {
	return isAdmin(user) || record.Visibility == "company" || record.TeamID == user.CurrentTeamID || record.CreatorID == user.ID || jsonTextContains(record.ParticipantIDsJSON, user.ID)
}

func canManageCalendar(user *models.User, record database.CalendarEventRecord) bool {
	return isAdmin(user) || userHasTeamRole(user, record.TeamID, "manager")
}

func canViewAnnouncement(user *models.User, record database.AnnouncementRecord) bool {
	return isAdmin(user) || record.Scope == "company" || record.TeamID == user.CurrentTeamID
}

func canWriteAnnouncement(user *models.User, scope string, teamID string) bool {
	if isAdmin(user) {
		return true
	}
	if scope == "company" {
		return false
	}
	return userHasTeamRole(user, valueOrDefault(teamID, user.CurrentTeamID), "manager")
}

func canManageAnnouncement(user *models.User, record database.AnnouncementRecord) bool {
	return isAdmin(user) || userHasTeamRole(user, record.TeamID, "manager")
}

func isAdmin(user *models.User) bool {
	return user.GlobalRole == "admin"
}

func userInTeam(user *models.User, teamID string) bool {
	for _, membership := range user.Memberships {
		if membership.TeamID == teamID {
			return true
		}
	}
	return false
}

func userHasTeamRole(user *models.User, teamID string, role string) bool {
	for _, membership := range user.Memberships {
		if membership.TeamID == teamID && membership.TeamRole == role {
			return true
		}
	}
	return false
}

func jsonTextContains(data datatypes.JSON, value string) bool {
	return strings.Contains(string(data), value)
}

func approvalScopeFromStatus(status string) string {
	if status == "pending" {
		return "pending"
	}
	return "mine"
}

func countApprovedToday(items []models.ApprovalItem) int {
	today := time.Now().Format("2006-01-02")
	count := 0
	for _, item := range items {
		if item.Status == "approved" && strings.HasPrefix(item.UpdatedAt, today) {
			count++
		}
	}
	return count
}

func idOrGenerated(id string, prefix string) string {
	if id != "" {
		return id
	}
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func valueOrDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func parseRFC3339OrNow(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Now()
	}
	return parsed
}
