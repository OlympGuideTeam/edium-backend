package repository

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisOTPStore struct {
	client *redis.Client
}

func NewRedisOTPStore(client *redis.Client) *RedisOTPStore {
	return &RedisOTPStore{client: client}
}

func (s *RedisOTPStore) Exists(ctx context.Context, phone string) (exists bool, err error) {
	key := fmt.Sprintf("otp:%s", phone)
	count, err := s.client.Exists(ctx, key).Result()

	return count > 0, err
}

func (s *RedisOTPStore) Save(ctx context.Context, phone string, hashOtp string, ttl time.Duration) error {
	key := fmt.Sprintf("otp:%s", phone)

	err := s.client.HSet(
		ctx, key,
		"hash", hashOtp,
		"attempts", 0,
	).Err()

	if err != nil {
		return err
	}

	return s.client.Expire(ctx, key, ttl).Err()
}
