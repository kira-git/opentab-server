package repositories

import (
	"opentab-server/internal/mockdata"
	"opentab-server/internal/models"
)

type MemoryTabRepository struct{}

func NewMemoryTabRepository() *MemoryTabRepository {
	return &MemoryTabRepository{}
}

func (r *MemoryTabRepository) ListAll() ([]models.TabManifest, error) {
	return mockdata.AllTabs(), nil
}

func (r *MemoryTabRepository) ListByUser(userID string) ([]models.TabManifest, error) {
	return mockdata.TabsForUser(userID), nil
}

func (r *MemoryTabRepository) ListCatalog(userID string) ([]models.TabManifest, error) {
	return mockdata.CatalogForUser(userID), nil
}

func (r *MemoryTabRepository) FindByID(tabID string) (*models.TabManifest, error) {
	tab := mockdata.FindTab(tabID)
	if tab == nil {
		return nil, ErrNotFound
	}
	return tab, nil
}

func (r *MemoryTabRepository) CreateCustom(userID string, tab models.TabManifest) (*models.TabManifest, error) {
	if mockdata.FindTab(tab.ID) != nil {
		return nil, ErrConflict
	}
	created := mockdata.CreateCustomTab(userID, tab)
	return &created, nil
}

func (r *MemoryTabRepository) UpdateCustom(userID string, tabID string, req models.UpdateCustomTabRequest) (*models.TabManifest, error) {
	if mockdata.FindTab(tabID) == nil {
		return nil, ErrNotFound
	}
	if !mockdata.IsCustomTabOwnedBy(userID, tabID) {
		return nil, ErrForbidden
	}
	tab := mockdata.UpdateCustomTab(userID, tabID, req)
	if tab == nil {
		return nil, ErrNotFound
	}
	return tab, nil
}

func (r *MemoryTabRepository) DeleteCustom(userID string, tabID string) error {
	if mockdata.FindTab(tabID) == nil {
		return ErrNotFound
	}
	if !mockdata.IsCustomTabOwnedBy(userID, tabID) {
		return ErrForbidden
	}
	if !mockdata.DeleteCustomTab(userID, tabID) {
		return ErrNotFound
	}
	return nil
}

func (r *MemoryTabRepository) RouteExistsForUser(userID string, route string, excludeTabID string) bool {
	return mockdata.UserRouteExists(userID, route, excludeTabID)
}

func (r *MemoryTabRepository) Reorder(userID string, items []models.ReorderTabItem) error {
	mockdata.ReorderUserTabs(userID, items)
	return nil
}

func (r *MemoryTabRepository) Enable(userID string, tabID string) error {
	mockdata.EnableTab(userID, tabID)
	return nil
}

func (r *MemoryTabRepository) Disable(userID string, tabID string) error {
	mockdata.DisableTab(userID, tabID)
	return nil
}

func (r *MemoryTabRepository) Count() int {
	return len(mockdata.Tabs)
}
