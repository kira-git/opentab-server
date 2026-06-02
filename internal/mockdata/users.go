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
		Account:     "admin",
		DisplayName: "系统管理员",
		Password:    "admin123",
		Token:       "mock-admin-token",
		GlobalRole:  "admin",
		Permissions: []string{
			"team.manage",
			"team.all.read",
			"tab.admin.manage",
			"tab.debug.read",
			"tab.company.read",
			"tab.announcement.read",
			"tab.announcement.write",
			"tab.fun.read",
			"tab.approval.read",
			"tab.approval.create",
			"tab.approval.approve",
			"tab.approval.all",
			"tab.calendar.read",
			"tab.calendar.create",
			"tab.calendar.manage",
			"tab.calendar.all",
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
	{
		ID:            "user-product-manager",
		Account:       "product-manager",
		DisplayName:   "产品主管",
		Password:      "manager123",
		Token:         "mock-product-manager-token",
		CurrentTeamID: "team-product",
		Permissions:   managerPermissions(),
	},
	{
		ID:            "user-product-employee",
		Account:       "product-employee",
		DisplayName:   "产品员工",
		Password:      "employee123",
		Token:         "mock-product-employee-token",
		CurrentTeamID: "team-product",
		Permissions:   employeePermissions(),
	},
	{
		ID:            "user-operation-manager",
		Account:       "operation-manager",
		DisplayName:   "运营主管",
		Password:      "manager123",
		Token:         "mock-operation-manager-token",
		CurrentTeamID: "team-operation",
		Permissions:   managerPermissions(),
	},
	{
		ID:            "user-operation-employee",
		Account:       "operation-employee",
		DisplayName:   "运营员工",
		Password:      "employee123",
		Token:         "mock-operation-employee-token",
		CurrentTeamID: "team-operation",
		Permissions:   employeePermissions(),
	},
}

func managerPermissions() []string {
	return []string{
		"team.member.read",
		"tab.company.read",
		"tab.announcement.read",
		"tab.announcement.write",
		"tab.fun.read",
		"tab.approval.read",
		"tab.approval.create",
		"tab.approval.approve",
		"tab.calendar.read",
		"tab.calendar.create",
		"tab.calendar.manage",
		"ai.oncall",
	}
}

func employeePermissions() []string {
	return []string{
		"tab.company.read",
		"tab.announcement.read",
		"tab.fun.read",
		"tab.approval.read",
		"tab.approval.create",
		"tab.calendar.read",
		"ai.oncall",
	}
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
