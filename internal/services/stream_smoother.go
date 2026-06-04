package services

import (
	"context"
	"time"
)

type RawChunk struct {
	Text string
	Done bool
	Err  error
}

type StreamSmoother struct {
	TickInterval    time.Duration
	ChunkSize       int
	MaxPendingRunes int
}

func NewStreamSmoother(interval time.Duration, chunkSize int) *StreamSmoother {
	if interval <= 0 {
		interval = 50 * time.Millisecond
	}
	if chunkSize <= 0 {
		chunkSize = 2
	}
	return &StreamSmoother{
		TickInterval:    interval,
		ChunkSize:       chunkSize,
		MaxPendingRunes: 4096,
	}
}

func (s *StreamSmoother) Run(ctx context.Context, raw <-chan RawChunk, out chan<- OnCallEvent) {
	defer close(out)

	ticker := time.NewTicker(s.TickInterval)
	defer ticker.Stop()

	var pending []rune
	aiDone := false
	rawClosed := false
	bufferingNotified := false
	lastOutputAt := time.Now()
	lastRawAt := time.Now()

	for {
		select {
		case chunk, ok := <-raw:
			if !ok {
				rawClosed = true
				raw = nil
				continue
			}
			lastRawAt = time.Now()
			if chunk.Err != nil {
				out <- OnCallEvent{
					Event: "error",
					Data:  `{"code":"AI_SERVICE_ERROR","message":"AI 服务调用失败"}`,
				}
				return
			}
			if chunk.Done {
				aiDone = true
				continue
			}
			if chunk.Text != "" {
				pending = append(pending, []rune(chunk.Text)...)
				if len(pending) > s.MaxPendingRunes && !bufferingNotified {
					bufferingNotified = true
					out <- OnCallEvent{
						Event: "status",
						Data:  `{"stage":"buffering","message":"AI 输出较快，正在平滑输出"}`,
					}
				}
			}
		case <-ticker.C:
			if len(pending) > 0 {
				n := s.ChunkSize
				if len(pending) < n {
					n = len(pending)
				}
				part := string(pending[:n])
				pending = pending[n:]
				lastOutputAt = time.Now()
				out <- OnCallEvent{
					Event: "delta",
					Data:  `{"text":"` + jsonEscape(part) + `"}`,
				}
				continue
			}
			if aiDone {
				out <- OnCallEvent{Event: "done", Data: `{}`}
				return
			}
			if rawClosed {
				return
			}
			if time.Since(lastRawAt) > 3*time.Second && time.Since(lastOutputAt) > 3*time.Second {
				out <- OnCallEvent{
					Event: "status",
					Data:  `{"stage":"waiting_ai","message":"AI 正在继续分析"}`,
				}
				lastOutputAt = time.Now()
			}
		case <-ctx.Done():
			return
		}
	}
}
