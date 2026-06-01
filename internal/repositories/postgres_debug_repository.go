package repositories

import (
	"opentab-server/internal/database"

	"gorm.io/gorm"
)

type PostgresDebugRepository struct {
	db *gorm.DB
}

func NewPostgresDebugRepository(db *gorm.DB) *PostgresDebugRepository {
	return &PostgresDebugRepository{db: db}
}

func (r *PostgresDebugRepository) ListPermissions() []map[string]string {
	var records []database.PermissionRecord
	if err := r.db.Order("code ASC").Find(&records).Error; err != nil {
		return []map[string]string{}
	}
	result := make([]map[string]string, 0, len(records))
	for _, record := range records {
		result = append(result, map[string]string{"code": record.Code, "description": record.Description})
	}
	return result
}
