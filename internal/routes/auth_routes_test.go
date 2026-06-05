package routes

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"opentab-server/internal/models"
	"opentab-server/internal/repositories"
)

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

func TestAuditLogRecordsLogin(t *testing.T) {
	router := setupTestRouter()

	recorder := performRequest(router, http.MethodPost, "/auth/login", "", `{"account":"opentab-demo","password":"demo123"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	logs := repositories.MemoryAuditLogs()
	if len(logs) == 0 {
		t.Fatalf("expected audit log")
	}
	last := logs[len(logs)-1]
	if last.Action != "auth.login" {
		t.Fatalf("expected auth.login audit action, got %q", last.Action)
	}
	if last.UserID != "user-demo" || last.Account != "opentab-demo" {
		t.Fatalf("unexpected audit user: %+v", last)
	}
	if last.RequestID == "" {
		t.Fatalf("expected audit request id")
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
	if len(tabs) != 5 {
		t.Fatalf("expected 5 default tabs, got %d", len(tabs))
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
	errResp := decodeJSON[models.ErrorResponse](t, meRecorder.Body)
	if errResp.Code != "TOKEN_REVOKED" {
		t.Fatalf("expected TOKEN_REVOKED, got %q", errResp.Code)
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
