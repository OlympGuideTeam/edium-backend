package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const window = time.Minute

type Limiter struct {
	rdb    *redis.Client
	limits map[string]int
}

func NewLimiter(rdb *redis.Client, limits map[string]int) *Limiter {
	return &Limiter{rdb: rdb, limits: limits}
}

func (l *Limiter) Allow(ctx context.Context, service string) (bool, error) {
	limit, ok := l.limits[service]
	if !ok {
		return true, nil
	}

	key := fmt.Sprintf("ratelimit:charon:%s", service)
	now := time.Now()
	windowStart := now.Add(-window)

	pipe := l.rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))
	countCmd := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixNano()), Member: now.UnixNano()})
	pipe.Expire(ctx, key, window)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, fmt.Errorf("rate limit pipeline: %w", err)
	}

	count := countCmd.Val()
	if count >= int64(limit) {
		return false, nil
	}

	return true, nil
}
