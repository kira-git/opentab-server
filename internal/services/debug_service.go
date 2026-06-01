package services

import "opentab-server/internal/repositories"

type DebugService struct {
	debug repositories.DebugRepository
}

func NewDebugService(debug repositories.DebugRepository) *DebugService {
	return &DebugService{debug: debug}
}

func (s *DebugService) ListPermissions() []map[string]string {
	return s.debug.ListPermissions()
}
