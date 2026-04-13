package db

import (
	"database/sql"
	"fmt"

	"github.com/XSAM/otelsql"
	_ "github.com/lib/pq"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"riddler/internal/config"
)

func NewDB(cfg config.PostgresConfig) (*sql.DB, error) {
	db, err := otelsql.Open("postgres", cfg.DSN,
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			OmitConnResetSession: true,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if _, err := otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(semconv.DBSystemPostgreSQL)); err != nil {
		return nil, fmt.Errorf("register db stats metrics: %w", err)
	}

	return db, nil
}
