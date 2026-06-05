package services

import (
	"opentab-server/internal/models"
	"opentab-server/internal/policies"
)

func hasPermission(user *models.User, permission string) bool {
	return policies.HasPermission(user, permission)
}
