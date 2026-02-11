package repository

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisRegTokenStore struct {
	client *redis.Client
}

func NewRedisRegTokenStore(client *redis.Client) *RedisRegTokenStore {
	return &RedisRegTokenStore{client: client}
}

func (s *RedisRegTokenStore) Save(ctx context.Context, phone string, regToken string, ttl time.Duration) error {
	key := s.getKey(phone)
	return s.client.Set(ctx, key, regToken, ttl).Err()
}

func (s *RedisRegTokenStore) getKey(phone string) string {
	return fmt.Sprintf("regtoken:%s", phone)
}
