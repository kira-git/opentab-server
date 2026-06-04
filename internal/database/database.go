package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PoolOptions struct {
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetimeMin int
}

func Connect(databaseURL string, opts PoolOptions) (*gorm.DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database url is empty")
	}
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if opts.MaxOpenConns <= 0 {
		opts.MaxOpenConns = 20
	}
	if opts.MaxIdleConns <= 0 {
		opts.MaxIdleConns = 10
	}
	if opts.ConnMaxLifetimeMin <= 0 {
		opts.ConnMaxLifetimeMin = 30
	}
	sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
	sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(opts.ConnMaxLifetimeMin) * time.Minute)
	return db, nil
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
		&AuditLogRecord{},
	); err != nil {
		return err
	}
	return db.Exec("ALTER TABLE tabs ALTER COLUMN is_system SET DEFAULT false").Error
}
