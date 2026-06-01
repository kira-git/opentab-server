package repositories

import "opentab-server/internal/models"

type TabRepository interface {
	ListAll() ([]models.TabManifest, error)
	ListByUser(userID string) ([]models.TabManifest, error)
	ListCatalog(userID string) ([]models.TabManifest, error)
	FindByID(tabID string) (*models.TabManifest, error)
	CreateCustom(userID string, tab models.TabManifest) (*models.TabManifest, error)
	UpdateCustom(userID string, tabID string, req models.UpdateCustomTabRequest) (*models.TabManifest, error)
	DeleteCustom(userID string, tabID string) error
	RouteExistsForUser(userID string, route string, excludeTabID string) bool
	Reorder(userID string, items []models.ReorderTabItem) error
	Enable(userID string, tabID string) error
	Disable(userID string, tabID string) error
	Count() int
}
