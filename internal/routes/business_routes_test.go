package routes

import (
	"net/http"
	"testing"

	"opentab-server/internal/models"
)

func TestBusinessAndDebugExpansionEndpoints(t *testing.T) {
	router := setupTestRouter()

	for _, item := range []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/business/approval/items?scope=mine"},
		{method: http.MethodGet, path: "/business/approval/items/apv-product-001"},
		{method: http.MethodPost, path: "/business/approval/items", body: `{"type":"leave","title":"测试请假","reason":"测试","form":{"days":1}}`},
		{method: http.MethodGet, path: "/business/calendar/events?scope=visible"},
		{method: http.MethodGet, path: "/business/calendar/events/evt-product-001"},
		{method: http.MethodPost, path: "/business/calendar/events", body: `{"title":"接口联调","description":"联调 TabManifest 和 AI OnCall","startTime":"2026-05-31T16:00:00+08:00","endTime":"2026-05-31T17:00:00+08:00","location":"线上会议","visibility":"team"}`},
		{method: http.MethodGet, path: "/business/announcements?scope=visible"},
		{method: http.MethodGet, path: "/debug/permissions"},
		{method: http.MethodGet, path: "/debug/sample-tabs"},
	} {
		recorder := performRequest(router, item.method, item.path, "mock-product-manager-token", item.body)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s %s expected status 200, got %d: %s", item.method, item.path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestBusinessDataIsScopedByTeam(t *testing.T) {
	router := setupTestRouter()

	approveRecorder := performRequest(router, http.MethodPost, "/business/approval/items/apv-product-001/approve", "mock-product-manager-token", `{"comment":"产品主管通过"}`)
	if approveRecorder.Code != http.StatusOK {
		t.Fatalf("product manager approve expected status 200, got %d: %s", approveRecorder.Code, approveRecorder.Body.String())
	}

	operationItemRecorder := performRequest(router, http.MethodGet, "/business/approval/items/apv-product-001", "mock-operation-manager-token", "")
	if operationItemRecorder.Code != http.StatusNotFound {
		t.Fatalf("operation manager should not read product approval, got %d: %s", operationItemRecorder.Code, operationItemRecorder.Body.String())
	}

	createCalendarRecorder := performRequest(router, http.MethodPost, "/business/calendar/events", "mock-product-manager-token", `{"title":"产品团队临时会议","startTime":"2026-05-31T18:00:00+08:00","endTime":"2026-05-31T19:00:00+08:00","location":"线上","visibility":"team"}`)
	if createCalendarRecorder.Code != http.StatusOK {
		t.Fatalf("product manager create calendar expected status 200, got %d: %s", createCalendarRecorder.Code, createCalendarRecorder.Body.String())
	}
	createCalendarResp := decodeJSON[models.CreateCalendarEventResponse](t, createCalendarRecorder.Body)

	operationEventRecorder := performRequest(router, http.MethodGet, "/business/calendar/events/"+createCalendarResp.EventID, "mock-operation-employee-token", "")
	if operationEventRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected operation employee cannot read product calendar event, got %d: %s", operationEventRecorder.Code, operationEventRecorder.Body.String())
	}
}

func TestApprovalCancelEndpoint(t *testing.T) {
	router := setupTestRouter()

	create := performRequest(router, http.MethodPost, "/business/approval/items", "mock-product-employee-token", `{"type":"leave","title":"我要撤回的申请","reason":"测试撤回","form":{"days":1}}`)
	if create.Code != http.StatusOK {
		t.Fatalf("create approval expected status 200, got %d: %s", create.Code, create.Body.String())
	}
	item := decodeJSON[models.ApprovalItem](t, create.Body)

	cancel := performRequest(router, http.MethodPost, "/business/approval/items/"+item.ID+"/cancel", "mock-product-employee-token", "")
	if cancel.Code != http.StatusOK {
		t.Fatalf("cancel approval expected status 200, got %d: %s", cancel.Code, cancel.Body.String())
	}
	resp := decodeJSON[models.ApprovalActionResponse](t, cancel.Body)
	if resp.Status != "cancelled" {
		t.Fatalf("expected cancelled status, got %q", resp.Status)
	}

	otherCancel := performRequest(router, http.MethodPost, "/business/approval/items/apv-product-001/cancel", "mock-operation-employee-token", "")
	if otherCancel.Code != http.StatusForbidden && otherCancel.Code != http.StatusNotFound && otherCancel.Code != http.StatusBadRequest {
		t.Fatalf("other team cancel should fail, got %d: %s", otherCancel.Code, otherCancel.Body.String())
	}
}
