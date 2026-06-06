package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"opentab-server/internal/models"

	"github.com/redis/go-redis/v9"
)

type RedisAuthCache struct {
	client *redis.Client
}

func NewRedisAuthCache(ctx context.Context, redisURL string) (*RedisAuthCache, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opt)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return &RedisAuthCache{client: client}, nil
}

func (c *RedisAuthCache) GetSession(ctx context.Context, token string) (*AuthSession, error) {
	var session AuthSession
	if err := c.getJSON(ctx, sessionKey(token), &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (c *RedisAuthCache) SetSession(ctx context.Context, token string, session AuthSession, ttl time.Duration) error {
	if ttl <= 0 {
		return c.DeleteSession(ctx, token)
	}
	return c.setJSON(ctx, sessionKey(token), session, ttl)
}

func (c *RedisAuthCache) DeleteSession(ctx context.Context, token string) error {
	return c.client.Del(ctx, sessionKey(token)).Err()
}

func (c *RedisAuthCache) GetUserContext(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	if err := c.getJSON(ctx, userContextKey(userID), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *RedisAuthCache) SetUserContext(ctx context.Context, userID string, user models.User, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	return c.setJSON(ctx, userContextKey(userID), user, ttl)
}

func (c *RedisAuthCache) DeleteUserContext(ctx context.Context, userID string) error {
	return c.client.Del(ctx, userContextKey(userID)).Err()
}

func (c *RedisAuthCache) getJSON(ctx context.Context, key string, target any) error {
	raw, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return ErrMiss
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func (c *RedisAuthCache) setJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, raw, ttl).Err()
}

func sessionKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "auth:session:" + hex.EncodeToString(sum[:])
}

func userContextKey(userID string) string {
	return "auth:userctx:" + userID
}
