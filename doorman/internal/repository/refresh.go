package repository

import (
	"context"
	jwtsvc "doorman/internal/service/jwt"
	"errors"
	"github.com/redis/go-redis/v9"
	"time"
)

func NewRedisRefreshTokenStore(client *redis.Client) *RefreshTokenStore {
	return &RefreshTokenStore{client: client}
}

type RefreshTokenStore struct {
	client *redis.Client
}

func (s *RefreshTokenStore) SaveToken(ctx context.Context, jti string, userID string, ttl time.Duration) error {
	key := s.getJtiKey(jti)

	err := s.client.Set(ctx, key, userID, ttl).Err()
	if err != nil {
		return err
	}

	return s.client.SAdd(ctx, s.getUserKey(key), jti).Err()
}

func (s *RefreshTokenStore) GetAndDelToken(ctx context.Context, jti string) (string, error) {
	key := s.getJtiKey(jti)

	userID, err := s.client.GetDel(ctx, key).Result()

	if errors.Is(err, redis.Nil) {
		return "", jwtsvc.ErrRefreshTokenExpired
	}

	if err != nil {
		return "", err
	}

	return userID, nil
}

func (s *RefreshTokenStore) DeleteToken(ctx context.Context, jti string) error {
	key := s.getJtiKey(jti)
	return s.client.Del(ctx, key).Err()
}

func (s *RefreshTokenStore) DeleteUserTokens(ctx context.Context, userID string) error {
	userKey := s.getUserKey(userID)

	jtis, err := s.client.SMembers(ctx, userKey).Result()
	if err != nil {
		return err
	}

	for _, jti := range jtis {
		s.client.Del(ctx, s.getJtiKey(jti))
	}

	return s.client.Del(ctx, userKey).Err()
}

func (s *RefreshTokenStore) getUserKey(userID string) string {
	return "jwt:user:" + userID
}

func (s *RefreshTokenStore) getJtiKey(jti string) string {
	return "jwt:refresh:" + jti
}
