package policies

import (
	"testing"

	"opentab-server/internal/models"
)

func TestCanPublishTabVisibility(t *testing.T) {
	admin := &models.User{ID: "admin", GlobalRole: "admin"}
	manager := &models.User{ID: "manager", Permissions: []string{"team.manage"}}
	employee := &models.User{ID: "employee"}

	if !CanPublishTabVisibility(employee, models.TabVisibility{Scope: "self"}) {
		t.Fatalf("expected normal user to publish self tab")
	}
	if CanPublishTabVisibility(employee, models.TabVisibility{Scope: "company"}) {
		t.Fatalf("expected normal user not to publish company tab")
	}
	if !CanPublishTabVisibility(admin, models.TabVisibility{Scope: "company"}) {
		t.Fatalf("expected admin to publish company tab")
	}
	if !CanPublishTabVisibility(manager, models.TabVisibility{Scope: "custom", TeamIDs: []string{"team-product"}}) {
		t.Fatalf("expected manager permission to publish custom tab")
	}
}

func TestCanViewTabByVisibilityScope(t *testing.T) {
	user := &models.User{
		ID:            "user-product-employee",
		CurrentTeamID: "team-product",
		Memberships:   []models.TeamMembership{{TeamID: "team-product"}},
	}

	cases := []struct {
		name       string
		ownerID    string
		visibility *models.TabVisibility
		isSystem   bool
		want       bool
	}{
		{name: "system tab", isSystem: true, want: true},
		{name: "owner tab", ownerID: "user-product-employee", visibility: &models.TabVisibility{Scope: "self"}, want: true},
		{name: "company tab", ownerID: "admin", visibility: &models.TabVisibility{Scope: "company"}, want: true},
		{name: "target user tab", ownerID: "admin", visibility: &models.TabVisibility{Scope: "custom", UserIDs: []string{"user-product-employee"}}, want: true},
		{name: "target team tab", ownerID: "admin", visibility: &models.TabVisibility{Scope: "custom", TeamIDs: []string{"team-product"}}, want: true},
		{name: "other team tab", ownerID: "admin", visibility: &models.TabVisibility{Scope: "custom", TeamIDs: []string{"team-operation"}}, want: false},
		{name: "self scope from other owner", ownerID: "admin", visibility: &models.TabVisibility{Scope: "self"}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CanViewTab(user, tc.ownerID, tc.visibility, tc.isSystem)
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestCanUseTabRequiresAllPermissions(t *testing.T) {
	user := &models.User{ID: "user-demo", Permissions: []string{"tab.approval.read", "ai.oncall"}}
	tab := models.TabManifest{Permissions: []string{"tab.approval.read", "ai.oncall"}}
	if !CanUseTab(user, tab) {
		t.Fatalf("expected user with all permissions to use tab")
	}

	tab.Permissions = []string{"tab.approval.read", "tab.admin.manage"}
	if CanUseTab(user, tab) {
		t.Fatalf("expected missing permission to deny tab usage")
	}
}
