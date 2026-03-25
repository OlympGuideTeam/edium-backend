package correlation

import "context"

type contextKey struct{}

// WithID добавляет correlation_id в контекст.
func WithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// IDFromContext извлекает correlation_id из контекста.
// Возвращает пустую строку, если ID не установлен.
func IDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(contextKey{}).(string)
	return id
}
