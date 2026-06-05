package routes

import (
	"net/http"
	"testing"
)

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
