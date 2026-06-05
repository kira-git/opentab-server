package database

import (
	"time"

	"gorm.io/datatypes"
)

type UserRecord struct {
	ID           string `gorm:"primaryKey;size:64"`
	Account      string `gorm:"uniqueIndex;size:64;not null"`
	DisplayName  string `gorm:"size:128;not null"`
	PasswordHash string `gorm:"size:255;not null"`
	GlobalRole   string `gorm:"index;size:32"`
	Enabled      bool   `gorm:"not null;default:true"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (UserRecord) TableName() string {
	return "users"
}

type TeamRecord struct {
	ID          string `gorm:"primaryKey;size:64"`
	Name        string `gorm:"size:128;not null"`
	Description string `gorm:"type:text"`
	Enabled     bool   `gorm:"not null;default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (TeamRecord) TableName() string {
	return "teams"
}

type TeamMemberRecord struct {
	TeamID    string    `gorm:"primaryKey;size:64;index"`
	UserID    string    `gorm:"primaryKey;size:64;index"`
	TeamRole  string    `gorm:"index;size:32;not null"`
	Enabled   bool      `gorm:"not null;default:true"`
	JoinedAt  time.Time `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (TeamMemberRecord) TableName() string {
	return "team_members"
}

type AuthSessionRecord struct {
	ID        string     `gorm:"primaryKey;size:64"`
	UserID    string     `gorm:"index;size:64;not null"`
	Token     string     `gorm:"uniqueIndex;size:255;not null"`
	ExpiresAt *time.Time `gorm:"index:idx_auth_sessions_expires_revoked,priority:1"`
	RevokedAt *time.Time `gorm:"index:idx_auth_sessions_expires_revoked,priority:2"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (AuthSessionRecord) TableName() string {
	return "auth_sessions"
}

type PermissionRecord struct {
	Code        string `gorm:"primaryKey;size:128"`
	Description string `gorm:"size:255;not null"`
	CreatedAt   time.Time
}

func (PermissionRecord) TableName() string {
	return "permissions"
}

type UserPermissionRecord struct {
	UserID         string `gorm:"primaryKey;size:64"`
	PermissionCode string `gorm:"primaryKey;size:128;index"`
	CreatedAt      time.Time
}

func (UserPermissionRecord) TableName() string {
	return "user_permissions"
}

type TabRecord struct {
	ID                  string         `gorm:"primaryKey;size:128"`
	OwnerUserID         *string        `gorm:"index;size:64"`
	VisibilityScope     string         `gorm:"index;size:32;not null;default:self"`
	DefaultEnabled      bool           `gorm:"not null;default:true"`
	ManagedByAdmin      bool           `gorm:"not null;default:false"`
	DisplayName         string         `gorm:"size:128;not null"`
	Description         string         `gorm:"type:text"`
	Icon                string         `gorm:"size:64"`
	Route               string         `gorm:"size:255;not null"`
	EntryType           string         `gorm:"index;size:32;not null"`
	EntryURI            string         `gorm:"type:text"`
	VersionMajor        int            `gorm:"not null"`
	VersionMinor        int            `gorm:"not null"`
	VersionPatch        int            `gorm:"not null"`
	MinContainerVersion int            `gorm:"not null"`
	PermissionsJSON     datatypes.JSON `gorm:"type:jsonb;not null"`
	ExtensionJSON       datatypes.JSON `gorm:"type:jsonb"`
	ExtraConfigJSON     datatypes.JSON `gorm:"type:jsonb"`
	IsSystem            bool           `gorm:"not null;default:false"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (TabRecord) TableName() string {
	return "tabs"
}

type TabVisibilityTargetRecord struct {
	ID         string `gorm:"primaryKey;size:64"`
	TabID      string `gorm:"index:idx_tab_visibility_target,priority:1;size:128;not null"`
	TargetType string `gorm:"index:idx_tab_visibility_target,priority:2;size:32;not null"`
	TargetID   string `gorm:"index:idx_tab_visibility_target,priority:3;size:64;not null"`
	CreatedAt  time.Time
}

func (TabVisibilityTargetRecord) TableName() string {
	return "tab_visibility_targets"
}

type UserTabRecord struct {
	UserID    string `gorm:"primaryKey;size:64"`
	TabID     string `gorm:"primaryKey;size:128;index"`
	Enabled   bool   `gorm:"not null;default:true"`
	Source    string `gorm:"index;size:32;not null;default:self"`
	SortOrder int    `gorm:"index;not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (UserTabRecord) TableName() string {
	return "user_tabs"
}

type OnCallSessionRecord struct {
	ID        string `gorm:"primaryKey;size:64"`
	UserID    string `gorm:"index:idx_oncall_sessions_user_updated,priority:1;size:64;not null"`
	Title     string `gorm:"size:255;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time `gorm:"index:idx_oncall_sessions_user_updated,priority:2"`
	DeletedAt *time.Time
}

func (OnCallSessionRecord) TableName() string {
	return "oncall_sessions"
}

type OnCallMessageRecord struct {
	ID          string    `gorm:"primaryKey;size:64"`
	SessionID   string    `gorm:"index:idx_oncall_messages_session_created,priority:1;size:64;not null"`
	Role        string    `gorm:"index;size:32;not null"`
	Content     string    `gorm:"type:text;not null"`
	ContentType string    `gorm:"size:32;not null;default:text"`
	CreatedAt   time.Time `gorm:"index:idx_oncall_messages_session_created,priority:2"`
}

func (OnCallMessageRecord) TableName() string {
	return "oncall_messages"
}

type ApprovalItemRecord struct {
	ID          string `gorm:"primaryKey;size:64"`
	UserID      string `gorm:"index:idx_approval_user_status_created,priority:1;size:64"`
	TeamID      string `gorm:"index:idx_approval_team_status_created,priority:1;size:64"`
	Type        string `gorm:"size:64"`
	Title       string `gorm:"size:255;not null"`
	ApplicantID string `gorm:"index;size:64"`
	Applicant   string `gorm:"size:128;not null"`
	ApproverID  string `gorm:"index;size:64"`
	Approver    string `gorm:"size:128"`
	Amount      int
	Reason      string         `gorm:"type:text"`
	Summary     string         `gorm:"type:text"`
	FormJSON    datatypes.JSON `gorm:"type:jsonb"`
	Status      string         `gorm:"index:idx_approval_user_status_created,priority:2;index:idx_approval_team_status_created,priority:2;size:32;not null"`
	Comment     string         `gorm:"type:text"`
	CreatedAt   time.Time      `gorm:"index:idx_approval_user_status_created,priority:3;index:idx_approval_team_status_created,priority:3"`
	UpdatedAt   time.Time
}

func (ApprovalItemRecord) TableName() string {
	return "approval_items"
}

type CalendarEventRecord struct {
	ID                 string         `gorm:"primaryKey;size:64"`
	UserID             string         `gorm:"index:idx_calendar_user_time_range,priority:1;size:64"`
	TeamID             string         `gorm:"index:idx_calendar_team_time_range,priority:1;size:64"`
	Visibility         string         `gorm:"index;size:32;not null;default:team"`
	CreatorID          string         `gorm:"index;size:64"`
	CreatorName        string         `gorm:"size:128"`
	Title              string         `gorm:"size:255;not null"`
	Description        string         `gorm:"type:text"`
	StartTime          time.Time      `gorm:"index:idx_calendar_user_time_range,priority:2;not null"`
	EndTime            time.Time      `gorm:"index:idx_calendar_user_time_range,priority:3;not null"`
	Location           string         `gorm:"size:255"`
	ParticipantsJSON   datatypes.JSON `gorm:"type:jsonb"`
	ParticipantIDsJSON datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (CalendarEventRecord) TableName() string {
	return "calendar_events"
}

type AnnouncementRecord struct {
	ID            string     `gorm:"primaryKey;size:64"`
	TeamID        string     `gorm:"index:idx_announcements_scope_team_created,priority:2;size:64"`
	Scope         string     `gorm:"index:idx_announcements_scope_team_created,priority:1;size:32;not null"`
	Title         string     `gorm:"size:255;not null"`
	Content       string     `gorm:"type:text;not null"`
	PublisherID   string     `gorm:"index;size:64;not null"`
	PublisherName string     `gorm:"size:128;not null"`
	Pinned        bool       `gorm:"not null;default:false"`
	DeletedAt     *time.Time `gorm:"index"`
	CreatedAt     time.Time  `gorm:"index:idx_announcements_scope_team_created,priority:3"`
	UpdatedAt     time.Time
}

func (AnnouncementRecord) TableName() string {
	return "announcements"
}

type AuditLogRecord struct {
	ID         string    `gorm:"primaryKey;size:64"`
	RequestID  string    `gorm:"index;size:64;not null"`
	UserID     string    `gorm:"index:idx_audit_user_created,priority:1;size:64"`
	Account    string    `gorm:"size:64"`
	Action     string    `gorm:"index;size:64;not null"`
	Method     string    `gorm:"size:16;not null"`
	Path       string    `gorm:"index;size:255;not null"`
	StatusCode int       `gorm:"index;not null"`
	Result     string    `gorm:"index;size:32;not null"`
	ErrorCode  string    `gorm:"size:64"`
	ClientIP   string    `gorm:"size:64"`
	UserAgent  string    `gorm:"type:text"`
	DurationMS int64     `gorm:"not null"`
	CreatedAt  time.Time `gorm:"index:idx_audit_user_created,priority:2"`
}

func (AuditLogRecord) TableName() string {
	return "audit_logs"
}
