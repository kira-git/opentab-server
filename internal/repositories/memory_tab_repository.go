package repositories

import (
	"opentab-server/internal/mockdata"
	"opentab-server/internal/models"
	"opentab-server/internal/policies"
)

type MemoryTabRepository struct{}

func NewMemoryTabRepository() *MemoryTabRepository {
	return &MemoryTabRepository{}
}

func (r *MemoryTabRepository) ListAll() ([]models.TabManifest, error) {
	return mockdata.AllTabs(), nil
}

func (r *MemoryTabRepository) ListByUser(user *models.User) ([]models.TabManifest, error) {
	if user == nil {
		return nil, ErrForbidden
	}
	copyUser := *user
	applyMemoryMembershipForTab(&copyUser)
	userTabs := mockdata.UserTabs[copyUser.ID]
	tabs := mockdata.AllTabs()
	result := make([]models.TabManifest, 0, len(tabs))
	for _, tab := range tabs {
		if !memoryTabVisibleToUser(&copyUser, &tab) {
			continue
		}
		enabled, hasUserTab := userTabs[tab.ID]
		if hasUserTab && !enabled {
			continue
		}
		if !hasUserTab && (tab.Visibility == nil || !tab.Visibility.DefaultEnabled) {
			continue
		}
		tab.Enabled = true
		result = append(result, tab)
	}
	return result, nil
}

func (r *MemoryTabRepository) ListCatalog(user *models.User) ([]models.TabManifest, error) {
	if user == nil {
		return nil, ErrForbidden
	}
	copyUser := *user
	applyMemoryMembershipForTab(&copyUser)
	userTabs := mockdata.UserTabs[copyUser.ID]
	tabs := mockdata.AllTabs()
	result := make([]models.TabManifest, 0, len(tabs))
	for _, tab := range tabs {
		if !memoryTabVisibleToUser(&copyUser, &tab) {
			continue
		}
		enabled, hasUserTab := userTabs[tab.ID]
		tab.Enabled = enabled || (!hasUserTab && tab.Visibility != nil && tab.Visibility.DefaultEnabled)
		result = append(result, tab)
	}
	return result, nil
}

func (r *MemoryTabRepository) FindByID(tabID string) (*models.TabManifest, error) {
	tab := mockdata.FindTab(tabID)
	if tab == nil {
		return nil, ErrNotFound
	}
	return tab, nil
}

func (r *MemoryTabRepository) FindVisibleByID(user *models.User, tabID string) (*models.TabManifest, error) {
	tab, err := r.FindByID(tabID)
	if err != nil {
		return nil, err
	}
	copyUser := *user
	applyMemoryMembershipForTab(&copyUser)
	if !memoryTabVisibleToUser(&copyUser, tab) {
		return nil, ErrForbidden
	}
	return tab, nil
}

func (r *MemoryTabRepository) CreateCustom(user *models.User, tab models.TabManifest, visibility models.TabVisibility) (*models.TabManifest, error) {
	if mockdata.FindTab(tab.ID) != nil {
		return nil, ErrConflict
	}
	tab.Visibility = &visibility
	created := mockdata.CreateCustomTab(user.ID, tab)
	if visibility.DefaultEnabled {
		for _, userID := range memoryTargetUserIDs(user, visibility) {
			mockdata.EnableTab(userID, tab.ID)
		}
	}
	return &created, nil
}

func (r *MemoryTabRepository) UpdateCustom(user *models.User, tabID string, req models.UpdateCustomTabRequest, visibility *models.TabVisibility) (*models.TabManifest, error) {
	if mockdata.FindTab(tabID) == nil {
		return nil, ErrNotFound
	}
	if !mockdata.IsCustomTabOwnedBy(user.ID, tabID) {
		return nil, ErrForbidden
	}
	tab := mockdata.UpdateCustomTab(user.ID, tabID, req)
	if tab == nil {
		return nil, ErrNotFound
	}
	if visibility != nil {
		tab.Visibility = visibility
		for i := range mockdata.Tabs {
			if mockdata.Tabs[i].ID == tabID {
				mockdata.Tabs[i].Visibility = visibility
				break
			}
		}
		if visibility.DefaultEnabled {
			for _, userID := range memoryTargetUserIDs(user, *visibility) {
				mockdata.EnableTab(userID, tabID)
			}
		}
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

func memoryTabVisibleToUser(user *models.User, tab *models.TabManifest) bool {
	if user == nil || tab == nil {
		return false
	}
	owner := mockdata.CustomTabOwners[tab.ID]
	return policies.CanViewTab(user, owner, tab.Visibility, owner == "")
}

func memoryTargetUserIDs(owner *models.User, visibility models.TabVisibility) []string {
	userIDs := map[string]bool{}
	userIDs[owner.ID] = true
	switch visibility.Scope {
	case "company":
		for _, user := range mockdata.Users {
			userIDs[user.ID] = true
		}
	case "custom":
		for _, userID := range visibility.UserIDs {
			userIDs[userID] = true
		}
		for _, user := range mockdata.Users {
			copy := user
			applyMemoryMembershipForTab(&copy)
			for _, membership := range copy.Memberships {
				if containsString(visibility.TeamIDs, membership.TeamID) {
					userIDs[copy.ID] = true
				}
			}
		}
	}
	result := make([]string, 0, len(userIDs))
	for userID := range userIDs {
		result = append(result, userID)
	}
	return result
}

func applyMemoryMembershipForTab(user *models.User) {
	switch user.ID {
	case "user-product-manager":
		user.Memberships = []models.TeamMembership{{TeamID: "team-product", TeamName: "产品研发部", TeamRole: "manager"}}
	case "user-product-employee", "user-demo":
		user.Memberships = []models.TeamMembership{{TeamID: "team-product", TeamName: "产品研发部", TeamRole: "employee"}}
	case "user-operation-manager":
		user.Memberships = []models.TeamMembership{{TeamID: "team-operation", TeamName: "运营支持部", TeamRole: "manager"}}
	case "user-operation-employee":
		user.Memberships = []models.TeamMembership{{TeamID: "team-operation", TeamName: "运营支持部", TeamRole: "employee"}}
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
