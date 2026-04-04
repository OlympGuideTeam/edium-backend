package completion

import (
	"charon/internal/domain"
	"charon/internal/pkg/apperr"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

var ErrRateLimitExceeded = apperr.New("RATE_LIMIT_EXCEEDED", "Rate limit exceeded", 429)

type Service struct {
	llm         LLMClient
	usage       UsageLogger
	rateLimiter RateLimiter
	tasks       TaskScheduler
}

func NewService(llm LLMClient, usage UsageLogger, rateLimiter RateLimiter, tasks TaskScheduler) *Service {
	return &Service{
		llm:         llm,
		usage:       usage,
		rateLimiter: rateLimiter,
		tasks:       tasks,
	}
}

func (s *Service) ProcessCompletion(ctx context.Context, req domain.CompletionRequest) error {
	allowed, err := s.rateLimiter.Allow(ctx, req.Service)
	if err != nil {
		slog.ErrorContext(ctx, "rate limiter error", "err", err)
	}
	if !allowed {
		return s.scheduleResponse(ctx, domain.CompletionResponse{
			RequestID: req.RequestID,
			Model:     req.Model,
			Error:     ErrRateLimitExceeded.Code,
		})
	}

	start := time.Now()
	resp, err := s.llm.Complete(ctx, req)
	duration := time.Since(start)

	record := domain.UsageRecord{
		Timestamp: start,
		RequestID: req.RequestID,
		Service:   req.Service,
		Model:     req.Model,
		DurationMs: uint32(duration.Milliseconds()),
		Status:    "ok",
	}

	if err != nil {
		record.Status = "error"
		record.Error = err.Error()
		s.logUsage(ctx, record)

		return s.scheduleResponse(ctx, domain.CompletionResponse{
			RequestID: req.RequestID,
			Model:     req.Model,
			Error:     fmt.Sprintf("llm error: %s", err.Error()),
		})
	}

	record.PromptTokens = uint32(resp.Usage.PromptTokens)
	record.CompletionTokens = uint32(resp.Usage.CompletionTokens)
	record.TotalTokens = uint32(resp.Usage.TotalTokens)
	s.logUsage(ctx, record)

	resp.RequestID = req.RequestID
	return s.scheduleResponse(ctx, *resp)
}

func (s *Service) scheduleResponse(ctx context.Context, resp domain.CompletionResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	return s.tasks.Schedule(ctx, domain.CompletionCompleted, data)
}

func (s *Service) logUsage(ctx context.Context, record domain.UsageRecord) {
	if err := s.usage.LogUsage(ctx, record); err != nil {
		slog.ErrorContext(ctx, "failed to log usage", "err", err)
	}
}
