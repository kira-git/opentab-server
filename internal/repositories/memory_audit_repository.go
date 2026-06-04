package repositories

import "opentab-server/internal/models"

type MemoryAuditRepository struct{}

var memoryAuditLogs []models.AuditLog

func NewMemoryAuditRepository() *MemoryAuditRepository {
	memoryAuditLogs = []models.AuditLog{}
	return &MemoryAuditRepository{}
}

func (r *MemoryAuditRepository) Record(log models.AuditLog) error {
	memoryAuditLogs = append(memoryAuditLogs, log)
	return nil
}

func MemoryAuditLogs() []models.AuditLog {
	return append([]models.AuditLog{}, memoryAuditLogs...)
}
