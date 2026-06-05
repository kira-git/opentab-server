package routes

import (
	"net/http"
	"strings"
	"testing"

	"opentab-server/internal/models"
)

func TestAIChatStreamRequiresAuthInPostgresMode(t *testing.T) {
	router := setupStatusTestRouter(RuntimeStatus{
		AppMode:         "postgres",
		DatabaseEnabled: true,
		DatabaseType:    "postgres",
	})

	recorder := performRequest(router, http.MethodPost, "/api/chat/stream", "", `{"message":"hello"}`)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", recorder.Code, recorder.Body.String())
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
	messages := decodeJSON[[]models.OnCallMessage](t, listRecorder.Body)
	if len(messages) != 2 {
		t.Fatalf("expected user message and one assistant reply, got %d: %+v", len(messages), messages)
	}

	repeatedStreamRecorder := performRequest(router, http.MethodGet, "/oncall/sessions/"+session.SessionID+"/stream?messageId="+message.MessageID, "mock-access-token", "")
	if repeatedStreamRecorder.Code != http.StatusOK {
		t.Fatalf("expected repeated session stream status 200, got %d: %s", repeatedStreamRecorder.Code, repeatedStreamRecorder.Body.String())
	}
	repeatedListRecorder := performRequest(router, http.MethodGet, "/oncall/sessions/"+session.SessionID+"/messages", "mock-access-token", "")
	if repeatedListRecorder.Code != http.StatusOK {
		t.Fatalf("expected repeated list messages status 200, got %d: %s", repeatedListRecorder.Code, repeatedListRecorder.Body.String())
	}
	repeatedMessages := decodeJSON[[]models.OnCallMessage](t, repeatedListRecorder.Body)
	if len(repeatedMessages) != 2 {
		t.Fatalf("expected repeated stream not to create duplicate assistant reply, got %d: %+v", len(repeatedMessages), repeatedMessages)
	}

	cancelRecorder := performRequest(router, http.MethodPost, "/oncall/sessions/"+session.SessionID+"/cancel", "mock-access-token", "")
	if cancelRecorder.Code != http.StatusOK {
		t.Fatalf("expected cancel generation status 200, got %d: %s", cancelRecorder.Code, cancelRecorder.Body.String())
	}
	cancel := decodeJSON[models.CancelOnCallGenerationResponse](t, cancelRecorder.Body)
	if !cancel.Success || cancel.SessionID != session.SessionID {
		t.Fatalf("unexpected cancel response: %+v", cancel)
	}

	deleteRecorder := performRequest(router, http.MethodDelete, "/oncall/sessions/"+session.SessionID, "mock-access-token", "")
	if deleteRecorder.Code != http.StatusOK {
		t.Fatalf("expected delete session status 200, got %d: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}
