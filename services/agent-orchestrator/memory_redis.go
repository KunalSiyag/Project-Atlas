package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient is the subset of go-redis used by RedisShortMemory (testable).
type RedisClient interface {
	RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	LRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	Ping(ctx context.Context) *redis.StatusCmd
}

// RedisShortMemory stores short-term facts in Redis lists keyed by run ID.
// Long-term operations delegate to Long (optional; empty ListLong if nil).
type RedisShortMemory struct {
	Client RedisClient
	// KeyPrefix defaults to "atlas:stm:".
	KeyPrefix string
	// TTL applied on each PutShort (0 = no expiry).
	TTL time.Duration
	// Long holds durable long-term memory (in-memory or Postgres).
	Long MemoryStore
}

// NewRedisShortMemory wraps a redis client. long may be nil (long ops error/empty).
func NewRedisShortMemory(client RedisClient, long MemoryStore) *RedisShortMemory {
	return &RedisShortMemory{
		Client:    client,
		KeyPrefix: "atlas:stm:",
		TTL:       24 * time.Hour,
		Long:      long,
	}
}

// OpenRedisShortMemory dials Redis via URL (e.g. redis://127.0.0.1:6379/0).
func OpenRedisShortMemory(ctx context.Context, redisURL string, long MemoryStore) (*RedisShortMemory, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opt)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return NewRedisShortMemory(client, long), nil
}

func (m *RedisShortMemory) key(runID string) string {
	prefix := m.KeyPrefix
	if prefix == "" {
		prefix = "atlas:stm:"
	}
	return prefix + runID
}

func (m *RedisShortMemory) PutShort(ctx context.Context, runID, content string) error {
	if m == nil || m.Client == nil {
		return errors.New("redis short memory not configured")
	}
	if runID == "" || content == "" {
		return errors.New("run_id and content required")
	}
	fact := MemoryFact{
		ID:        newID(),
		Scope:     "short",
		RunID:     runID,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}
	raw, err := json.Marshal(fact)
	if err != nil {
		return err
	}
	key := m.key(runID)
	if err := m.Client.RPush(ctx, key, string(raw)).Err(); err != nil {
		return err
	}
	if m.TTL > 0 {
		_ = m.Client.Expire(ctx, key, m.TTL).Err()
	}
	return nil
}

func (m *RedisShortMemory) ListShort(ctx context.Context, runID string) ([]MemoryFact, error) {
	if m == nil || m.Client == nil {
		return nil, errors.New("redis short memory not configured")
	}
	if runID == "" {
		return nil, nil
	}
	vals, err := m.Client.LRange(ctx, m.key(runID), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	out := make([]MemoryFact, 0, len(vals))
	for _, v := range vals {
		var f MemoryFact
		if err := json.Unmarshal([]byte(v), &f); err != nil {
			f = MemoryFact{Scope: "short", RunID: runID, Content: v}
		}
		out = append(out, f)
	}
	return out, nil
}

func (m *RedisShortMemory) PutLong(ctx context.Context, fact MemoryFact) error {
	if m.Long == nil {
		return errors.New("long-term memory not configured")
	}
	return m.Long.PutLong(ctx, fact)
}

func (m *RedisShortMemory) ListLong(ctx context.Context, workspaceID, userID string, limit int) ([]MemoryFact, error) {
	if m.Long == nil {
		return nil, nil
	}
	return m.Long.ListLong(ctx, workspaceID, userID, limit)
}

// Close closes the underlying redis client when it supports Close.
func (m *RedisShortMemory) Close() error {
	if m == nil || m.Client == nil {
		return nil
	}
	if c, ok := m.Client.(interface{ Close() error }); ok {
		return c.Close()
	}
	return nil
}
