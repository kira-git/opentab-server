package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

type aiRequest struct {
	Message        string `json:"message"`
	ConversationID string `json:"conversationId"`
}

func main() {
	port := flag.String("port", getenv("OPENTAB_PROBE_PORT", "18080"), "local OpenTab server port")
	smoothInterval := flag.String("interval", getenv("AI_STREAM_SMOOTH_INTERVAL_MS", "120"), "smooth interval in milliseconds")
	chunkSize := flag.String("chunk", getenv("AI_STREAM_SMOOTH_CHUNK_SIZE", "2"), "smooth chunk size in runes")
	scenario := flag.String("scenario", getenv("AI_STREAM_PROBE_SCENARIO", "all"), "burst, slow, stall, error, mixed, or all")
	flag.Parse()

	fakeAI := httptest.NewServer(http.HandlerFunc(fakeAIHandler))
	defer fakeAI.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := exec.CommandContext(ctx, "go", "run", "./cmd/server")
	server.Env = append(os.Environ(),
		"APP_MODE=mock",
		"DATABASE_URL=",
		"HOST=127.0.0.1",
		"PORT="+*port,
		"AI_SERVICE_BASE_URL="+fakeAI.URL,
		"AI_CONCURRENT_LIMIT=3",
		"AI_USER_CONCURRENT_LIMIT=1",
		"AI_STREAM_SMOOTH_INTERVAL_MS="+*smoothInterval,
		"AI_STREAM_SMOOTH_CHUNK_SIZE="+*chunkSize,
	)
	var serverLog bytes.Buffer
	server.Stdout = &serverLog
	server.Stderr = &serverLog

	if err := server.Start(); err != nil {
		log.Fatalf("start OpenTab server: %v", err)
	}
	defer func() {
		cancel()
		_ = server.Process.Kill()
		_, _ = server.Process.Wait()
	}()

	baseURL := "http://127.0.0.1:" + *port
	if err := waitHealth(baseURL, 10*time.Second); err != nil {
		fmt.Println(serverLog.String())
		log.Fatalf("OpenTab server did not become healthy: %v", err)
	}

	fmt.Printf("Fake AI service: %s\n", fakeAI.URL)
	fmt.Printf("OpenTab server:   %s\n", baseURL)
	fmt.Printf("Smooth config:    interval=%sms chunk=%s runes\n\n", *smoothInterval, *chunkSize)

	scenarios := []string{*scenario}
	if *scenario == "all" {
		scenarios = []string{"burst", "slow", "stall", "mixed", "error"}
	}

	for _, item := range scenarios {
		fmt.Printf("========== scenario: %s ==========\n", item)
		if err := probeScenario(baseURL, item); err != nil {
			fmt.Printf("probe error: %v\n", err)
		}
		fmt.Println()
	}
}

func fakeAIHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/chat/stream" {
		http.NotFound(w, r)
		return
	}

	var req aiRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	scenario := strings.TrimSpace(req.Message)
	if scenario == "" {
		scenario = "burst"
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, _ := w.(http.Flusher)

	send := func(data map[string]any) bool {
		payload, _ := json.Marshal(data)
		if _, err := fmt.Fprintf(w, "event: message\ndata: %s\n\n", payload); err != nil {
			return false
		}
		if flusher != nil {
			flusher.Flush()
		}
		return true
	}

	switch scenario {
	case "burst":
		_ = send(map[string]any{
			"type":  "content",
			"delta": "这是一段由假 AI 一次性快速输出的大段中文文本，用来观察 OpenTab 服务端是否会通过双缓冲机制把它平滑拆成小块返回给客户端。",
		})
		_ = send(map[string]any{"type": "done", "messageId": "fake-burst"})
	case "slow":
		for _, text := range []string{"第一段慢速输出。", "第二段稍后到达。", "第三段继续输出。"} {
			if !send(map[string]any{"type": "content", "delta": text}) {
				return
			}
			time.Sleep(700 * time.Millisecond)
		}
		_ = send(map[string]any{"type": "done", "messageId": "fake-slow"})
	case "stall":
		time.Sleep(3600 * time.Millisecond)
		_ = send(map[string]any{"type": "content", "delta": "AI 卡顿几秒后才输出。"})
		_ = send(map[string]any{"type": "done", "messageId": "fake-stall"})
	case "mixed":
		_ = send(map[string]any{"type": "tool", "tool": "search"})
		time.Sleep(300 * time.Millisecond)
		_ = send(map[string]any{"type": "content", "delta": "工具事件会立即到达，正文 delta 会被平滑输出。"})
		_ = send(map[string]any{"type": "done", "messageId": "fake-mixed"})
	case "error":
		_ = send(map[string]any{"type": "error", "code": "FAKE_AI_ERROR", "delta": "假 AI 主动返回错误"})
	default:
		_ = send(map[string]any{"type": "content", "delta": "未知场景，默认返回一段文本。"})
		_ = send(map[string]any{"type": "done", "messageId": "fake-default"})
	}
}

func probeScenario(baseURL string, scenario string) error {
	endpoint := baseURL + "/oncall/stream?message=" + url.QueryEscape(scenario)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer mock-access-token")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	startedAt := time.Now()
	lastAt := startedAt
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	eventName := "message"
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, ":") {
			printTiming(startedAt, lastAt, "comment", strings.TrimSpace(line))
			lastAt = time.Now()
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		printTiming(startedAt, lastAt, eventName, data)
		lastAt = time.Now()
		if eventName == "done" || eventName == "error" {
			break
		}
	}
	return scanner.Err()
}

func printTiming(startedAt time.Time, previousAt time.Time, eventName string, data string) {
	now := time.Now()
	fmt.Printf(
		"+%7s  Δ%7s  event=%-8s data=%s\n",
		now.Sub(startedAt).Truncate(time.Millisecond),
		now.Sub(previousAt).Truncate(time.Millisecond),
		eventName,
		data,
	)
}

func waitHealth(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timeout")
}

func getenv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
