package routes

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"opentab-server/internal/models"
	"opentab-server/internal/repositories"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewHandler()
	handler.sseDelay = 0
	registerWithHandler(router, handler)
	return router
}

func setupStatusTestRouter(status RuntimeStatus) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewHandlerWithStatus(repositories.NewMemoryRepositorySet(), status)
	handler.sseDelay = 0
	registerWithHandler(router, handler)
	return router
}

func performRequest(router http.Handler, method string, path string, token string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func decodeJSON[T any](t *testing.T, reader io.Reader) T {
	t.Helper()

	var result T
	if err := json.NewDecoder(reader).Decode(&result); err != nil {
		t.Fatalf("decode json response: %v", err)
	}
	return result
}

func TestLoginSuccess(t *testing.T) {
	router := setupTestRouter()

	recorder := performRequest(router, http.MethodPost, "/auth/login", "", `{"account":"opentab-demo","password":"demo123"}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	resp := decodeJSON[models.LoginResponse](t, recorder.Body)
	if resp.Token == "" {
		t.Fatalf("expected token")
	}
	if resp.Token == "mock-access-token" {
		t.Fatalf("expected login to issue a fresh token")
	}
	if resp.UserID != "user-demo" {
		t.Fatalf("expected user-demo, got %q", resp.UserID)
	}
}

func TestLoginFailed(t *testing.T) {
	router := setupTestRouter()

	recorder := performRequest(router, http.MethodPost, "/auth/login", "", `{"account":"opentab-demo","password":"wrong"}`)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}

	resp := decodeJSON[models.ErrorResponse](t, recorder.Body)
	if resp.Code != "INVALID_CREDENTIALS" {
		t.Fatalf("expected INVALID_CREDENTIALS, got %q", resp.Code)
	}
}

func TestRegisterSuccess(t *testing.T) {
	router := setupTestRouter()

	recorder := performRequest(router, http.MethodPost, "/auth/register", "", `{"account":"new-user-register-test","password":"new123456","displayName":"新注册用户"}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	resp := decodeJSON[models.LoginResponse](t, recorder.Body)
	if resp.Token == "" {
		t.Fatalf("expected token")
	}
	if resp.UserID == "" {
		t.Fatalf("expected user id")
	}
	if resp.DisplayName != "新注册用户" {
		t.Fatalf("expected display name, got %q", resp.DisplayName)
	}

	tabsRecorder := performRequest(router, http.MethodGet, "/tabs", resp.Token, "")
	if tabsRecorder.Code != http.StatusOK {
		t.Fatalf("expected registered user tabs status 200, got %d: %s", tabsRecorder.Code, tabsRecorder.Body.String())
	}
	tabs := decodeJSON[[]models.TabManifest](t, tabsRecorder.Body)
	if len(tabs) != 2 {
		t.Fatalf("expected 2 default tabs, got %d", len(tabs))
	}
}

func TestRegisterDuplicateAccount(t *testing.T) {
	router := setupTestRouter()

	recorder := performRequest(router, http.MethodPost, "/auth/register", "", `{"account":"opentab-demo","password":"new123456","displayName":"重复账号"}`)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", recorder.Code, recorder.Body.String())
	}

	resp := decodeJSON[models.ErrorResponse](t, recorder.Body)
	if resp.Code != "ACCOUNT_EXISTS" {
		t.Fatalf("expected ACCOUNT_EXISTS, got %q", resp.Code)
	}
}

func TestLogoutRequiresToken(t *testing.T) {
	router := setupTestRouter()

	recorder := performRequest(router, http.MethodPost, "/auth/logout", "", "")

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestLogoutSuccess(t *testing.T) {
	router := setupTestRouter()

	loginRecorder := performRequest(router, http.MethodPost, "/auth/login", "", `{"account":"opentab-demo","password":"demo123"}`)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d: %s", loginRecorder.Code, loginRecorder.Body.String())
	}
	login := decodeJSON[models.LoginResponse](t, loginRecorder.Body)

	recorder := performRequest(router, http.MethodPost, "/auth/logout", login.Token, "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	resp := decodeJSON[models.SuccessResponse](t, recorder.Body)
	if !resp.Success {
		t.Fatalf("expected logout success")
	}

	meRecorder := performRequest(router, http.MethodGet, "/me", login.Token, "")
	if meRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked token status 401, got %d: %s", meRecorder.Code, meRecorder.Body.String())
	}
}

func TestListTabsRequiresToken(t *testing.T) {
	router := setupTestRouter()

	recorder := performRequest(router, http.MethodGet, "/tabs", "", "")

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestRuntimeStatusReflectsPostgresMode(t *testing.T) {
	router := setupStatusTestRouter(RuntimeStatus{
		AppMode:         "postgres",
		DatabaseEnabled: true,
		DatabaseType:    "postgres",
	})

	healthRecorder := performRequest(router, http.MethodGet, "/health", "", "")
	if healthRecorder.Code != http.StatusOK {
		t.Fatalf("expected health status 200, got %d: %s", healthRecorder.Code, healthRecorder.Body.String())
	}
	health := decodeJSON[map[string]any](t, healthRecorder.Body)
	if health["mode"] != "postgres" {
		t.Fatalf("expected health mode postgres, got %v", health["mode"])
	}

	statusRecorder := performRequest(router, http.MethodGet, "/debug/status", "mock-access-token", "")
	if statusRecorder.Code != http.StatusOK {
		t.Fatalf("expected debug status 200, got %d: %s", statusRecorder.Code, statusRecorder.Body.String())
	}
	status := decodeJSON[map[string]any](t, statusRecorder.Body)
	if status["mockMode"] != false {
		t.Fatalf("expected mockMode false, got %v", status["mockMode"])
	}
	database, ok := status["database"].(map[string]any)
	if !ok {
		t.Fatalf("expected database object, got %T", status["database"])
	}
	if database["enabled"] != true || database["type"] != "postgres" {
		t.Fatalf("unexpected database status: %+v", database)
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
	if len(tabs) != 5 {
		t.Fatalf("expected 5 catalog tabs, got %d", len(tabs))
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

func TestOnCallStreamReturnsSSE(t *testing.T) {
	router := setupTestRouter()

	recorder := performRequest(router, http.MethodGet, "/oncall/stream?message=tabdefinition配置", "mock-access-token", "")

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("expected event-stream content type, got %q", contentType)
	}

	body := recorder.Body.String()
	for _, want := range []string{"event: delta", "event: tool", "event: done"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected SSE body to contain %q, got %s", want, body)
		}
	}
}

func TestAIChatStreamReturnsDocumentProtocolSSE(t *testing.T) {
	router := setupTestRouter()

	recorder := performRequest(router, http.MethodPost, "/api/chat/stream", "", `{"message":"如何注册 Tab？","conversationId":"test-001"}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("expected event-stream content type, got %q", contentType)
	}

	body := recorder.Body.String()
	for _, want := range []string{"event: message", `"type":"tool"`, `"type":"content"`, `"type":"done"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected SSE body to contain %q, got %s", want, body)
		}
	}
}

func TestInvalidJSONReturnsBadRequest(t *testing.T) {
	router := setupTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", recorder.Code, recorder.Body.String())
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

func TestOnCallSessionFlow(t *testing.T) {
	router := setupTestRouter()

	createRecorder := performRequest(router, http.MethodPost, "/oncall/sessions", "mock-access-token", `{"title":"Tab 接入咨询"}`)
	if createRecorder.Code != http.StatusOK {
		t.Fatalf("expected create session status 200, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}

	session := decodeJSON[models.OnCallSession](t, createRecorder.Body)
	messageRecorder := performRequest(router, http.MethodPost, "/oncall/sessions/"+session.SessionID+"/messages", "mock-access-token", `{"content":"如何接入审批 Tab？","contentType":"text"}`)
	if messageRecorder.Code != http.StatusOK {
		t.Fatalf("expected add message status 200, got %d: %s", messageRecorder.Code, messageRecorder.Body.String())
	}

	message := decodeJSON[models.OnCallMessage](t, messageRecorder.Body)
	streamRecorder := performRequest(router, http.MethodGet, "/oncall/sessions/"+session.SessionID+"/stream?messageId="+message.MessageID, "mock-access-token", "")
	if streamRecorder.Code != http.StatusOK {
		t.Fatalf("expected session stream status 200, got %d: %s", streamRecorder.Code, streamRecorder.Body.String())
	}

	listRecorder := performRequest(router, http.MethodGet, "/oncall/sessions/"+session.SessionID+"/messages", "mock-access-token", "")
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected list messages status 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}

	deleteRecorder := performRequest(router, http.MethodDelete, "/oncall/sessions/"+session.SessionID, "mock-access-token", "")
	if deleteRecorder.Code != http.StatusOK {
		t.Fatalf("expected delete session status 200, got %d: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

func TestBusinessAndDebugExpansionEndpoints(t *testing.T) {
	router := setupTestRouter()

	for _, item := range []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/business/approval/items?status=all"},
		{method: http.MethodGet, path: "/business/approval/items/apv-001"},
		{method: http.MethodPost, path: "/business/approval/items/apv-001/approve", body: `{"comment":"同意"}`},
		{method: http.MethodPost, path: "/business/approval/items/apv-001/reject", body: `{"comment":"资料不完整"}`},
		{method: http.MethodGet, path: "/business/calendar/events?date=2026-05-31"},
		{method: http.MethodGet, path: "/business/calendar/events/evt-001"},
		{method: http.MethodPost, path: "/business/calendar/events", body: `{"title":"接口联调","description":"联调 TabManifest 和 AI OnCall","startTime":"2026-05-31T16:00:00+08:00","endTime":"2026-05-31T17:00:00+08:00","location":"线上会议"}`},
		{method: http.MethodGet, path: "/debug/permissions"},
		{method: http.MethodGet, path: "/debug/sample-tabs"},
	} {
		recorder := performRequest(router, item.method, item.path, "mock-access-token", item.body)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s %s expected status 200, got %d: %s", item.method, item.path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestBusinessDataIsScopedByUser(t *testing.T) {
	router := setupTestRouter()

	userARecorder := performRequest(router, http.MethodPost, "/auth/register", "", `{"account":"scope-user-a","password":"scope123","displayName":"隔离用户 A"}`)
	if userARecorder.Code != http.StatusOK {
		t.Fatalf("register user a expected status 200, got %d: %s", userARecorder.Code, userARecorder.Body.String())
	}
	userA := decodeJSON[models.LoginResponse](t, userARecorder.Body)

	userBRecorder := performRequest(router, http.MethodPost, "/auth/register", "", `{"account":"scope-user-b","password":"scope123","displayName":"隔离用户 B"}`)
	if userBRecorder.Code != http.StatusOK {
		t.Fatalf("register user b expected status 200, got %d: %s", userBRecorder.Code, userBRecorder.Body.String())
	}
	userB := decodeJSON[models.LoginResponse](t, userBRecorder.Body)

	approveRecorder := performRequest(router, http.MethodPost, "/business/approval/items/apv-001/approve", userA.Token, `{"comment":"A 通过"}`)
	if approveRecorder.Code != http.StatusOK {
		t.Fatalf("user a approve expected status 200, got %d: %s", approveRecorder.Code, approveRecorder.Body.String())
	}

	userBItemRecorder := performRequest(router, http.MethodGet, "/business/approval/items/apv-001", userB.Token, "")
	if userBItemRecorder.Code != http.StatusOK {
		t.Fatalf("user b approval detail expected status 200, got %d: %s", userBItemRecorder.Code, userBItemRecorder.Body.String())
	}
	userBItem := decodeJSON[models.ApprovalItem](t, userBItemRecorder.Body)
	if userBItem.Status != "pending" {
		t.Fatalf("expected user b approval item to stay pending, got %q", userBItem.Status)
	}

	createCalendarRecorder := performRequest(router, http.MethodPost, "/business/calendar/events", userA.Token, `{"title":"A 的私有日程","startTime":"2026-05-31T18:00:00+08:00","endTime":"2026-05-31T19:00:00+08:00","location":"线上"}`)
	if createCalendarRecorder.Code != http.StatusOK {
		t.Fatalf("user a create calendar expected status 200, got %d: %s", createCalendarRecorder.Code, createCalendarRecorder.Body.String())
	}
	createCalendarResp := decodeJSON[models.CreateCalendarEventResponse](t, createCalendarRecorder.Body)

	userBEventRecorder := performRequest(router, http.MethodGet, "/business/calendar/events/"+createCalendarResp.EventID, userB.Token, "")
	if userBEventRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected user b cannot read user a calendar event, got %d: %s", userBEventRecorder.Code, userBEventRecorder.Body.String())
	}
}
