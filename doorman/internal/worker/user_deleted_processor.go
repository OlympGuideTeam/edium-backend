package worker

import (
	"context"
	"doorman/internal/domain"
	"doorman/internal/pkg/correlation"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

const (
	userDeletedPollInterval = 2 * time.Second
	userDeletedBatchSize    = 10
	userDeletedRetryAfter   = 30 * time.Second
)

type identityStatusUpdater interface {
	UpdateStatus(ctx context.Context, userID string, status domain.IdentityStatus) error
}

type userTokensCleaner interface {
	DeleteUserTokens(ctx context.Context, userID string) error
}

type UserDeletedProcessor struct {
	tasks    taskRepository
	identity identityStatusUpdater
	tokens   userTokensCleaner
}

func NewUserDeletedProcessor(
	tasks taskRepository,
	identity identityStatusUpdater,
	tokens userTokensCleaner,
) *UserDeletedProcessor {
	return &UserDeletedProcessor{tasks: tasks, identity: identity, tokens: tokens}
}

func (w *UserDeletedProcessor) Run(ctx context.Context) error {
	log.Printf("[user-deleted-processor] started, interval=%s", userDeletedPollInterval)
	ticker := time.NewTicker(userDeletedPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				log.Printf("[user-deleted-processor] batch error: %v", err)
			}
		}
	}
}

func (w *UserDeletedProcessor) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.UserDeleted, userDeletedBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for _, t := range tasks {
		if err := w.processTask(ctx, t); err != nil {
			log.Printf("[user-deleted-processor] task_id=%s error: %v", t.ID, err)
			_ = w.tasks.MarkFailed(ctx, t.ID, err.Error(), userDeletedRetryAfter)
		}
	}
	return nil
}

func (w *UserDeletedProcessor) processTask(ctx context.Context, t domain.Task) error {
	var payload struct {
		UserID        string `json:"user_id"`
		CorrelationID string `json:"correlation_id,omitempty"`
	}
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	if payload.CorrelationID != "" {
		ctx = correlation.WithID(ctx, payload.CorrelationID)
	}

	log.Printf("[user-deleted-processor] task_id=%s correlation_id=%s user_id=%s",
		t.ID, payload.CorrelationID, payload.UserID)

	if err := w.identity.UpdateStatus(ctx, payload.UserID, domain.IdentityStatusDeleted); err != nil {
		return fmt.Errorf("UpdateStatus: %w", err)
	}

	if err := w.tokens.DeleteUserTokens(ctx, payload.UserID); err != nil {
		return fmt.Errorf("DeleteUserTokens: %w", err)
	}

	return w.tasks.MarkDone(ctx, t.ID)
}
