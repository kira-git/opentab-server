package repositories

import (
	"encoding/json"
	"fmt"
	"time"

	"opentab-server/internal/database"
	"opentab-server/internal/mockdata"
	"opentab-server/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostgresUserRepository struct {
	db *gorm.DB
}

func NewPostgresUserRepository(db *gorm.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) FindByAccount(account string) (*models.User, error) {
	var user database.UserRecord
	if err := r.db.Where("account = ?", account).First(&user).Error; err != nil {
		return nil, mapGormError(err)
	}
	if !user.Enabled {
		return nil, ErrUserDisabled
	}
	return r.userWithPermissions(user)
}

func (r *PostgresUserRepository) FindByToken(token string) (*models.User, error) {
	var session database.AuthSessionRecord
	if err := r.db.Where("token = ?", token).First(&session).Error; err != nil {
		return nil, mapGormError(err)
	}
	if session.RevokedAt != nil {
		return nil, ErrTokenRevoked
	}
	if session.ExpiresAt != nil && session.ExpiresAt.Before(timeNow()) {
		return nil, ErrTokenExpired
	}

	var user database.UserRecord
	if err := r.db.Where("id = ?", session.UserID).First(&user).Error; err != nil {
		return nil, mapGormError(err)
	}
	if !user.Enabled {
		return nil, ErrUserDisabled
	}
	result, err := r.userWithPermissions(user)
	if err != nil {
		return nil, err
	}
	result.Token = token
	return result, nil
}

func (r *PostgresUserRepository) Create(user models.User, enabledTabIDs []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		userRecord := database.UserRecord{
			ID:           user.ID,
			Account:      user.Account,
			DisplayName:  user.DisplayName,
			PasswordHash: user.Password,
			GlobalRole:   user.GlobalRole,
			Enabled:      true,
		}
		if err := tx.Create(&userRecord).Error; err != nil {
			return mapCreateError(err)
		}

		for _, permission := range user.Permissions {
			record := database.UserPermissionRecord{
				UserID:         user.ID,
				PermissionCode: permission,
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&record).Error; err != nil {
				return err
			}
		}

		for index, tabID := range enabledTabIDs {
			record := database.UserTabRecord{
				UserID:    user.ID,
				TabID:     tabID,
				Enabled:   true,
				SortOrder: (index + 1) * 10,
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&record).Error; err != nil {
				return err
			}
		}
		member := database.TeamMemberRecord{
			TeamID:   "team-product",
			UserID:   user.ID,
			TeamRole: "employee",
			Enabled:  true,
			JoinedAt: time.Now(),
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&member).Error; err != nil {
			return err
		}
		if err := createDefaultBusinessData(tx, user.ID); err != nil {
			return err
		}
		return nil
	})
}

func (r *PostgresUserRepository) UpdatePasswordHash(userID string, passwordHash string) error {
	result := r.db.Model(&database.UserRecord{}).Where("id = ?", userID).Update("password_hash", passwordHash)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresUserRepository) CreateSession(userID string, token string, expiresAt time.Time) error {
	session := database.AuthSessionRecord{
		ID:        fmt.Sprintf("session-%d", time.Now().UnixNano()),
		UserID:    userID,
		Token:     token,
		ExpiresAt: &expiresAt,
	}
	if err := r.db.Create(&session).Error; err != nil {
		return mapCreateError(err)
	}
	return nil
}

func (r *PostgresUserRepository) RevokeToken(token string) error {
	now := time.Now()
	result := r.db.Model(&database.AuthSessionRecord{}).
		Where("token = ? AND revoked_at IS NULL", token).
		Update("revoked_at", &now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresUserRepository) userWithPermissions(user database.UserRecord) (*models.User, error) {
	var permissionRecords []database.UserPermissionRecord
	if err := r.db.Where("user_id = ?", user.ID).Find(&permissionRecords).Error; err != nil {
		return nil, err
	}
	permissions := make([]string, 0, len(permissionRecords))
	for _, permission := range permissionRecords {
		permissions = append(permissions, permission.PermissionCode)
	}

	var session database.AuthSessionRecord
	_ = r.db.Where("user_id = ? AND revoked_at IS NULL", user.ID).First(&session).Error
	memberships, currentTeamID, err := r.userMemberships(user.ID)
	if err != nil {
		return nil, err
	}

	return &models.User{
		ID:            user.ID,
		Account:       user.Account,
		DisplayName:   user.DisplayName,
		Password:      user.PasswordHash,
		Token:         session.Token,
		GlobalRole:    user.GlobalRole,
		CurrentTeamID: currentTeamID,
		Memberships:   memberships,
		Permissions:   permissions,
		Enabled:       user.Enabled,
	}, nil
}

func (r *PostgresUserRepository) userMemberships(userID string) ([]models.TeamMembership, string, error) {
	type row struct {
		TeamID   string
		TeamName string
		TeamRole string
	}
	var rows []row
	if err := r.db.Table("team_members").
		Select("team_members.team_id, teams.name AS team_name, team_members.team_role").
		Joins("JOIN teams ON teams.id = team_members.team_id").
		Where("team_members.user_id = ? AND team_members.enabled = true AND teams.enabled = true", userID).
		Order("team_members.joined_at ASC").
		Scan(&rows).Error; err != nil {
		return nil, "", err
	}
	memberships := make([]models.TeamMembership, 0, len(rows))
	currentTeamID := ""
	for i, item := range rows {
		memberships = append(memberships, models.TeamMembership{
			TeamID:   item.TeamID,
			TeamName: item.TeamName,
			TeamRole: item.TeamRole,
		})
		if i == 0 {
			currentTeamID = item.TeamID
		}
	}
	return memberships, currentTeamID, nil
}

func timeNow() time.Time {
	return time.Now()
}

func createDefaultBusinessData(tx *gorm.DB, userID string) error {
	for _, item := range mockdata.ApprovalSummary.Items {
		record := database.ApprovalItemRecord{
			ID:        userID + "-" + item.ID,
			UserID:    userID,
			Title:     item.Title,
			Applicant: item.Applicant,
			Amount:    item.Amount,
			Reason:    item.Reason,
			Status:    item.Status,
			Comment:   item.Comment,
			CreatedAt: parseRFC3339OrNow(item.CreatedAt),
			UpdatedAt: parseRFC3339OrNow(item.UpdatedAt),
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&record).Error; err != nil {
			return err
		}
	}

	for _, event := range mockdata.CalendarSummary.Events {
		participantsJSON, err := json.Marshal(event.Participants)
		if err != nil {
			return err
		}
		record := database.CalendarEventRecord{
			ID:               userID + "-" + event.ID,
			UserID:           userID,
			Title:            event.Title,
			Description:      event.Description,
			StartTime:        parseRFC3339OrNow(event.StartTime),
			EndTime:          parseRFC3339OrNow(event.EndTime),
			Location:         event.Location,
			ParticipantsJSON: datatypes.JSON(participantsJSON),
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&record).Error; err != nil {
			return err
		}
	}
	return nil
}
