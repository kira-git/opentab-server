package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"opentab-server/internal/mockdata"
	"opentab-server/internal/models"
	"opentab-server/internal/security"

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
		passwordHash, err := security.HashPassword(user.Password)
		if err != nil {
			return fmt.Errorf("hash seed user password %s: %w", user.ID, err)
		}

		var record UserRecord
		err = db.Where("id = ?", user.ID).First(&record).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			record = UserRecord{
				ID:           user.ID,
				Account:      user.Account,
				DisplayName:  user.DisplayName,
				PasswordHash: passwordHash,
				GlobalRole:   user.GlobalRole,
				Enabled:      true,
			}
			if err := db.Create(&record).Error; err != nil {
				return fmt.Errorf("seed user %s: %w", user.ID, err)
			}
		} else if err != nil {
			return fmt.Errorf("seed user %s: %w", user.ID, err)
		} else {
			updates := map[string]any{
				"account":      user.Account,
				"display_name": user.DisplayName,
				"global_role":  user.GlobalRole,
				"enabled":      true,
			}
			if !security.IsBcryptHash(record.PasswordHash) || record.PasswordHash == user.Password {
				updates["password_hash"] = passwordHash
			}
			if err := db.Model(&UserRecord{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
				return fmt.Errorf("seed user %s: %w", user.ID, err)
			}
		}

		session := AuthSessionRecord{
			ID:        "session-" + user.ID,
			UserID:    user.ID,
			Token:     user.Token,
			ExpiresAt: seedSessionExpiresAt(),
		}
		if err := db.Where("token = ?", session.Token).Assign(map[string]any{
			"user_id":    user.ID,
			"expires_at": seedSessionExpiresAt(),
			"revoked_at": nil,
		}).FirstOrCreate(&session).Error; err != nil {
			return fmt.Errorf("seed auth session %s: %w", user.ID, err)
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
		{TeamID: "team-product", UserID: "user-admin", TeamRole: "manager", Enabled: true, JoinedAt: now},
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
			Title: "周五下午请假", ApplicantID: "user-product-employee", Applicant: "陈磊", ApproverID: "user-product-manager", Approver: "刘洋",
			Reason: "周五下午处理个人事务，上午完成接口联调记录交接", Summary: "请假 0.5 天，已补充交接安排", Status: "pending",
			FormJSON:  datatypes.JSON([]byte(`{"leaveType":"事假","days":0.5,"handover":"Tab 接入联调记录已同步到项目群"}`)),
			CreatedAt: parseTimeOrNow("2026-06-03T09:20:00+08:00"), UpdatedAt: parseTimeOrNow("2026-06-03T09:20:00+08:00"),
		},
		{
			ID: "apv-operation-001", UserID: "user-operation-employee", TeamID: "team-operation", Type: "expense",
			Title: "客户走访物料报销", ApplicantID: "user-operation-employee", Applicant: "李静", ApproverID: "user-operation-manager", Approver: "张敏",
			Amount: 320, Reason: "客户走访使用的资料打印和贴纸物料", Summary: "报销 320 元，客户走访物料", Status: "pending",
			FormJSON:  datatypes.JSON([]byte(`{"amount":320,"category":"客户走访","invoice":"已上传电子发票"}`)),
			CreatedAt: parseTimeOrNow("2026-06-03T10:05:00+08:00"), UpdatedAt: parseTimeOrNow("2026-06-03T10:05:00+08:00"),
		},
		{
			ID: "apv-product-002", UserID: "user-product-employee", TeamID: "team-product", Type: "purchase",
			Title: "测试设备采购申请", ApplicantID: "user-product-employee", Applicant: "陈磊", ApproverID: "user-product-manager", Approver: "刘洋",
			Amount: 1299, Reason: "用于 Android 端真机兼容性测试", Summary: "采购一台测试机，预算 1299 元", Status: "approved", Comment: "同意采购，注意登记资产编号",
			FormJSON:  datatypes.JSON([]byte(`{"amount":1299,"category":"测试设备","assetRequired":true}`)),
			CreatedAt: parseTimeOrNow("2026-06-02T15:40:00+08:00"), UpdatedAt: parseTimeOrNow("2026-06-02T16:10:00+08:00"),
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
		calendarSeedRecord("evt-product-001", "team-product", "user-product-manager", "刘洋", "产品研发部晨会", "确认 Tab 注册、权限和 AI OnCall 联调进展", "线上会议", "2026-06-03T09:30:00+08:00", "2026-06-03T10:00:00+08:00", []string{"刘洋", "陈磊"}, []string{"user-product-manager", "user-product-employee"}),
		calendarSeedRecord("evt-operation-001", "team-operation", "user-operation-manager", "张敏", "客户反馈整理", "汇总近期客户对工作台 Tab 的反馈", "会议室 A", "2026-06-03T10:30:00+08:00", "2026-06-03T11:00:00+08:00", []string{"张敏", "李静"}, []string{"user-operation-manager", "user-operation-employee"}),
		calendarSeedRecord("evt-product-002", "team-product", "user-product-manager", "刘洋", "Tab 容器联调复盘", "检查客户端 Tab 列表、审批和日程数据展示", "开发群语音", "2026-06-03T14:00:00+08:00", "2026-06-03T15:00:00+08:00", []string{"刘洋", "陈磊"}, []string{"user-product-manager", "user-product-employee"}),
		calendarSeedRecord("evt-operation-002", "team-operation", "user-operation-manager", "张敏", "公告发布确认", "确认阶段演示公告内容和发布范围", "会议室 B", "2026-06-03T16:00:00+08:00", "2026-06-03T16:40:00+08:00", []string{"张敏", "李静"}, []string{"user-operation-manager", "user-operation-employee"}),
		calendarSeedRecord("evt-company-001", "", "user-admin", "张伟", "阶段演示彩排", "开放式 Tab 容器与 AI OnCall 助理阶段演示", "线上会议", "2026-06-04T15:30:00+08:00", "2026-06-04T16:30:00+08:00", []string{"全员"}, []string{}),
	}
	for _, record := range events {
		if err := db.Where("id = ?", record.ID).Assign(record).FirstOrCreate(&record).Error; err != nil {
			return fmt.Errorf("seed calendar event %s: %w", record.ID, err)
		}
	}
	return nil
}

func calendarSeedRecord(id string, teamID string, creatorID string, creatorName string, title string, description string, location string, startTime string, endTime string, participants []string, participantIDs []string) CalendarEventRecord {
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
		StartTime:          parseTimeOrNow(startTime),
		EndTime:            parseTimeOrNow(endTime),
		Location:           location,
		ParticipantsJSON:   datatypes.JSON(participantsJSON),
		ParticipantIDsJSON: datatypes.JSON(participantIDsJSON),
	}
}

func seedAnnouncements(db *gorm.DB) error {
	records := []AnnouncementRecord{
		{ID: "ann-company-001", Scope: "company", Title: "阶段演示安排", Content: "本周四 15:30 进行开放式 Tab 容器与 AI OnCall 助理阶段演示，请相关成员提前完成数据检查。", PublisherID: "user-admin", PublisherName: "张伟", Pinned: true},
		{ID: "ann-product-001", TeamID: "team-product", Scope: "team", Title: "产品研发部联调提醒", Content: "请在今天 14:00 前确认 Tab 列表、审批中心和日程接口在客户端展示正常。", PublisherID: "user-product-manager", PublisherName: "刘洋"},
		{ID: "ann-operation-001", TeamID: "team-operation", Scope: "team", Title: "客户反馈整理", Content: "请在周三下班前整理客户反馈和常见问题，重点标注和工作台 Tab 相关的需求。", PublisherID: "user-operation-manager", PublisherName: "张敏"},
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
