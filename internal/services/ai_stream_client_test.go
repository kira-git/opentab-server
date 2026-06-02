package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAIStreamClientParsesNestedSSEData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat/stream" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data:retry: 3000\n"))
		_, _ = w.Write([]byte("data:\n\n"))
		_, _ = w.Write([]byte("data:id: 1\n"))
		_, _ = w.Write([]byte("data:event: message\n"))
		_, _ = w.Write([]byte("data:data: {\"type\":\"content\",\"delta\":\"hello\"}\n\n"))
		_, _ = w.Write([]byte("data:id: 2\n"))
		_, _ = w.Write([]byte("data:data: {\"type\":\"done\",\"messageId\":\"msg-1\"}\n\n"))
	}))
	defer server.Close()

	client := NewAIStreamClient(server.URL)
	var events []AIChatEvent
	err := client.Stream(context.Background(), AIChatRequest{Message: "test"}, func(event AIChatEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("stream failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 parsed events, got %d", len(events))
	}
	if events[0].Data["type"] != "content" || events[0].Data["delta"] != "hello" {
		t.Fatalf("unexpected first event: %#v", events[0].Data)
	}
	if events[1].Data["type"] != "done" || events[1].Data["messageId"] != "msg-1" {
		t.Fatalf("unexpected second event: %#v", events[1].Data)
	}
}
