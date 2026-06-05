package routes

import (
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

func responseHasTab(t *testing.T, recorder *httptest.ResponseRecorder, tabID string) bool {
	t.Helper()
	tabs := decodeJSON[[]models.TabManifest](t, recorder.Body)
	for _, tab := range tabs {
		if tab.ID == tabID {
			return true
		}
	}
	return false
}
