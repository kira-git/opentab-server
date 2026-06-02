package services

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestProbeRealAIService(t *testing.T) {
	baseURL := os.Getenv("PROBE_AI_SERVICE_BASE_URL")
	if baseURL == "" {
		t.Skip("PROBE_AI_SERVICE_BASE_URL is not set")
	}

	client := NewAIStreamClient(baseURL)
	err := client.Stream(context.Background(), AIChatRequest{
		Message:        "如何注册Tab",
		ConversationID: "probe",
	}, func(event AIChatEvent) error {
		fmt.Printf("event=%s data=%v\n", event.Event, event.Data)
		return nil
	})
	if err != nil {
		t.Fatalf("stream failed: %v", err)
	}
}
