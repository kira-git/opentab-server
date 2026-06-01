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
		return seedCalendarEvents(tx)
	})
}

func seedUsers(db *gorm.DB) error {
	for _, user := range mockdata.Users {
		record := UserRecord{
			ID:           user.ID,
			Account:      user.Account,
			DisplayName:  user.DisplayName,
			PasswordHash: user.Password,
		}
		if err := db.Where("id = ?", record.ID).FirstOrCreate(&record).Error; err != nil {
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

func seedPermissions(db *gorm.DB) error {
	for _, permission := range mockdata.Permissions {
		record := PermissionRecord{
			Code:        permission["code"],
			Description: permission["description"],
			CreatedAt:   time.Now(),
		}
		if err := db.Where("code = ?", record.Code).FirstOrCreate(&record).Error; err != nil {
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
		if err := db.Where("id = ?", record.ID).FirstOrCreate(&record).Error; err != nil {
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
	for _, user := range mockdata.Users {
		for _, item := range mockdata.ApprovalSummary.Items {
			record := ApprovalItemRecord{
				ID:        businessSeedID(user.ID, item.ID),
				UserID:    user.ID,
				Title:     item.Title,
				Applicant: item.Applicant,
				Amount:    item.Amount,
				Reason:    item.Reason,
				Status:    item.Status,
				Comment:   item.Comment,
				CreatedAt: parseTimeOrNow(item.CreatedAt),
				UpdatedAt: parseTimeOrNow(item.UpdatedAt),
			}
			if err := db.Where("id = ?", record.ID).FirstOrCreate(&record).Error; err != nil {
				return fmt.Errorf("seed approval item %s: %w", record.ID, err)
			}
			if err := db.Model(&ApprovalItemRecord{}).Where("id = ? AND user_id = ''", record.ID).Update("user_id", user.ID).Error; err != nil {
				return fmt.Errorf("backfill approval item user %s: %w", record.ID, err)
			}
		}
	}
	return nil
}

func seedCalendarEvents(db *gorm.DB) error {
	for _, user := range mockdata.Users {
		for _, event := range mockdata.CalendarSummary.Events {
			participantsJSON, err := json.Marshal(event.Participants)
			if err != nil {
				return fmt.Errorf("marshal calendar participants %s: %w", event.ID, err)
			}
			record := CalendarEventRecord{
				ID:               businessSeedID(user.ID, event.ID),
				UserID:           user.ID,
				Title:            event.Title,
				Description:      event.Description,
				StartTime:        parseTimeOrNow(event.StartTime),
				EndTime:          parseTimeOrNow(event.EndTime),
				Location:         event.Location,
				ParticipantsJSON: datatypes.JSON(participantsJSON),
			}
			if err := db.Where("id = ?", record.ID).FirstOrCreate(&record).Error; err != nil {
				return fmt.Errorf("seed calendar event %s: %w", record.ID, err)
			}
			if err := db.Model(&CalendarEventRecord{}).Where("id = ? AND user_id = ''", record.ID).Update("user_id", user.ID).Error; err != nil {
				return fmt.Errorf("backfill calendar event user %s: %w", record.ID, err)
			}
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
