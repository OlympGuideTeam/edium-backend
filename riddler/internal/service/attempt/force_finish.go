package attempt

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

func (s *Service) FinishInProgressBySession(ctx context.Context, sessionID uuid.UUID) error {
	summaries, err := s.attempts.FindBySessionID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("find attempts: %w", err)
	}
	for _, a := range summaries {
		if a.Status != domain.AttemptStatusInProgress {
			continue
		}
		full, err := s.attempts.GetByID(ctx, a.ID)
		if err != nil {
			return fmt.Errorf("get attempt %s: %w", a.ID, err)
		}
		if full == nil {
			continue
		}
		if err := s.finishAttempt(ctx, full); err != nil {
			return fmt.Errorf("finish attempt %s: %w", a.ID, err)
		}
	}
	return nil
}
