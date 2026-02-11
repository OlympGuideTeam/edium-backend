package repository

import (
	"context"
	otpsvc "doorman/internal/service/otp"
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

func (s *RedisOTPStore) Exists(ctx context.Context, phone string) (bool, error) {
	key := s.getKey(phone)
	count, err := s.client.Exists(ctx, key).Result()

	return count > 0, err
}

func (s *RedisOTPStore) Save(ctx context.Context, phone string, hashOtp string, ttl time.Duration) error {
	key := s.getKey(phone)
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

func (s *RedisOTPStore) Get(ctx context.Context, phone string) (*otpsvc.OtpData, error) {
	key := s.getKey(phone)
	cmd := s.client.HGetAll(ctx, key)

	data, err := cmd.Result()
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, nil
	}

	var otp otpsvc.OtpData
	err = cmd.Scan(&otp)

	return &otp, nil
}

func (s *RedisOTPStore) Delete(ctx context.Context, phone string) error {
	key := s.getKey(phone)
	cmd := s.client.Del(ctx, key)
	return cmd.Err()
}

func (s *RedisOTPStore) IncrAttempts(ctx context.Context, phone string) error {
	key := fmt.Sprintf("otp:%s", phone)
	return s.client.HIncrBy(ctx, key, "attempts", 1).Err()
}

func (s *RedisOTPStore) getKey(phone string) string {
	return fmt.Sprintf("otp:%s", phone)
}
