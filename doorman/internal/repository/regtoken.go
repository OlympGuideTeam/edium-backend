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

func (s *RedisRegTokenStore) Get(ctx context.Context, phone string) (string, error) {
	key := s.getKey(phone)
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

func (s *RedisRegTokenStore) Delete(ctx context.Context, phone string) error {
	return s.client.Del(ctx, s.getKey(phone)).Err()
}

func (s *RedisRegTokenStore) getKey(phone string) string {
	return fmt.Sprintf("regtoken:%s", phone)
}
