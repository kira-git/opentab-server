package services

import (
	"context"
	"errors"
	"sync"
)

var ErrAITooManyRequests = errors.New("too many ai requests")
var ErrAIUserBusy = errors.New("user already has active ai stream")

type AIConcurrencyLimiter struct {
	sem chan struct{}
}

func NewAIConcurrencyLimiter(limit int) *AIConcurrencyLimiter {
	if limit <= 0 {
		limit = 3
	}
	return &AIConcurrencyLimiter{sem: make(chan struct{}, limit)}
}

func (l *AIConcurrencyLimiter) TryAcquire(ctx context.Context) error {
	select {
	case l.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrAITooManyRequests
	}
}

func (l *AIConcurrencyLimiter) Release() {
	select {
	case <-l.sem:
	default:
	}
}

type UserAIStreamLimiter struct {
	mu     sync.Mutex
	active map[string]map[string]string
	limit  int
}

func NewUserAIStreamLimiter(limit int) *UserAIStreamLimiter {
	if limit <= 0 {
		limit = 1
	}
	return &UserAIStreamLimiter{
		active: map[string]map[string]string{},
		limit:  limit,
	}
}

func (l *UserAIStreamLimiter) TryAcquire(userID string) error {
	return l.TryAcquireStream(userID, "", "")
}

func (l *UserAIStreamLimiter) Release(userID string) {
	l.ReleaseStream(userID, "", "")
}

func (l *UserAIStreamLimiter) TryAcquireStream(userID string, streamKey string, streamID string) error {
	if userID == "" {
		userID = "anonymous"
	}
	if streamKey == "" {
		streamKey = streamID
	}
	if streamKey == "" {
		streamKey = "default"
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	streams := l.active[userID]
	if streams == nil {
		streams = map[string]string{}
		l.active[userID] = streams
	}
	if _, exists := streams[streamKey]; exists {
		streams[streamKey] = streamID
		return nil
	}
	if len(streams) >= l.limit {
		return ErrAIUserBusy
	}
	streams[streamKey] = streamID
	return nil
}

func (l *UserAIStreamLimiter) ReleaseStream(userID string, streamKey string, streamID string) {
	if userID == "" {
		userID = "anonymous"
	}
	if streamKey == "" {
		streamKey = streamID
	}
	if streamKey == "" {
		streamKey = "default"
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	streams := l.active[userID]
	if streams == nil {
		return
	}
	if current, exists := streams[streamKey]; exists && (streamID == "" || current == streamID) {
		delete(streams, streamKey)
	}
	if len(streams) == 0 {
		delete(l.active, userID)
	}
}
