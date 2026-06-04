package services

import (
	"context"
	"testing"
	"time"
)

func TestStreamSmootherSplitsRunes(t *testing.T) {
	smoother := NewStreamSmoother(time.Millisecond, 2)
	raw := make(chan RawChunk, 2)
	out := make(chan OnCallEvent, 8)

	go smoother.Run(context.Background(), raw, out)
	raw <- RawChunk{Text: "中文测试"}
	raw <- RawChunk{Done: true}
	close(raw)

	var events []OnCallEvent
	for event := range out {
		events = append(events, event)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d: %+v", len(events), events)
	}
	if events[0].Event != "delta" || readJSONText(events[0].Data, "text") != "中文" {
		t.Fatalf("unexpected first event: %+v", events[0])
	}
	if events[1].Event != "delta" || readJSONText(events[1].Data, "text") != "测试" {
		t.Fatalf("unexpected second event: %+v", events[1])
	}
	if events[2].Event != "done" {
		t.Fatalf("expected done event, got %+v", events[2])
	}
}

func TestStreamSmootherDoesNotEmitDoneOnRawCloseWithoutDone(t *testing.T) {
	smoother := NewStreamSmoother(time.Millisecond, 2)
	raw := make(chan RawChunk)
	out := make(chan OnCallEvent, 8)

	go smoother.Run(context.Background(), raw, out)
	close(raw)

	for event := range out {
		if event.Event == "done" {
			t.Fatalf("raw close without done should not emit done")
		}
	}
}
