package repositories

import (
	"errors"

	"opentab-server/internal/database"
	"opentab-server/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostgresTabRepository struct {
	db *gorm.DB
}

func NewPostgresTabRepository(db *gorm.DB) *PostgresTabRepository {
	return &PostgresTabRepository{db: db}
}

func (r *PostgresTabRepository) ListAll() ([]models.TabManifest, error) {
	var records []database.TabRecord
	if err := r.db.Order("id ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	result := make([]models.TabManifest, 0, len(records))
	for _, record := range records {
		tab, err := tabRecordToManifest(record, true, 0)
		if err != nil {
			return nil, err
		}
		result = append(result, tab)
	}
	return result, nil
}

func (r *PostgresTabRepository) ListByUser(userID string) ([]models.TabManifest, error) {
	var rows []struct {
		database.TabRecord
		Enabled   bool
		SortOrder int
	}
	err := r.db.Table("tabs").
		Select("tabs.*, user_tabs.enabled, user_tabs.sort_order").
		Joins("JOIN user_tabs ON user_tabs.tab_id = tabs.id").
		Where("user_tabs.user_id = ? AND user_tabs.enabled = ?", userID, true).
		Order("user_tabs.sort_order ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]models.TabManifest, 0, len(rows))
	for _, row := range rows {
		tab, err := tabRecordToManifest(row.TabRecord, row.Enabled, row.SortOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, tab)
	}
	return result, nil
}

func (r *PostgresTabRepository) ListCatalog(userID string) ([]models.TabManifest, error) {
	var records []database.TabRecord
	if err := r.db.
		Where("is_system = ? OR owner_user_id = ?", true, userID).
		Order("id ASC").
		Find(&records).Error; err != nil {
		return nil, err
	}

	var userTabs []database.UserTabRecord
	if err := r.db.Where("user_id = ?", userID).Find(&userTabs).Error; err != nil {
		return nil, err
	}
	enabledByTabID := map[string]database.UserTabRecord{}
	for _, userTab := range userTabs {
		enabledByTabID[userTab.TabID] = userTab
	}

	result := make([]models.TabManifest, 0, len(records))
	for _, record := range records {
		userTab := enabledByTabID[record.ID]
		tab, err := tabRecordToManifest(record, userTab.Enabled, userTab.SortOrder)
		if err != nil {
			return nil, err
		}
		result = append(result, tab)
	}
	return result, nil
}

func (r *PostgresTabRepository) FindByID(tabID string) (*models.TabManifest, error) {
	var record database.TabRecord
	if err := r.db.Where("id = ?", tabID).First(&record).Error; err != nil {
		return nil, mapGormError(err)
	}
	tab, err := tabRecordToManifest(record, true, 0)
	if err != nil {
		return nil, err
	}
	return &tab, nil
}

func (r *PostgresTabRepository) CreateCustom(userID string, tab models.TabManifest) (*models.TabManifest, error) {
	record, err := manifestToTabRecord(tab, userID, false)
	if err != nil {
		return nil, err
	}

	err = r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		userTab := database.UserTabRecord{
			UserID:    userID,
			TabID:     tab.ID,
			Enabled:   true,
			SortOrder: tab.SortOrder,
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "tab_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"enabled", "sort_order", "updated_at"}),
		}).Create(&userTab).Error
	})
	if err != nil {
		return nil, err
	}

	created, err := tabRecordToManifest(record, true, tab.SortOrder)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (r *PostgresTabRepository) UpdateCustom(userID string, tabID string, req models.UpdateCustomTabRequest) (*models.TabManifest, error) {
	var record database.TabRecord
	if err := r.db.Where("id = ?", tabID).First(&record).Error; err != nil {
		return nil, mapGormError(err)
	}
	if record.IsSystem || record.OwnerUserID == nil || *record.OwnerUserID != userID {
		return nil, ErrForbidden
	}

	if req.DisplayName != "" {
		record.DisplayName = req.DisplayName
	}
	record.Description = req.Description
	if req.Icon != "" {
		record.Icon = req.Icon
	}
	if req.EntryURI != "" {
		record.EntryURI = req.EntryURI
	}
	if err := r.db.Save(&record).Error; err != nil {
		return nil, err
	}
	if req.SortOrder > 0 {
		if err := r.db.Model(&database.UserTabRecord{}).
			Where("user_id = ? AND tab_id = ?", userID, tabID).
			Update("sort_order", req.SortOrder).Error; err != nil {
			return nil, err
		}
	}

	tab, err := tabRecordToManifest(record, true, req.SortOrder)
	if err != nil {
		return nil, err
	}
	return &tab, nil
}

func (r *PostgresTabRepository) DeleteCustom(userID string, tabID string) error {
	var record database.TabRecord
	if err := r.db.Where("id = ?", tabID).First(&record).Error; err != nil {
		return mapGormError(err)
	}
	if record.IsSystem || record.OwnerUserID == nil || *record.OwnerUserID != userID {
		return ErrForbidden
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tab_id = ?", tabID).Delete(&database.UserTabRecord{}).Error; err != nil {
			return err
		}
		return tx.Delete(&record).Error
	})
}

func (r *PostgresTabRepository) RouteExistsForUser(userID string, route string, excludeTabID string) bool {
	var count int64
	query := r.db.Model(&database.TabRecord{}).
		Where("route = ? AND (is_system = ? OR owner_user_id = ?)", route, true, userID)
	if excludeTabID != "" {
		query = query.Where("id <> ?", excludeTabID)
	}
	return query.Count(&count).Error == nil && count > 0
}

func (r *PostgresTabRepository) Reorder(userID string, items []models.ReorderTabItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Model(&database.UserTabRecord{}).
				Where("user_id = ? AND tab_id = ?", userID, item.TabID).
				Update("sort_order", item.SortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PostgresTabRepository) Enable(userID string, tabID string) error {
	userTab := database.UserTabRecord{
		UserID:    userID,
		TabID:     tabID,
		Enabled:   true,
		SortOrder: r.nextSortOrder(userID),
	}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "tab_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled", "updated_at"}),
	}).Create(&userTab).Error
}

func (r *PostgresTabRepository) Disable(userID string, tabID string) error {
	result := r.db.Where("user_id = ? AND tab_id = ?", userID, tabID).Delete(&database.UserTabRecord{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil
	}
	return nil
}

func (r *PostgresTabRepository) Count() int {
	var count int64
	_ = r.db.Model(&database.TabRecord{}).Count(&count).Error
	return int(count)
}

func (r *PostgresTabRepository) nextSortOrder(userID string) int {
	var userTab database.UserTabRecord
	err := r.db.Where("user_id = ?", userID).Order("sort_order DESC").First(&userTab).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 10
	}
	if err != nil {
		return 10
	}
	return userTab.SortOrder + 10
}
