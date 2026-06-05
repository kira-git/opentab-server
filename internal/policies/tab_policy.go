package policies

import "opentab-server/internal/models"

const (
	TabVisibilitySelf    = "self"
	TabVisibilityCompany = "company"
	TabVisibilityCustom  = "custom"
)

func CanPublishTabVisibility(user *models.User, visibility models.TabVisibility) bool {
	if visibility.Scope == "" || visibility.Scope == TabVisibilitySelf {
		return user != nil
	}
	return IsAdmin(user) || HasAnyPermission(user, "tab.admin.manage", "team.manage")
}

func CanViewTab(user *models.User, ownerUserID string, visibility *models.TabVisibility, isSystem bool) bool {
	if user == nil {
		return false
	}
	if isSystem {
		return true
	}
	if ownerUserID != "" && ownerUserID == user.ID {
		return true
	}
	if visibility == nil {
		return false
	}
	switch visibility.Scope {
	case TabVisibilityCompany:
		return true
	case TabVisibilityCustom:
		if contains(visibility.UserIDs, user.ID) {
			return true
		}
		return anyTeamMatches(user, visibility.TeamIDs)
	default:
		return false
	}
}

func CanUseTab(user *models.User, tab models.TabManifest) bool {
	if user == nil {
		return false
	}
	for _, permission := range tab.Permissions {
		if !HasPermission(user, permission) {
			return false
		}
	}
	return true
}

func UserTeamIDs(user *models.User) []string {
	seen := map[string]bool{}
	if user == nil {
		return []string{}
	}
	if user.CurrentTeamID != "" {
		seen[user.CurrentTeamID] = true
	}
	for _, membership := range user.Memberships {
		if membership.TeamID != "" {
			seen[membership.TeamID] = true
		}
	}
	result := make([]string, 0, len(seen))
	for teamID := range seen {
		result = append(result, teamID)
	}
	return result
}

func anyTeamMatches(user *models.User, teamIDs []string) bool {
	for _, userTeamID := range UserTeamIDs(user) {
		if contains(teamIDs, userTeamID) {
			return true
		}
	}
	return false
}
