package repositories

import "opentab-server/internal/mockdata"

type MemoryDebugRepository struct{}

func NewMemoryDebugRepository() *MemoryDebugRepository {
	return &MemoryDebugRepository{}
}

func (r *MemoryDebugRepository) ListPermissions() []map[string]string {
	return mockdata.Permissions
}
