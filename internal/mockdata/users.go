package mockdata

import "opentab-server/internal/models"

var Users = []models.User{
	{
		ID:          "user-demo",
		Account:     "opentab-demo",
		DisplayName: "OpenTab 演示账号",
		Password:    "demo123",
		Token:       "mock-access-token",
		Permissions: []string{
			"tab.approval.read",
			"tab.calendar.read",
			"ai.oncall",
		},
	},
	{
		ID:          "user-admin",
		Account:     "opentab-admin",
		DisplayName: "OpenTab 管理员",
		Password:    "admin123",
		Token:       "mock-admin-token",
		Permissions: []string{
			"tab.approval.read",
			"tab.calendar.read",
			"tab.finance.read",
			"ai.oncall",
		},
	},
	{
		ID:          "user-guest",
		Account:     "opentab-guest",
		DisplayName: "OpenTab 访客",
		Password:    "guest123",
		Token:       "mock-guest-token",
		Permissions: []string{
			"ai.oncall",
		},
	},
}

func FindUser(account string) *models.User {
	for i := range Users {
		if Users[i].Account == account {
			return &Users[i]
		}
	}
	return nil
}

func ValidateLogin(account string, password string) *models.User {
	user := FindUser(account)
	if user == nil || user.Password != password {
		return nil
	}
	return user
}

func FindUserByToken(token string) *models.User {
	for i := range Users {
		if Users[i].Token == token {
			return &Users[i]
		}
	}
	return nil
}
