package mockdata

import (
	"encoding/json"
	"sort"
	"strings"

	"opentab-server/internal/models"
)

var Tabs = []models.TabManifest{
	{
		ID:                  "company-intro",
		DisplayName:         "公司介绍",
		Description:         "查看企业信息、组织说明和项目背景。",
		Icon:                "company",
		Route:               "/company",
		EntryType:           "native",
		EntryURI:            "native://company",
		Version:             models.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		MinContainerVersion: 1,
		Permissions:         []string{"tab.company.read"},
		Enabled:             true,
		SortOrder:           5,
		ExtraConfig:         json.RawMessage(`{"scope":"company"}`),
	},
	{
		ID:                  "announcements",
		DisplayName:         "集体公告",
		Description:         "查看全公司公告和团队公告。",
		Icon:                "docs",
		Route:               "/announcements",
		EntryType:           "native",
		EntryURI:            "native://announcements",
		Version:             models.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		MinContainerVersion: 1,
		Permissions:         []string{"tab.announcement.read"},
		Enabled:             true,
		SortOrder:           8,
	},
	{
		ID:                  "fun",
		DisplayName:         "放松一刻",
		Description:         "团队间歇放松和内容展示。",
		Icon:                "fun",
		Route:               "/fun",
		EntryType:           "native",
		EntryURI:            "native://fun",
		Version:             models.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		MinContainerVersion: 1,
		Permissions:         []string{"tab.fun.read"},
		Enabled:             true,
		SortOrder:           9,
	},
	{
		ID:                  "approval",
		DisplayName:         "审批中心",
		Description:         "处理待审批、已审批和发起审批。",
		Icon:                "approval",
		Route:               "/approval",
		EntryType:           "native",
		EntryURI:            "native://approval",
		Version:             models.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		MinContainerVersion: 1,
		Permissions:         []string{"tab.approval.read"},
		Enabled:             true,
		SortOrder:           10,
		Extension: &models.TabExtension{
			TitleBar: &models.TitleBarExtension{
				RightText: "刷新",
				MenuItems: []models.MenuItem{
					{ID: "filter", Label: "筛选"},
					{ID: "stats", Label: "统计"},
				},
			},
			Fab: &models.FabExtension{
				ID:    "create",
				Icon:  "add",
				Label: "发起",
			},
		},
		ExtraConfig: json.RawMessage(`{"mockBusinessId":"approval-demo"}`),
	},
	{
		ID:                  "permission-admin",
		DisplayName:         "权限管理",
		Description:         "管理员查看团队、成员和权限关系。",
		Icon:                "admin",
		Route:               "/permission-admin",
		EntryType:           "native",
		EntryURI:            "native://permission-admin",
		Version:             models.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		MinContainerVersion: 1,
		Permissions:         []string{"tab.admin.manage"},
		Enabled:             true,
		SortOrder:           60,
	},
	{
		ID:                  "calendar",
		DisplayName:         "团队日程",
		Description:         "查看今日会议、团队日程和待参加事项。",
		Icon:                "calendar",
		Route:               "/calendar",
		EntryType:           "native",
		EntryURI:            "native://calendar",
		Version:             models.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		MinContainerVersion: 1,
		Permissions:         []string{"tab.calendar.read"},
		Enabled:             true,
		SortOrder:           20,
		Extension: &models.TabExtension{
			TitleBar: &models.TitleBarExtension{
				RightText: "今天",
				MenuItems: []models.MenuItem{
					{ID: "month", Label: "月视图"},
				},
			},
		},
		ExtraConfig: json.RawMessage(`{"mockBusinessId":"calendar-demo"}`),
	},
	{
		ID:                  "finance",
		DisplayName:         "财务看板",
		Description:         "查看费用、报销和经营指标。",
		Icon:                "finance",
		Route:               "/finance",
		EntryType:           "native",
		EntryURI:            "native://finance",
		Version:             models.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		MinContainerVersion: 1,
		Permissions:         []string{"tab.finance.read"},
		Enabled:             true,
		SortOrder:           30,
		Extension: &models.TabExtension{
			TitleBar: &models.TitleBarExtension{
				MenuItems: []models.MenuItem{
					{ID: "export", Label: "导出"},
				},
			},
		},
		ExtraConfig: json.RawMessage(`{"mockBusinessId":"finance-demo"}`),
	},
	{
		ID:                  "next",
		DisplayName:         "新版实验 Tab",
		Description:         "演示容器版本不足时的受限状态。",
		Icon:                "next",
		Route:               "/next",
		EntryType:           "native",
		EntryURI:            "native://next",
		Version:             models.SemanticVersion{Major: 2, Minor: 0, Patch: 0},
		MinContainerVersion: 2,
		Permissions:         []string{},
		Enabled:             true,
		SortOrder:           40,
		Extension: &models.TabExtension{
			TitleBar: &models.TitleBarExtension{
				MenuItems: []models.MenuItem{},
			},
		},
	},
	{
		ID:                  "docs",
		DisplayName:         "接入文档",
		Description:         "通过 WebView 演示接入一个全新的网页 Tab。",
		Icon:                "docs",
		Route:               "/docs",
		EntryType:           "web",
		EntryURI:            "https://example.com/opentab/docs",
		Version:             models.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		MinContainerVersion: 1,
		Permissions:         []string{},
		Enabled:             true,
		SortOrder:           50,
		Extension: &models.TabExtension{
			TitleBar: &models.TitleBarExtension{
				RightText: "打开",
				MenuItems: []models.MenuItem{},
			},
		},
		ExtraConfig: json.RawMessage(`{"source":"mock-web-tab"}`),
	},
}

var UserTabs = map[string]map[string]bool{
	"user-demo": {
		"approval": true,
		"calendar": true,
		"next":     true,
	},
	"user-admin": {
		"company-intro":    true,
		"announcements":    true,
		"fun":              true,
		"approval":         true,
		"calendar":         true,
		"permission-admin": true,
		"next":             true,
		"docs":             true,
	},
	"user-guest": {
		"docs": true,
	},
	"user-product-manager": {
		"company-intro": true,
		"announcements": true,
		"fun":           true,
		"approval":      true,
		"calendar":      true,
	},
	"user-product-employee": {
		"company-intro": true,
		"announcements": true,
		"fun":           true,
		"approval":      true,
		"calendar":      true,
	},
	"user-operation-manager": {
		"company-intro": true,
		"announcements": true,
		"fun":           true,
		"approval":      true,
		"calendar":      true,
	},
	"user-operation-employee": {
		"company-intro": true,
		"announcements": true,
		"fun":           true,
		"approval":      true,
		"calendar":      true,
	},
}

var CustomTabOwners = map[string]string{}

func FindTab(id string) *models.TabManifest {
	for i := range Tabs {
		if Tabs[i].ID == id {
			return &Tabs[i]
		}
	}
	return nil
}

func AllTabs() []models.TabManifest {
	result := make([]models.TabManifest, len(Tabs))
	copy(result, Tabs)
	sortTabs(result)
	return result
}

func TabsForUser(userID string) []models.TabManifest {
	enabled := UserTabs[userID]
	result := make([]models.TabManifest, 0)
	for _, tab := range Tabs {
		if enabled[tab.ID] {
			result = append(result, tab)
		}
	}
	sortTabs(result)
	return result
}

func CatalogForUser(userID string) []models.TabManifest {
	enabled := UserTabs[userID]
	result := make([]models.TabManifest, 0, len(Tabs))
	for _, tab := range Tabs {
		copy := tab
		copy.Enabled = enabled[tab.ID]
		result = append(result, copy)
	}
	sortTabs(result)
	return result
}

func EnableTab(userID string, tabID string) {
	if UserTabs[userID] == nil {
		UserTabs[userID] = map[string]bool{}
	}
	UserTabs[userID][tabID] = true
}

func DisableTab(userID string, tabID string) {
	if UserTabs[userID] == nil {
		UserTabs[userID] = map[string]bool{}
	}
	UserTabs[userID][tabID] = false
}

func CreateCustomTab(userID string, tab models.TabManifest) models.TabManifest {
	tab.Enabled = true
	if tab.SortOrder == 0 {
		tab.SortOrder = nextSortOrder(userID)
	}
	Tabs = append(Tabs, tab)
	CustomTabOwners[tab.ID] = userID
	EnableTab(userID, tab.ID)
	return tab
}

func UpdateCustomTab(userID string, tabID string, req models.UpdateCustomTabRequest) *models.TabManifest {
	for i := range Tabs {
		if Tabs[i].ID == tabID {
			if CustomTabOwners[tabID] != userID {
				return nil
			}
			if strings.TrimSpace(req.DisplayName) != "" {
				Tabs[i].DisplayName = req.DisplayName
			}
			Tabs[i].Description = req.Description
			if strings.TrimSpace(req.Icon) != "" {
				Tabs[i].Icon = req.Icon
			}
			if strings.TrimSpace(req.EntryURI) != "" {
				Tabs[i].EntryURI = req.EntryURI
			}
			if req.SortOrder > 0 {
				Tabs[i].SortOrder = req.SortOrder
			}
			copy := Tabs[i]
			copy.Enabled = UserTabs[userID][tabID]
			return &copy
		}
	}
	return nil
}

func DeleteCustomTab(userID string, tabID string) bool {
	if CustomTabOwners[tabID] != userID {
		return false
	}
	for i := range Tabs {
		if Tabs[i].ID == tabID {
			Tabs = append(Tabs[:i], Tabs[i+1:]...)
			break
		}
	}
	delete(CustomTabOwners, tabID)
	for existingUserID := range UserTabs {
		DisableTab(existingUserID, tabID)
	}
	return true
}

func IsCustomTabOwnedBy(userID string, tabID string) bool {
	return CustomTabOwners[tabID] == userID
}

func UserRouteExists(userID string, route string, excludeTabID string) bool {
	for _, tab := range TabsForUser(userID) {
		if tab.ID != excludeTabID && tab.Route == route {
			return true
		}
	}
	return false
}

func ReorderUserTabs(userID string, items []models.ReorderTabItem) {
	for _, item := range items {
		for i := range Tabs {
			if Tabs[i].ID == item.TabID && UserTabs[userID][item.TabID] {
				Tabs[i].SortOrder = item.SortOrder
			}
		}
	}
}

func sortTabs(tabs []models.TabManifest) {
	sort.SliceStable(tabs, func(i, j int) bool {
		return tabs[i].SortOrder < tabs[j].SortOrder
	})
}

func nextSortOrder(userID string) int {
	maxSortOrder := 0
	for _, tab := range TabsForUser(userID) {
		if tab.SortOrder > maxSortOrder {
			maxSortOrder = tab.SortOrder
		}
	}
	return maxSortOrder + 10
}
