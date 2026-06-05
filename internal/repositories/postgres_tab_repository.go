package repositories

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"sort"

	"opentab-server/internal/database"
	"opentab-server/internal/models"
	"opentab-server/internal/policies"

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
	targets, err := r.visibilityTargetsByTabID(records)
	if err != nil {
		return nil, err
	}
	result := make([]models.TabManifest, 0, len(records))
	for _, record := range records {
		tab, err := tabRecordToManifest(record, true, 0)
		if err != nil {
			return nil, err
		}
		attachVisibilityTargets(&tab, targets[record.ID])
		result = append(result, tab)
	}
	return result, nil
}

func (r *PostgresTabRepository) ListByUser(user *models.User) ([]models.TabManifest, error) {
	records, err := r.visibleRecords(user)
	if err != nil {
		return nil, err
	}
	userTabs, err := r.userTabsByTabID(user.ID)
	if err != nil {
		return nil, err
	}
	targets, err := r.visibilityTargetsByTabID(records)
	if err != nil {
		return nil, err
	}
	result := make([]models.TabManifest, 0, len(records))
	for _, record := range records {
		userTab, hasUserTab := userTabs[record.ID]
		if hasUserTab && !userTab.Enabled {
			continue
		}
		if !hasUserTab && !record.DefaultEnabled {
			continue
		}
		sortOrder := userTab.SortOrder
		if sortOrder == 0 {
			sortOrder = 1000
		}
		tab, err := tabRecordToManifest(record, true, sortOrder)
		if err != nil {
			return nil, err
		}
		attachVisibilityTargets(&tab, targets[record.ID])
		result = append(result, tab)
	}
	sortTabManifests(result)
	return result, nil
}

func (r *PostgresTabRepository) ListCatalog(user *models.User) ([]models.TabManifest, error) {
	records, err := r.visibleRecords(user)
	if err != nil {
		return nil, err
	}
	userTabs, err := r.userTabsByTabID(user.ID)
	if err != nil {
		return nil, err
	}
	targets, err := r.visibilityTargetsByTabID(records)
	if err != nil {
		return nil, err
	}
	result := make([]models.TabManifest, 0, len(records))
	for _, record := range records {
		userTab, hasUserTab := userTabs[record.ID]
		sortOrder := userTab.SortOrder
		if sortOrder == 0 {
			sortOrder = 1000
		}
		enabled := userTab.Enabled || (!hasUserTab && record.DefaultEnabled)
		tab, err := tabRecordToManifest(record, enabled, sortOrder)
		if err != nil {
			return nil, err
		}
		attachVisibilityTargets(&tab, targets[record.ID])
		result = append(result, tab)
	}
	sortTabManifests(result)
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
	targets, err := r.visibilityTargetsByTabID([]database.TabRecord{record})
	if err != nil {
		return nil, err
	}
	attachVisibilityTargets(&tab, targets[record.ID])
	return &tab, nil
}

func (r *PostgresTabRepository) FindVisibleByID(user *models.User, tabID string) (*models.TabManifest, error) {
	records, err := r.visibleRecords(user)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if record.ID != tabID {
			continue
		}
		tab, err := tabRecordToManifest(record, true, 0)
		if err != nil {
			return nil, err
		}
		targets, err := r.visibilityTargetsByTabID([]database.TabRecord{record})
		if err != nil {
			return nil, err
		}
		attachVisibilityTargets(&tab, targets[record.ID])
		return &tab, nil
	}
	return nil, ErrForbidden
}

func (r *PostgresTabRepository) CreateCustom(user *models.User, tab models.TabManifest, visibility models.TabVisibility) (*models.TabManifest, error) {
	record, err := manifestToTabRecord(tab, user.ID, false)
	if err != nil {
		return nil, err
	}
	record.VisibilityScope = visibility.Scope
	record.DefaultEnabled = visibility.DefaultEnabled
	record.ManagedByAdmin = visibility.Scope != "self"

	err = r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return mapCreateError(err)
		}
		if err := r.replaceVisibilityTargets(tx, tab.ID, visibility); err != nil {
			return err
		}
		if !visibility.DefaultEnabled {
			return nil
		}
		targetUserIDs, err := r.resolveTargetUserIDs(tx, user.ID, visibility)
		if err != nil {
			return err
		}
		return r.upsertUserTabs(tx, targetUserIDs, tab.ID, tabUserTabSource(visibility), tab.SortOrder)
	})
	if err != nil {
		return nil, err
	}
	created, err := tabRecordToManifest(record, true, tab.SortOrder)
	if err != nil {
		return nil, err
	}
	created.Visibility = &visibility
	return &created, nil
}

func (r *PostgresTabRepository) UpdateCustom(user *models.User, tabID string, req models.UpdateCustomTabRequest, visibility *models.TabVisibility) (*models.TabManifest, error) {
	var record database.TabRecord
	if err := r.db.Where("id = ?", tabID).First(&record).Error; err != nil {
		return nil, mapGormError(err)
	}
	if record.IsSystem || record.OwnerUserID == nil || *record.OwnerUserID != user.ID {
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
	if visibility != nil {
		record.VisibilityScope = visibility.Scope
		record.DefaultEnabled = visibility.DefaultEnabled
		record.ManagedByAdmin = visibility.Scope != "self"
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&record).Error; err != nil {
			return err
		}
		if req.SortOrder > 0 {
			if err := tx.Model(&database.UserTabRecord{}).Where("tab_id = ?", tabID).Update("sort_order", req.SortOrder).Error; err != nil {
				return err
			}
		}
		if visibility == nil {
			return nil
		}
		if err := r.replaceVisibilityTargets(tx, tabID, *visibility); err != nil {
			return err
		}
		if !visibility.DefaultEnabled {
			return nil
		}
		targetUserIDs, err := r.resolveTargetUserIDs(tx, user.ID, *visibility)
		if err != nil {
			return err
		}
		return r.upsertUserTabs(tx, targetUserIDs, tabID, tabUserTabSource(*visibility), req.SortOrder)
	})
	if err != nil {
		return nil, err
	}
	tab, err := tabRecordToManifest(record, true, req.SortOrder)
	if err != nil {
		return nil, err
	}
	if visibility != nil {
		tab.Visibility = visibility
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
		if err := tx.Where("tab_id = ?", tabID).Delete(&database.TabVisibilityTargetRecord{}).Error; err != nil {
			return err
		}
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
		Source:    "user_add",
		SortOrder: r.nextSortOrder(userID),
	}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "tab_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled", "source", "updated_at"}),
	}).Create(&userTab).Error
}

func (r *PostgresTabRepository) Disable(userID string, tabID string) error {
	userTab := database.UserTabRecord{
		UserID:  userID,
		TabID:   tabID,
		Enabled: false,
		Source:  "user_disabled",
	}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "tab_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled", "source", "updated_at"}),
	}).Create(&userTab).Error
}

func (r *PostgresTabRepository) Count() int {
	var count int64
	_ = r.db.Model(&database.TabRecord{}).Count(&count).Error
	return int(count)
}

func (r *PostgresTabRepository) nextSortOrder(userID string) int {
	var userTab database.UserTabRecord
	err := r.db.Where("user_id = ? AND enabled = true", userID).Order("sort_order DESC").First(&userTab).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || err != nil {
		return 10
	}
	return userTab.SortOrder + 10
}

func (r *PostgresTabRepository) visibleRecords(user *models.User) ([]database.TabRecord, error) {
	if user == nil {
		return nil, ErrForbidden
	}
	query := r.visibleTabQuery(user).Order("tabs.id ASC")

	var records []database.TabRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (r *PostgresTabRepository) visibleTabQuery(user *models.User) *gorm.DB {
	teamIDs := policies.UserTeamIDs(user)
	query := r.db.Model(&database.TabRecord{}).Distinct("tabs.*").
		Joins("LEFT JOIN tab_visibility_targets tvt_user ON tvt_user.tab_id = tabs.id AND tvt_user.target_type = 'user' AND tvt_user.target_id = ?", user.ID)
	if len(teamIDs) > 0 {
		query = query.Joins("LEFT JOIN tab_visibility_targets tvt_team ON tvt_team.tab_id = tabs.id AND tvt_team.target_type = 'team' AND tvt_team.target_id IN ?", teamIDs)
	} else {
		query = query.Joins("LEFT JOIN tab_visibility_targets tvt_team ON tvt_team.tab_id = tabs.id AND false")
	}
	query = query.Where(
		"tabs.is_system = true OR tabs.owner_user_id = ? OR tabs.visibility_scope = 'company' OR tvt_user.id IS NOT NULL OR tvt_team.id IS NOT NULL",
		user.ID,
	)
	return query
}

func (r *PostgresTabRepository) userTabsByTabID(userID string) (map[string]database.UserTabRecord, error) {
	var records []database.UserTabRecord
	if err := r.db.Where("user_id = ?", userID).Find(&records).Error; err != nil {
		return nil, err
	}
	result := map[string]database.UserTabRecord{}
	for _, record := range records {
		result[record.TabID] = record
	}
	return result, nil
}

func (r *PostgresTabRepository) visibilityTargetsByTabID(records []database.TabRecord) (map[string][]database.TabVisibilityTargetRecord, error) {
	result := map[string][]database.TabVisibilityTargetRecord{}
	tabIDs := make([]string, 0, len(records))
	for _, record := range records {
		tabIDs = append(tabIDs, record.ID)
	}
	if len(tabIDs) == 0 {
		return result, nil
	}
	var targets []database.TabVisibilityTargetRecord
	if err := r.db.Where("tab_id IN ?", tabIDs).Find(&targets).Error; err != nil {
		return nil, err
	}
	for _, target := range targets {
		result[target.TabID] = append(result[target.TabID], target)
	}
	return result, nil
}

func attachVisibilityTargets(tab *models.TabManifest, targets []database.TabVisibilityTargetRecord) {
	if tab.Visibility == nil {
		return
	}
	for _, target := range targets {
		switch target.TargetType {
		case "team":
			tab.Visibility.TeamIDs = append(tab.Visibility.TeamIDs, target.TargetID)
		case "user":
			tab.Visibility.UserIDs = append(tab.Visibility.UserIDs, target.TargetID)
		}
	}
}

func (r *PostgresTabRepository) replaceVisibilityTargets(tx *gorm.DB, tabID string, visibility models.TabVisibility) error {
	if err := tx.Where("tab_id = ?", tabID).Delete(&database.TabVisibilityTargetRecord{}).Error; err != nil {
		return err
	}
	records := make([]database.TabVisibilityTargetRecord, 0, len(visibility.TeamIDs)+len(visibility.UserIDs))
	for _, teamID := range uniqueStrings(visibility.TeamIDs) {
		records = append(records, database.TabVisibilityTargetRecord{ID: generatedVisibilityTargetID(tabID, "team", teamID), TabID: tabID, TargetType: "team", TargetID: teamID})
	}
	for _, userID := range uniqueStrings(visibility.UserIDs) {
		records = append(records, database.TabVisibilityTargetRecord{ID: generatedVisibilityTargetID(tabID, "user", userID), TabID: tabID, TargetType: "user", TargetID: userID})
	}
	if len(records) == 0 {
		return nil
	}
	return tx.Create(&records).Error
}

func (r *PostgresTabRepository) resolveTargetUserIDs(tx *gorm.DB, ownerUserID string, visibility models.TabVisibility) ([]string, error) {
	userIDs := map[string]bool{ownerUserID: true}
	switch visibility.Scope {
	case "self":
	case "company":
		var users []database.UserRecord
		if err := tx.Where("enabled = true").Find(&users).Error; err != nil {
			return nil, err
		}
		for _, user := range users {
			userIDs[user.ID] = true
		}
	case "custom":
		if len(visibility.UserIDs) > 0 {
			userTargets := uniqueStrings(visibility.UserIDs)
			var count int64
			if err := tx.Model(&database.UserRecord{}).Where("id IN ? AND enabled = true", userTargets).Count(&count).Error; err != nil {
				return nil, err
			}
			if count != int64(len(userTargets)) {
				return nil, ErrInvalidTarget
			}
			for _, userID := range userTargets {
				userIDs[userID] = true
			}
		}
		if len(visibility.TeamIDs) > 0 {
			teamTargets := uniqueStrings(visibility.TeamIDs)
			var count int64
			if err := tx.Model(&database.TeamRecord{}).Where("id IN ? AND enabled = true", teamTargets).Count(&count).Error; err != nil {
				return nil, err
			}
			if count != int64(len(teamTargets)) {
				return nil, ErrInvalidTarget
			}
			var members []database.TeamMemberRecord
			if err := tx.Where("team_id IN ? AND enabled = true", teamTargets).Find(&members).Error; err != nil {
				return nil, err
			}
			for _, member := range members {
				userIDs[member.UserID] = true
			}
		}
	}
	result := make([]string, 0, len(userIDs))
	for userID := range userIDs {
		result = append(result, userID)
	}
	return result, nil
}

func (r *PostgresTabRepository) upsertUserTabs(tx *gorm.DB, userIDs []string, tabID string, source string, sortOrder int) error {
	if sortOrder <= 0 {
		sortOrder = 100
	}
	for _, userID := range userIDs {
		record := database.UserTabRecord{UserID: userID, TabID: tabID, Enabled: true, Source: source, SortOrder: sortOrder}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "tab_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"enabled", "source", "sort_order", "updated_at"}),
		}).Create(&record).Error; err != nil {
			return err
		}
	}
	return nil
}

func tabUserTabSource(visibility models.TabVisibility) string {
	if visibility.Scope == "self" {
		return "self"
	}
	return "admin_publish"
}

func generatedVisibilityTargetID(tabID string, targetType string, targetID string) string {
	sum := sha1.Sum([]byte(tabID + "|" + targetType + "|" + targetID))
	return "tvt-" + hex.EncodeToString(sum[:])
}

func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}

func sortTabManifests(tabs []models.TabManifest) {
	sort.SliceStable(tabs, func(i, j int) bool {
		return tabs[i].SortOrder < tabs[j].SortOrder
	})
}
