package routes

import (
	"net/http"
	"testing"

	"opentab-server/internal/models"
)

func TestListTabsRequiresToken(t *testing.T) {
	router := setupTestRouter()

	recorder := performRequest(router, http.MethodGet, "/tabs", "", "")

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestListTabsForDemoUser(t *testing.T) {
	router := setupTestRouter()

	recorder := performRequest(router, http.MethodGet, "/tabs", "mock-access-token", "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	tabs := decodeJSON[[]models.TabManifest](t, recorder.Body)
	if len(tabs) != 3 {
		t.Fatalf("expected 3 demo tabs, got %d", len(tabs))
	}
	if tabs[0].ID != "approval" {
		t.Fatalf("expected first tab approval, got %q", tabs[0].ID)
	}
}

func TestListCatalogReturnsAllTabs(t *testing.T) {
	router := setupTestRouter()

	recorder := performRequest(router, http.MethodGet, "/tabs/catalog", "mock-access-token", "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	tabs := decodeJSON[[]models.TabManifest](t, recorder.Body)
	if len(tabs) < 8 {
		t.Fatalf("expected at least 8 catalog tabs, got %d", len(tabs))
	}
}

func TestGuestCannotEnableApprovalTab(t *testing.T) {
	router := setupTestRouter()

	recorder := performRequest(router, http.MethodPost, "/me/tabs", "mock-guest-token", `{"tabId":"approval"}`)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", recorder.Code, recorder.Body.String())
	}

	resp := decodeJSON[models.ErrorResponse](t, recorder.Body)
	if resp.Code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %q", resp.Code)
	}
}

func TestEnableAndDisableTab(t *testing.T) {
	router := setupTestRouter()

	enableRecorder := performRequest(router, http.MethodPost, "/me/tabs", "mock-access-token", `{"tabId":"docs"}`)
	if enableRecorder.Code != http.StatusOK {
		t.Fatalf("expected enable status 200, got %d: %s", enableRecorder.Code, enableRecorder.Body.String())
	}

	enableResp := decodeJSON[models.TabMutationResponse](t, enableRecorder.Body)
	if !enableResp.Success || enableResp.TabID != "docs" {
		t.Fatalf("unexpected enable response: %+v", enableResp)
	}

	disableRecorder := performRequest(router, http.MethodDelete, "/me/tabs/docs", "mock-access-token", "")
	if disableRecorder.Code != http.StatusOK {
		t.Fatalf("expected disable status 200, got %d: %s", disableRecorder.Code, disableRecorder.Body.String())
	}

	disableResp := decodeJSON[models.TabMutationResponse](t, disableRecorder.Body)
	if !disableResp.Success || disableResp.TabID != "docs" {
		t.Fatalf("unexpected disable response: %+v", disableResp)
	}
}

func TestValidateTabDetectsMissingFields(t *testing.T) {
	router := setupTestRouter()

	body := `{
		"containerVersion": 1,
		"permissions": [],
		"tab": {
			"displayName": "",
			"route": "",
			"entryType": "web"
		}
	}`
	recorder := performRequest(router, http.MethodPost, "/tabs/validate", "mock-access-token", body)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	resp := decodeJSON[models.ValidateTabResponse](t, recorder.Body)
	if resp.Valid {
		t.Fatalf("expected invalid tab")
	}
	if len(resp.Errors) < 3 {
		t.Fatalf("expected at least 3 validation errors, got %d", len(resp.Errors))
	}
}

func TestCustomTabCRUDAndOrder(t *testing.T) {
	router := setupTestRouter()

	createBody := `{
		"id": "custom-docs-test",
		"displayName": "我的文档",
		"description": "用户自定义网页 Tab",
		"icon": "docs",
		"route": "/custom-docs-test",
		"entryType": "web",
		"entryUri": "https://example.com",
		"minContainerVersion": 1
	}`
	createRecorder := performRequest(router, http.MethodPost, "/tabs", "mock-access-token", createBody)
	if createRecorder.Code != http.StatusOK {
		t.Fatalf("expected create status 200, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}

	updateRecorder := performRequest(router, http.MethodPut, "/tabs/custom-docs-test", "mock-access-token", `{"displayName":"我的文档新版","entryUri":"https://example.com/new-docs","sortOrder":100}`)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected update status 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}

	orderRecorder := performRequest(router, http.MethodPut, "/me/tabs/order", "mock-access-token", `{"items":[{"tabId":"approval","sortOrder":20},{"tabId":"custom-docs-test","sortOrder":10}]}`)
	if orderRecorder.Code != http.StatusOK {
		t.Fatalf("expected reorder status 200, got %d: %s", orderRecorder.Code, orderRecorder.Body.String())
	}

	deleteRecorder := performRequest(router, http.MethodDelete, "/tabs/custom-docs-test", "mock-access-token", "")
	if deleteRecorder.Code != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

func TestAdminPublishedCustomTabVisibleToTargetTeamOnly(t *testing.T) {
	router := setupTestRouter()

	createBody := `{
		"id": "custom-team-product-docs-test",
		"displayName": "产品部资料库",
		"description": "仅产品研发部可见的网页 Tab",
		"icon": "docs",
		"route": "/custom-team-product-docs-test",
		"entryType": "web",
		"entryUri": "https://example.com/product-docs",
		"minContainerVersion": 1,
		"visibility": {
			"scope": "custom",
			"teamIds": ["team-product"],
			"defaultEnabled": true
		}
	}`
	createRecorder := performRequest(router, http.MethodPost, "/tabs", "mock-admin-token", createBody)
	if createRecorder.Code != http.StatusOK {
		t.Fatalf("expected create status 200, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	createResp := decodeJSON[models.CustomTabResponse](t, createRecorder.Body)
	if createResp.Tab.Visibility == nil || createResp.Tab.Visibility.Scope != "custom" {
		t.Fatalf("expected custom visibility in response, got %+v", createResp.Tab.Visibility)
	}

	productTabs := performRequest(router, http.MethodGet, "/tabs", "mock-product-employee-token", "")
	if productTabs.Code != http.StatusOK {
		t.Fatalf("expected product tabs status 200, got %d: %s", productTabs.Code, productTabs.Body.String())
	}
	if !responseHasTab(t, productTabs, "custom-team-product-docs-test") {
		t.Fatalf("expected product employee to see published tab")
	}

	operationTabs := performRequest(router, http.MethodGet, "/tabs", "mock-operation-employee-token", "")
	if operationTabs.Code != http.StatusOK {
		t.Fatalf("expected operation tabs status 200, got %d: %s", operationTabs.Code, operationTabs.Body.String())
	}
	if responseHasTab(t, operationTabs, "custom-team-product-docs-test") {
		t.Fatalf("expected operation employee not to see product-only tab")
	}

	operationCatalog := performRequest(router, http.MethodGet, "/tabs/catalog", "mock-operation-employee-token", "")
	if operationCatalog.Code != http.StatusOK {
		t.Fatalf("expected operation catalog status 200, got %d: %s", operationCatalog.Code, operationCatalog.Body.String())
	}
	if responseHasTab(t, operationCatalog, "custom-team-product-docs-test") {
		t.Fatalf("expected operation catalog not to include product-only tab")
	}

	enableRecorder := performRequest(router, http.MethodPost, "/me/tabs", "mock-operation-employee-token", `{"tabId":"custom-team-product-docs-test"}`)
	if enableRecorder.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden enable status, got %d: %s", enableRecorder.Code, enableRecorder.Body.String())
	}
}

func TestNonAdminCannotPublishCompanyTab(t *testing.T) {
	router := setupTestRouter()

	createBody := `{
		"id": "custom-company-denied-test",
		"displayName": "公司门户测试",
		"description": "普通员工不能发布到全公司",
		"icon": "web",
		"route": "/custom-company-denied-test",
		"entryType": "web",
		"entryUri": "https://example.com/company-denied",
		"minContainerVersion": 1,
		"visibility": {
			"scope": "company",
			"defaultEnabled": true
		}
	}`
	recorder := performRequest(router, http.MethodPost, "/tabs", "mock-product-employee-token", createBody)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestDisableAdminPublishedTabLeavesTombstone(t *testing.T) {
	router := setupTestRouter()

	createBody := `{
		"id": "custom-company-tombstone-test",
		"displayName": "全员入口测试",
		"description": "用于验证停用后不会被默认发布重新带回",
		"icon": "web",
		"route": "/custom-company-tombstone-test",
		"entryType": "web",
		"entryUri": "https://example.com/company-tombstone",
		"minContainerVersion": 1,
		"visibility": {
			"scope": "company",
			"defaultEnabled": true
		}
	}`
	createRecorder := performRequest(router, http.MethodPost, "/tabs", "mock-admin-token", createBody)
	if createRecorder.Code != http.StatusOK {
		t.Fatalf("expected create status 200, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}

	before := performRequest(router, http.MethodGet, "/tabs", "mock-product-employee-token", "")
	if !responseHasTab(t, before, "custom-company-tombstone-test") {
		t.Fatalf("expected employee to see company tab before disabling")
	}

	disableRecorder := performRequest(router, http.MethodDelete, "/me/tabs/custom-company-tombstone-test", "mock-product-employee-token", "")
	if disableRecorder.Code != http.StatusOK {
		t.Fatalf("expected disable status 200, got %d: %s", disableRecorder.Code, disableRecorder.Body.String())
	}

	after := performRequest(router, http.MethodGet, "/tabs", "mock-product-employee-token", "")
	if responseHasTab(t, after, "custom-company-tombstone-test") {
		t.Fatalf("expected disabled default-published tab to stay hidden")
	}
}
