package repositories

import "gorm.io/gorm"

type RepositorySet struct {
	Users    UserRepository
	Tabs     TabRepository
	Business BusinessRepository
	OnCall   OnCallRepository
	Debug    DebugRepository
}

func NewMemoryRepositorySet() RepositorySet {
	return RepositorySet{
		Users:    NewMemoryUserRepository(),
		Tabs:     NewMemoryTabRepository(),
		Business: NewMemoryBusinessRepository(),
		OnCall:   NewMemoryOnCallRepository(),
		Debug:    NewMemoryDebugRepository(),
	}
}

func NewPostgresRepositorySet(db *gorm.DB) RepositorySet {
	return RepositorySet{
		Users:    NewPostgresUserRepository(db),
		Tabs:     NewPostgresTabRepository(db),
		Business: NewPostgresBusinessRepository(db),
		OnCall:   NewPostgresOnCallRepository(db),
		Debug:    NewPostgresDebugRepository(db),
	}
}
