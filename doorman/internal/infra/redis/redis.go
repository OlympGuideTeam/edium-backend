package redis

import (
	"context"
	"doorman/internal/config"
	"fmt"
	"github.com/redis/go-redis/v9"
)

func New(cfg config.RedisConfig) (*redis.Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       0,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return rdb, nil
}
