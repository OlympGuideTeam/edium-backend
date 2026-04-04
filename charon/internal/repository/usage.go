package repository

import (
	"charon/internal/domain"
	"context"
	"fmt"
	"log/slog"

	"github.com/ClickHouse/clickhouse-go/v2"
)

type ClickHouseUsageWriter struct {
	conn clickhouse.Conn
}

func NewClickHouseUsageWriter(conn clickhouse.Conn) *ClickHouseUsageWriter {
	return &ClickHouseUsageWriter{conn: conn}
}

func (w *ClickHouseUsageWriter) LogUsage(ctx context.Context, record domain.UsageRecord) error {
	if w.conn == nil {
		slog.DebugContext(ctx, "clickhouse disabled, skipping usage log")
		return nil
	}

	err := w.conn.Exec(ctx, `
		INSERT INTO llm_usage (
			timestamp, request_id, service, model,
			prompt_tokens, completion_tokens, total_tokens,
			duration_ms, status, error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.Timestamp,
		record.RequestID,
		record.Service,
		record.Model,
		record.PromptTokens,
		record.CompletionTokens,
		record.TotalTokens,
		record.DurationMs,
		record.Status,
		record.Error,
	)
	if err != nil {
		return fmt.Errorf("clickhouse insert usage: %w", err)
	}
	return nil
}
