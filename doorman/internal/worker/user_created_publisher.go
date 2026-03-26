package worker

import (
	"context"
	"doorman/internal/domain"
	natsinf "doorman/internal/infra/nats"
	"doorman/internal/pkg/correlation"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

const (
	userCreatedPollInterval = 2 * time.Second
	userCreatedBatchSize    = 10
	userCreatedRetryAfter   = 30 * time.Second
)

type UserCreatedPublisher struct {
	tasks     taskRepository
	publisher *natsinf.Publisher
}

func NewUserCreatedPublisher(tasks taskRepository, publisher *natsinf.Publisher) *UserCreatedPublisher {
	return &UserCreatedPublisher{tasks: tasks, publisher: publisher}
}

func (w *UserCreatedPublisher) Run(ctx context.Context) error {
	log.Printf("[user-created-publisher] started, interval=%s", userCreatedPollInterval)
	ticker := time.NewTicker(userCreatedPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				log.Printf("[user-created-publisher] batch error: %v", err)
			}
		}
	}
}

type userCreatedMsg struct {
	UserID        string `json:"user_id"`
	Phone         string `json:"phone"`
	Name          string `json:"name"`
	Surname       string `json:"surname"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

func (w *UserCreatedPublisher) processBatch(ctx context.Context) error {
	tasks, err := w.tasks.ClaimPending(ctx, domain.UserCreated, userCreatedBatchSize)
	if err != nil {
		return fmt.Errorf("ClaimPending: %w", err)
	}
	for _, t := range tasks {
		if err := w.processTask(ctx, t); err != nil {
			log.Printf("[user-created-publisher] task_id=%s error: %v", t.ID, err)
			_ = w.tasks.MarkFailed(ctx, t.ID, err.Error(), userCreatedRetryAfter)
		}
	}
	return nil
}

func (w *UserCreatedPublisher) processTask(ctx context.Context, t domain.Task) error {
	var payload userCreatedMsg
	if err := json.Unmarshal(t.Payload, &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	pubCtx := ctx
	if payload.CorrelationID != "" {
		pubCtx = correlation.WithID(ctx, payload.CorrelationID)
	}

	log.Printf("[user-created-publisher] task_id=%s correlation_id=%s user_id=%s",
		t.ID, payload.CorrelationID, payload.UserID)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	if err := w.publisher.Publish(pubCtx, natsinf.SubjectUserCreated, data); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return w.tasks.MarkDone(ctx, t.ID)
}
