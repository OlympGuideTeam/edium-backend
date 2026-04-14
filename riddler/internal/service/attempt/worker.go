package attempt

import (
	"context"
	"log/slog"
	"time"
)

// RunExpiryWorker каждые 30 секунд завершает просроченные попытки.
// Блокирует горутину до отмены контекста.
func (s *Service) RunExpiryWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.finishExpired(ctx)
		}
	}
}

func (s *Service) finishExpired(ctx context.Context) {
	attempts, err := s.attempts.FindExpiredInProgress(ctx)
	if err != nil {
		slog.Error("expiry worker: find expired", "err", err)
		return
	}
	for i := range attempts {
		if err := s.finishAttempt(ctx, &attempts[i]); err != nil {
			slog.Error("expiry worker: finish attempt", "attempt_id", attempts[i].ID, "err", err)
		}
	}
}
