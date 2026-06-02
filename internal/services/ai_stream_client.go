package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type AIChatRequest struct {
	Message        string `json:"message"`
	ConversationID string `json:"conversationId,omitempty"`
}

type AIChatEvent struct {
	Event string
	Data  map[string]any
}

type AIStreamClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAIStreamClient(baseURL string) *AIStreamClient {
	return &AIStreamClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (c *AIStreamClient) Available() bool {
	return c != nil && c.baseURL != ""
}

func (c *AIStreamClient) Stream(ctx context.Context, req AIChatRequest, emit func(AIChatEvent) error) error {
	if !c.Available() {
		return fmt.Errorf("AI service base URL is empty")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat/stream", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json; charset=utf-8")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("AI service returned %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	eventName := "message"
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		rawData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if rawData == "" || rawData == "[DONE]" {
			continue
		}

		if strings.HasPrefix(rawData, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(rawData, "event:"))
			continue
		}
		if strings.HasPrefix(rawData, "id:") || strings.HasPrefix(rawData, "retry:") {
			continue
		}
		if strings.HasPrefix(rawData, "data:") {
			rawData = strings.TrimSpace(strings.TrimPrefix(rawData, "data:"))
		}
		if rawData == "" || rawData == "[DONE]" {
			continue
		}

		var data map[string]any
		if err := json.Unmarshal([]byte(rawData), &data); err != nil {
			continue
		}
		if err := emit(AIChatEvent{Event: eventName, Data: data}); err != nil {
			return err
		}
	}

	return scanner.Err()
}
