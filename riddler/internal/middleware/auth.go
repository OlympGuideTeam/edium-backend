package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"riddler/internal/infra/jwks"
	"riddler/internal/pkg/apperr"
	"riddler/internal/pkg/httpx"
)

type userIDKeyType struct{}

var userIDKey = userIDKeyType{}

// UserIDFromContext возвращает ID пользователя, установленный middleware Auth.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}

// Auth проверяет JWT из заголовка Authorization и прокидывает user_id в контекст.
func Auth(keys *jwks.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			httpx.WriteError(c, apperr.ErrUnauthorized)
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("неверный алгоритм подписи: %v", t.Header["alg"])
			}
			kid, _ := t.Header["kid"].(string)
			key, ok := keys.GetKey(kid)
			if !ok {
				return nil, fmt.Errorf("ключ не найден: %q", kid)
			}
			return key, nil
		})
		if err != nil || !token.Valid {
			httpx.WriteError(c, apperr.ErrUnauthorizedToken)
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			httpx.WriteError(c, apperr.ErrUnauthorizedClaims)
			c.Abort()
			return
		}

		subStr, _ := claims["sub"].(string)
		userID, err := uuid.Parse(subStr)
		if err != nil {
			httpx.WriteError(c, apperr.ErrUnauthorizedSub)
			c.Abort()
			return
		}

		ctx := context.WithValue(c.Request.Context(), userIDKey, userID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
