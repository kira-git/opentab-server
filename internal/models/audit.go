package models

type AuditLog struct {
	ID         string
	RequestID  string
	UserID     string
	Account    string
	Action     string
	Method     string
	Path       string
	StatusCode int
	Result     string
	ErrorCode  string
	ClientIP   string
	UserAgent  string
	DurationMS int64
	CreatedAt  string
}
