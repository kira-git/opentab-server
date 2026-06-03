package policies

import "opentab-server/internal/models"

func HasPermission(user *models.User, permission string) bool {
	if user == nil {
		return false
	}
	for _, item := range user.Permissions {
		if item == permission {
			return true
		}
	}
	return false
}

func HasAnyPermission(user *models.User, permissions ...string) bool {
	for _, permission := range permissions {
		if HasPermission(user, permission) {
			return true
		}
	}
	return false
}

func IsAdmin(user *models.User) bool {
	return user != nil && user.GlobalRole == "admin"
}

func InTeam(user *models.User, teamID string) bool {
	if user == nil || teamID == "" {
		return false
	}
	for _, membership := range user.Memberships {
		if membership.TeamID == teamID {
			return true
		}
	}
	return false
}

func HasTeamRole(user *models.User, teamID string, role string) bool {
	if user == nil || teamID == "" {
		return false
	}
	for _, membership := range user.Memberships {
		if membership.TeamID == teamID && membership.TeamRole == role {
			return true
		}
	}
	return false
}

func CanReadTeamMembers(user *models.User) bool {
	return HasAnyPermission(user, "team.manage", "team.member.read")
}

func CanManageTeam(user *models.User) bool {
	return HasPermission(user, "team.manage")
}

func CanCreateApproval(user *models.User, teamID string) bool {
	return HasPermission(user, "tab.approval.create") && (IsAdmin(user) || InTeam(user, teamID))
}

func CanViewApproval(user *models.User, applicantID string, ownerUserID string, teamID string) bool {
	return IsAdmin(user) || userOwns(user, applicantID) || userOwns(user, ownerUserID) || HasTeamRole(user, teamID, "manager")
}

func CanApproveTeamApproval(user *models.User, teamID string) bool {
	return IsAdmin(user) || HasTeamRole(user, teamID, "manager")
}

func CanCancelApproval(user *models.User, applicantID string, ownerUserID string) bool {
	return userOwns(user, applicantID) || userOwns(user, ownerUserID)
}

func CanViewCalendar(user *models.User, visibility string, teamID string, creatorID string, participantIDs []string) bool {
	return IsAdmin(user) || visibility == "company" || InTeam(user, teamID) || userOwns(user, creatorID) || contains(participantIDs, userID(user))
}

func CanManageCalendar(user *models.User, teamID string) bool {
	return IsAdmin(user) || HasTeamRole(user, teamID, "manager")
}

func CanViewAnnouncement(user *models.User, scope string, teamID string) bool {
	return IsAdmin(user) || scope == "company" || InTeam(user, teamID)
}

func CanWriteAnnouncement(user *models.User, scope string, teamID string) bool {
	if IsAdmin(user) {
		return true
	}
	if scope == "company" {
		return false
	}
	return HasTeamRole(user, teamID, "manager")
}

func CanManageAnnouncement(user *models.User, teamID string) bool {
	return IsAdmin(user) || HasTeamRole(user, teamID, "manager")
}

func userOwns(user *models.User, ownerID string) bool {
	return user != nil && ownerID != "" && user.ID == ownerID
}

func userID(user *models.User) string {
	if user == nil {
		return ""
	}
	return user.ID
}

func contains(items []string, target string) bool {
	if target == "" {
		return false
	}
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
