package services

import "opentab-server/internal/models"

func hasPermission(user *models.User, permission string) bool {
	for _, item := range user.Permissions {
		if item == permission {
			return true
		}
	}
	return false
}

func hasAllPermissions(user *models.User, permissions []string) bool {
	for _, permission := range permissions {
		if !hasPermission(user, permission) {
			return false
		}
	}
	return true
}
