package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(databaseURL string) (*gorm.DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database url is empty")
	}
	return gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
}

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&UserRecord{},
		&TeamRecord{},
		&TeamMemberRecord{},
		&AuthSessionRecord{},
		&PermissionRecord{},
		&UserPermissionRecord{},
		&TabRecord{},
		&UserTabRecord{},
		&OnCallSessionRecord{},
		&OnCallMessageRecord{},
		&ApprovalItemRecord{},
		&CalendarEventRecord{},
		&AnnouncementRecord{},
	); err != nil {
		return err
	}
	return db.Exec("ALTER TABLE tabs ALTER COLUMN is_system SET DEFAULT false").Error
}
