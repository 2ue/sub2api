package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const openAI429CounterPrefix = "openai429_count:account:"

var openAI429CounterIncrScript = redis.NewScript(`
	local key = KEYS[1]
	local ttl = tonumber(ARGV[1])

	local count = redis.call('INCR', key)
	if count == 1 then
		redis.call('EXPIRE', key, ttl)
	end

	return count
`)

type openAI429CounterCache struct {
	rdb *redis.Client
}

// NewOpenAI429CounterCache 创建特殊 OpenAI OAuth 429 计数器缓存实例。
func NewOpenAI429CounterCache(rdb *redis.Client) service.OpenAI429CounterCache {
	return &openAI429CounterCache{rdb: rdb}
}

// IncrementOpenAI429Count 原子递增 429 计数并返回当前值。
func (c *openAI429CounterCache) IncrementOpenAI429Count(ctx context.Context, accountID int64, windowSeconds int) (int64, error) {
	key := fmt.Sprintf("%s%d", openAI429CounterPrefix, accountID)
	if windowSeconds <= 0 {
		windowSeconds = 1
	}

	result, err := openAI429CounterIncrScript.Run(ctx, c.rdb, []string{key}, windowSeconds).Int64()
	if err != nil {
		return 0, fmt.Errorf("increment openai 429 count: %w", err)
	}
	return result, nil
}

// ResetOpenAI429Count 清零 429 计数器。
func (c *openAI429CounterCache) ResetOpenAI429Count(ctx context.Context, accountID int64) error {
	key := fmt.Sprintf("%s%d", openAI429CounterPrefix, accountID)
	return c.rdb.Del(ctx, key).Err()
}
