package repositories

import "opentab-server/internal/models"

type TabRepository interface {
	ListAll() ([]models.TabManifest, error)
	ListByUser(user *models.User) ([]models.TabManifest, error)
	ListCatalog(user *models.User) ([]models.TabManifest, error)
	FindByID(tabID string) (*models.TabManifest, error)
	FindVisibleByID(user *models.User, tabID string) (*models.TabManifest, error)
	CreateCustom(user *models.User, tab models.TabManifest, visibility models.TabVisibility) (*models.TabManifest, error)
	UpdateCustom(user *models.User, tabID string, req models.UpdateCustomTabRequest, visibility *models.TabVisibility) (*models.TabManifest, error)
	DeleteCustom(userID string, tabID string) error
	RouteExistsForUser(userID string, route string, excludeTabID string) bool
	Reorder(userID string, items []models.ReorderTabItem) error
	Enable(userID string, tabID string) error
	Disable(userID string, tabID string) error
	Count() int
}
