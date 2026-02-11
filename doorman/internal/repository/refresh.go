package repository

import (
	"context"
	"github.com/redis/go-redis/v9"
)

func NewRedisRefreshTokenStore(client *redis.Client) *RefreshTokenStore {
	return &RefreshTokenStore{client: client}
}

type RefreshTokenStore struct {
	client *redis.Client
}

func (s *RefreshTokenStore) Save(ctx context.Context, userID, token string) error {
	key := s.getKey(userID)
	return s.client.LPush(ctx, key, token).Err()
}

func (s *RefreshTokenStore) getKey(userID string) string {
	return "jwt:refresh:" + userID
}
