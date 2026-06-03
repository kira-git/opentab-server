package services

import (
	"opentab-server/internal/models"
	"opentab-server/internal/policies"
)

func hasPermission(user *models.User, permission string) bool {
	return policies.HasPermission(user, permission)
}

func hasAllPermissions(user *models.User, permissions []string) bool {
	for _, permission := range permissions {
		if !hasPermission(user, permission) {
			return false
		}
	}
	return true
}
