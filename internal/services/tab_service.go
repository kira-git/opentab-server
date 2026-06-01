package services

import (
	"errors"
	"net/http"
	"strings"

	"opentab-server/internal/models"
	"opentab-server/internal/repositories"
)

type TabService struct {
	tabs repositories.TabRepository
}

func NewTabService(tabs repositories.TabRepository) *TabService {
	return &TabService{tabs: tabs}
}

func (s *TabService) ListUserTabs(user *models.User) ([]models.TabManifest, *AppError) {
	tabs, err := s.tabs.ListByUser(user.ID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "获取 Tab 列表失败")
	}
	return tabs, nil
}

func (s *TabService) ListCatalog(user *models.User) ([]models.TabManifest, *AppError) {
	tabs, err := s.tabs.ListCatalog(user.ID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "获取 Tab 目录失败")
	}
	return tabs, nil
}

func (s *TabService) ListAll() ([]models.TabManifest, *AppError) {
	tabs, err := s.tabs.ListAll()
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "获取示例 Tab 失败")
	}
	return tabs, nil
}

func (s *TabService) GetTab(user *models.User, tabID string) (*models.TabManifest, *AppError) {
	tabs, err := s.tabs.ListByUser(user.ID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "获取 Tab 详情失败")
	}
	for _, tab := range tabs {
		if tab.ID == tabID {
			return &tab, nil
		}
	}
	return nil, NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "Tab 不存在或当前账号未启用")
}

func (s *TabService) EnableTab(user *models.User, tabID string) (*models.TabMutationResponse, *AppError) {
	if strings.TrimSpace(tabID) == "" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_REQUEST", "tabId 不可为空")
	}

	tab, err := s.tabs.FindByID(tabID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "Tab 不存在")
	}
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "启用 Tab 失败")
	}
	if !hasAllPermissions(user, tab.Permissions) {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "当前账号无权启用"+tab.DisplayName)
	}

	if err := s.tabs.Enable(user.ID, tab.ID); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "启用 Tab 失败")
	}
	return &models.TabMutationResponse{Success: true, TabID: tab.ID}, nil
}

func (s *TabService) CreateCustomTab(user *models.User, req models.CreateCustomTabRequest) (*models.CustomTabResponse, *AppError) {
	if strings.TrimSpace(req.ID) == "" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_REQUEST", "id 不可为空")
	}
	if strings.TrimSpace(req.DisplayName) == "" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_REQUEST", "displayName 不可为空")
	}
	if strings.TrimSpace(req.Route) == "" || !strings.HasPrefix(req.Route, "/") {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_TAB_CONFIG", "route 不可为空且必须以 / 开头")
	}
	if req.EntryType != "web" {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_TAB_CONFIG", "自定义 Tab 第一版仅支持 web 类型")
	}
	if !isHTTPURL(req.EntryURI) {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_TAB_CONFIG", "entryUri 必须是 http:// 或 https:// 地址")
	}
	if s.tabs.RouteExistsForUser(user.ID, req.Route, "") {
		return nil, NewAppError(http.StatusConflict, "RESOURCE_CONFLICT", "当前账号下 route 已存在")
	}

	tab := models.TabManifest{
		ID:                  req.ID,
		DisplayName:         req.DisplayName,
		Description:         req.Description,
		Icon:                req.Icon,
		Route:               req.Route,
		EntryType:           req.EntryType,
		EntryURI:            req.EntryURI,
		Version:             models.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		MinContainerVersion: req.MinContainerVersion,
		Permissions:         []string{},
		Enabled:             true,
	}
	if tab.Icon == "" {
		tab.Icon = "web"
	}
	if tab.MinContainerVersion == 0 {
		tab.MinContainerVersion = 1
	}

	created, err := s.tabs.CreateCustom(user.ID, tab)
	if errors.Is(err, repositories.ErrConflict) {
		return nil, NewAppError(http.StatusConflict, "RESOURCE_CONFLICT", "Tab ID 已存在")
	}
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "创建自定义 Tab 失败")
	}
	return &models.CustomTabResponse{Success: true, TabID: created.ID, Tab: *created}, nil
}

func (s *TabService) UpdateCustomTab(user *models.User, tabID string, req models.UpdateCustomTabRequest) (*models.CustomTabResponse, *AppError) {
	if strings.TrimSpace(req.EntryURI) != "" && !isHTTPURL(req.EntryURI) {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_TAB_CONFIG", "entryUri 必须是 http:// 或 https:// 地址")
	}

	tab, err := s.tabs.UpdateCustom(user.ID, tabID, req)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "Tab 不存在")
	}
	if errors.Is(err, repositories.ErrForbidden) {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "系统内置 Tab 不允许修改")
	}
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "修改自定义 Tab 失败")
	}
	return &models.CustomTabResponse{Success: true, TabID: tab.ID, Tab: *tab}, nil
}

func (s *TabService) DeleteCustomTab(user *models.User, tabID string) (*models.TabMutationResponse, *AppError) {
	err := s.tabs.DeleteCustom(user.ID, tabID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "Tab 不存在")
	}
	if errors.Is(err, repositories.ErrForbidden) {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "系统内置 Tab 不允许删除")
	}
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "删除自定义 Tab 失败")
	}
	return &models.TabMutationResponse{Success: true, TabID: tabID}, nil
}

func (s *TabService) ReorderTabs(user *models.User, req models.ReorderTabsRequest) (*models.SuccessResponse, *AppError) {
	if len(req.Items) == 0 {
		return nil, NewAppError(http.StatusBadRequest, "INVALID_REQUEST", "items 不可为空")
	}
	if err := s.tabs.Reorder(user.ID, req.Items); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "调整 Tab 顺序失败")
	}
	return &models.SuccessResponse{Success: true}, nil
}

func (s *TabService) DisableTab(user *models.User, tabID string) (*models.TabMutationResponse, *AppError) {
	_, err := s.tabs.FindByID(tabID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "Tab 不存在")
	}
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "停用 Tab 失败")
	}

	if err := s.tabs.Disable(user.ID, tabID); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "停用 Tab 失败")
	}
	return &models.TabMutationResponse{Success: true, TabID: tabID}, nil
}

func (s *TabService) ValidateTab(req models.ValidateTabRequest) models.ValidateTabResponse {
	errors := make([]models.ValidationIssue, 0)
	warnings := make([]models.ValidationIssue, 0)
	tab := req.Tab

	if strings.TrimSpace(tab.ID) == "" {
		errors = append(errors, models.ValidationIssue{Code: "MISSING_REQUIRED_FIELD", ProtocolCode: 1003, Message: "Tab 缺失必填字段：id", Field: "id"})
	}
	if strings.TrimSpace(tab.DisplayName) == "" {
		errors = append(errors, models.ValidationIssue{Code: "MISSING_REQUIRED_FIELD", ProtocolCode: 1003, Message: "Tab 缺失必填字段：displayName", Field: "displayName"})
	}
	if strings.TrimSpace(tab.Route) == "" {
		errors = append(errors, models.ValidationIssue{Code: "MISSING_REQUIRED_FIELD", ProtocolCode: 1003, Message: "Tab 缺失必填字段：route", Field: "route"})
	} else if !strings.HasPrefix(tab.Route, "/") {
		errors = append(errors, models.ValidationIssue{Code: "INVALID_TAB_CONFIG", Message: "route 必须以 / 开头", Field: "route"})
	}
	if tab.EntryType == "" {
		tab.EntryType = "native"
	}
	if !isSupportedEntryType(tab.EntryType) {
		errors = append(errors, models.ValidationIssue{Code: "INVALID_TAB_CONFIG", Message: "不支持的 entryType：" + tab.EntryType, Field: "entryType"})
	}
	if tab.EntryType == "web" || tab.EntryType == "external" {
		if strings.TrimSpace(tab.EntryURI) == "" {
			errors = append(errors, models.ValidationIssue{Code: "MISSING_REQUIRED_FIELD", ProtocolCode: 1003, Message: "web/external Tab 必须提供 entryUri", Field: "entryUri"})
		}
	}
	if req.ContainerVersion > 0 && tab.MinContainerVersion > req.ContainerVersion {
		warnings = append(warnings, models.ValidationIssue{Code: "CONTAINER_VERSION_TOO_LOW", ProtocolCode: 1002, Message: "当前容器版本低于 Tab 最低要求", Field: "minContainerVersion"})
	}
	for _, permission := range missingPermissions(req.Permissions, tab.Permissions) {
		warnings = append(warnings, models.ValidationIssue{Code: "MISSING_PERMISSION", ProtocolCode: 1006, Message: "当前账号缺少权限：" + permission, Field: "permissions"})
	}

	valid := len(errors) == 0
	openable := valid && len(warnings) == 0
	tab.Enabled = true
	return models.ValidateTabResponse{
		Valid:         valid,
		Openable:      openable,
		Errors:        errors,
		Warnings:      warnings,
		NormalizedTab: &tab,
	}
}

func (s *TabService) ReportAction(user *models.User, tabID string, actionID string) (*models.ActionResponse, *AppError) {
	tab, err := s.tabs.FindByID(tabID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, NewAppError(http.StatusNotFound, "RESOURCE_NOT_FOUND", "Tab 不存在")
	}
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "处理 Tab 动作失败")
	}
	if !hasAllPermissions(user, tab.Permissions) {
		return nil, NewAppError(http.StatusForbidden, "FORBIDDEN", "当前账号无权操作"+tab.DisplayName)
	}

	next := models.ActionNext{Type: "toast", Text: tab.DisplayName + "动作已触发：" + actionID}
	if actionID == "refresh" {
		next = models.ActionNext{Type: "refresh", Text: tab.DisplayName + "已刷新"}
	}
	return &models.ActionResponse{
		Success: true,
		Message: "动作已触发",
		Next:    next,
	}, nil
}

func (s *TabService) Count() int {
	return s.tabs.Count()
}

func isSupportedEntryType(entryType string) bool {
	switch entryType {
	case "native", "web", "hybrid", "external":
		return true
	default:
		return false
	}
}

func missingPermissions(current []string, required []string) []string {
	currentSet := map[string]bool{}
	for _, permission := range current {
		currentSet[permission] = true
	}

	result := make([]string, 0)
	for _, permission := range required {
		if !currentSet[permission] {
			result = append(result, permission)
		}
	}
	return result
}

func isHTTPURL(value string) bool {
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}
