package database

import (
	"encoding/json"
	"fmt"
	"time"

	"opentab-server/internal/mockdata"
	"opentab-server/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := seedUsers(tx); err != nil {
			return err
		}
		if err := seedTeams(tx); err != nil {
			return err
		}
		if err := seedTeamMembers(tx); err != nil {
			return err
		}
		if err := seedPermissions(tx); err != nil {
			return err
		}
		if err := seedUserPermissions(tx); err != nil {
			return err
		}
		if err := seedTabs(tx); err != nil {
			return err
		}
		if err := seedUserTabs(tx); err != nil {
			return err
		}
		if err := seedOnCall(tx); err != nil {
			return err
		}
		if err := seedApprovalItems(tx); err != nil {
			return err
		}
		if err := seedCalendarEvents(tx); err != nil {
			return err
		}
		return seedAnnouncements(tx)
	})
}

func seedUsers(db *gorm.DB) error {
	for _, user := range mockdata.Users {
		record := UserRecord{
			ID:           user.ID,
			Account:      user.Account,
			DisplayName:  user.DisplayName,
			PasswordHash: user.Password,
			GlobalRole:   user.GlobalRole,
			Enabled:      true,
		}
		if err := db.Where("id = ?", record.ID).Assign(record).FirstOrCreate(&record).Error; err != nil {
			return fmt.Errorf("seed user %s: %w", user.ID, err)
		}

		session := AuthSessionRecord{
			ID:        "session-" + user.ID,
			UserID:    user.ID,
			Token:     user.Token,
			ExpiresAt: seedSessionExpiresAt(),
		}
		if err := db.Where("token = ?", session.Token).FirstOrCreate(&session).Error; err != nil {
			return fmt.Errorf("seed auth session %s: %w", user.ID, err)
		}
		if err := db.Model(&AuthSessionRecord{}).Where("token = ? AND expires_at IS NULL", session.Token).Update("expires_at", seedSessionExpiresAt()).Error; err != nil {
			return fmt.Errorf("backfill auth session expires_at %s: %w", user.ID, err)
		}
	}
	return nil
}

func seedTeams(db *gorm.DB) error {
	teams := []TeamRecord{
		{ID: "team-product", Name: "产品研发部", Description: "负责产品、客户端和服务端联调", Enabled: true},
		{ID: "team-operation", Name: "运营支持部", Description: "负责运营支持和客户协同", Enabled: true},
	}
	for _, record := range teams {
		if err := db.Where("id = ?", record.ID).Assign(record).FirstOrCreate(&record).Error; err != nil {
			return fmt.Errorf("seed team %s: %w", record.ID, err)
		}
	}
	return nil
}

func seedTeamMembers(db *gorm.DB) error {
	now := time.Now()
	members := []TeamMemberRecord{
		{TeamID: "team-product", UserID: "user-product-manager", TeamRole: "manager", Enabled: true, JoinedAt: now},
		{TeamID: "team-product", UserID: "user-product-employee", TeamRole: "employee", Enabled: true, JoinedAt: now},
		{TeamID: "team-operation", UserID: "user-operation-manager", TeamRole: "manager", Enabled: true, JoinedAt: now},
		{TeamID: "team-operation", UserID: "user-operation-employee", TeamRole: "employee", Enabled: true, JoinedAt: now},
	}
	for _, record := range members {
		if err := db.Where("team_id = ? AND user_id = ?", record.TeamID, record.UserID).Assign(record).FirstOrCreate(&record).Error; err != nil {
			return fmt.Errorf("seed team member %s/%s: %w", record.TeamID, record.UserID, err)
		}
	}
	return nil
}

func seedPermissions(db *gorm.DB) error {
	for _, permission := range mockdata.Permissions {
		record := PermissionRecord{
			Code:        permission["code"],
			Description: permission["description"],
			CreatedAt:   time.Now(),
		}
		if err := db.Where("code = ?", record.Code).Assign(record).FirstOrCreate(&record).Error; err != nil {
			return fmt.Errorf("seed permission %s: %w", record.Code, err)
		}
	}
	return nil
}

func seedUserPermissions(db *gorm.DB) error {
	for _, user := range mockdata.Users {
		for _, permission := range user.Permissions {
			record := UserPermissionRecord{
				UserID:         user.ID,
				PermissionCode: permission,
			}
			if err := db.Where("user_id = ? AND permission_code = ?", record.UserID, record.PermissionCode).FirstOrCreate(&record).Error; err != nil {
				return fmt.Errorf("seed user permission %s/%s: %w", record.UserID, record.PermissionCode, err)
			}
		}
	}
	return nil
}

func seedTabs(db *gorm.DB) error {
	for _, tab := range mockdata.Tabs {
		record, err := tabToRecord(tab)
		if err != nil {
			return err
		}
		if err := db.Where("id = ?", record.ID).Assign(record).FirstOrCreate(&record).Error; err != nil {
			return fmt.Errorf("seed tab %s: %w", record.ID, err)
		}
	}
	return nil
}

func seedUserTabs(db *gorm.DB) error {
	for userID, tabs := range mockdata.UserTabs {
		for tabID, enabled := range tabs {
			if !enabled {
				continue
			}
			tab := mockdata.FindTab(tabID)
			sortOrder := 0
			if tab != nil {
				sortOrder = tab.SortOrder
			}
			record := UserTabRecord{
				UserID:    userID,
				TabID:     tabID,
				Enabled:   true,
				SortOrder: sortOrder,
			}
			if err := db.Where("user_id = ? AND tab_id = ?", record.UserID, record.TabID).FirstOrCreate(&record).Error; err != nil {
				return fmt.Errorf("seed user tab %s/%s: %w", record.UserID, record.TabID, err)
			}
		}
	}
	return nil
}

func seedOnCall(db *gorm.DB) error {
	for userID, sessions := range mockdata.OnCallSessions {
		for _, session := range sessions {
			createdAt := parseTimeOrNow(session.CreatedAt)
			updatedAt := parseTimeOrNow(session.UpdatedAt)
			record := OnCallSessionRecord{
				ID:        session.SessionID,
				UserID:    userID,
				Title:     session.Title,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			}
			if err := db.Where("id = ?", record.ID).FirstOrCreate(&record).Error; err != nil {
				return fmt.Errorf("seed oncall session %s: %w", record.ID, err)
			}
		}
	}

	for sessionID, messages := range mockdata.OnCallMessages {
		for _, message := range messages {
			record := OnCallMessageRecord{
				ID:          message.MessageID,
				SessionID:   sessionID,
				Role:        message.Role,
				Content:     message.Content,
				ContentType: message.ContentType,
				CreatedAt:   parseTimeOrNow(message.CreatedAt),
			}
			if err := db.Where("id = ?", record.ID).FirstOrCreate(&record).Error; err != nil {
				return fmt.Errorf("seed oncall message %s: %w", record.ID, err)
			}
		}
	}
	return nil
}

func seedApprovalItems(db *gorm.DB) error {
	items := []ApprovalItemRecord{
		{
			ID: "apv-product-001", UserID: "user-product-employee", TeamID: "team-product", Type: "leave",
			Title: "请假申请", ApplicantID: "user-product-employee", Applicant: "产品员工", ApproverID: "user-product-manager", Approver: "产品主管",
			Reason: "家中有事，请假一天", Summary: "请假 1 天", Status: "pending",
			FormJSON:  datatypes.JSON([]byte(`{"leaveType":"事假","days":1}`)),
			CreatedAt: time.Now().Add(-2 * time.Hour), UpdatedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			ID: "apv-operation-001", UserID: "user-operation-employee", TeamID: "team-operation", Type: "expense",
			Title: "活动物料报销", ApplicantID: "user-operation-employee", Applicant: "运营员工", ApproverID: "user-operation-manager", Approver: "运营主管",
			Amount: 320, Reason: "线下活动物料采购", Summary: "报销 320 元", Status: "pending",
			FormJSON:  datatypes.JSON([]byte(`{"amount":320,"category":"活动物料"}`)),
			CreatedAt: time.Now().Add(-1 * time.Hour), UpdatedAt: time.Now().Add(-1 * time.Hour),
		},
	}
	for _, record := range items {
		if err := db.Where("id = ?", record.ID).Assign(record).FirstOrCreate(&record).Error; err != nil {
			return fmt.Errorf("seed approval item %s: %w", record.ID, err)
		}
	}
	return nil
}

func seedCalendarEvents(db *gorm.DB) error {
	events := []CalendarEventRecord{
		calendarSeedRecord("evt-product-001", "team-product", "user-product-manager", "产品主管", "产品研发部周会", "同步开放式 Tab 容器联调进展", "线上会议", []string{"产品主管", "产品员工"}, []string{"user-product-manager", "user-product-employee"}),
		calendarSeedRecord("evt-operation-001", "team-operation", "user-operation-manager", "运营主管", "运营支持部周会", "同步运营支持和客户反馈", "会议室 A", []string{"运营主管", "运营员工"}, []string{"user-operation-manager", "user-operation-employee"}),
		calendarSeedRecord("evt-company-001", "", "user-admin", "系统管理员", "全公司阶段同步", "开放式 Tab 容器阶段演示", "线上会议", []string{"全员"}, []string{}),
	}
	for _, record := range events {
		if err := db.Where("id = ?", record.ID).Assign(record).FirstOrCreate(&record).Error; err != nil {
			return fmt.Errorf("seed calendar event %s: %w", record.ID, err)
		}
	}
	return nil
}

func calendarSeedRecord(id string, teamID string, creatorID string, creatorName string, title string, description string, location string, participants []string, participantIDs []string) CalendarEventRecord {
	participantsJSON, _ := json.Marshal(participants)
	participantIDsJSON, _ := json.Marshal(participantIDs)
	visibility := "team"
	if teamID == "" {
		visibility = "company"
	}
	return CalendarEventRecord{
		ID:                 id,
		UserID:             creatorID,
		TeamID:             teamID,
		Visibility:         visibility,
		CreatorID:          creatorID,
		CreatorName:        creatorName,
		Title:              title,
		Description:        description,
		StartTime:          time.Now().Add(24 * time.Hour),
		EndTime:            time.Now().Add(25 * time.Hour),
		Location:           location,
		ParticipantsJSON:   datatypes.JSON(participantsJSON),
		ParticipantIDsJSON: datatypes.JSON(participantIDsJSON),
	}
}

func seedAnnouncements(db *gorm.DB) error {
	records := []AnnouncementRecord{
		{ID: "ann-company-001", Scope: "company", Title: "全公司阶段同步", Content: "本周进行开放式 Tab 容器和 AI OnCall 阶段演示。", PublisherID: "user-admin", PublisherName: "系统管理员", Pinned: true},
		{ID: "ann-product-001", TeamID: "team-product", Scope: "team", Title: "产品研发部周五分享", Content: "本周五 16:00 分享开放式 Tab 容器联调进展。", PublisherID: "user-product-manager", PublisherName: "产品主管"},
		{ID: "ann-operation-001", TeamID: "team-operation", Scope: "team", Title: "运营支持部客户反馈整理", Content: "请在周五前整理客户反馈和常见问题。", PublisherID: "user-operation-manager", PublisherName: "运营主管"},
	}
	for _, record := range records {
		if err := db.Where("id = ?", record.ID).Assign(record).FirstOrCreate(&record).Error; err != nil {
			return fmt.Errorf("seed announcement %s: %w", record.ID, err)
		}
	}
	return nil
}

func businessSeedID(userID string, itemID string) string {
	if userID == "user-demo" {
		return itemID
	}
	return userID + "-" + itemID
}

func tabToRecord(tab models.TabManifest) (TabRecord, error) {
	permissionsJSON, err := json.Marshal(tab.Permissions)
	if err != nil {
		return TabRecord{}, fmt.Errorf("marshal tab permissions %s: %w", tab.ID, err)
	}

	extensionJSON, err := marshalNullableJSON(tab.Extension)
	if err != nil {
		return TabRecord{}, fmt.Errorf("marshal tab extension %s: %w", tab.ID, err)
	}

	return TabRecord{
		ID:                  tab.ID,
		DisplayName:         tab.DisplayName,
		Description:         valueOrEmpty(tab.Description),
		Icon:                tab.Icon,
		Route:               tab.Route,
		EntryType:           tab.EntryType,
		EntryURI:            tab.EntryURI,
		VersionMajor:        tab.Version.Major,
		VersionMinor:        tab.Version.Minor,
		VersionPatch:        tab.Version.Patch,
		MinContainerVersion: tab.MinContainerVersion,
		PermissionsJSON:     datatypes.JSON(permissionsJSON),
		ExtensionJSON:       extensionJSON,
		ExtraConfigJSON:     datatypes.JSON(tab.ExtraConfig),
		IsSystem:            true,
	}, nil
}

func marshalNullableJSON(value any) (datatypes.JSON, error) {
	if value == nil {
		return nil, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(data), nil
}

func valueOrEmpty(value string) string {
	return value
}

func parseTimeOrNow(value string) time.Time {
	if value == "" {
		return time.Now()
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Now()
	}
	return parsed
}

func seedSessionExpiresAt() *time.Time {
	value := time.Now().Add(7 * 24 * time.Hour)
	return &value
}
