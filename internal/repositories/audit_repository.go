package repositories

import "opentab-server/internal/models"

type AuditRepository interface {
	Record(log models.AuditLog) error
}
