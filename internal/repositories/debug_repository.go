package repositories

type DebugRepository interface {
	ListPermissions() []map[string]string
}
