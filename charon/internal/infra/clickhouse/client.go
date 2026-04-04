package clickhouse

import (
	"charon/internal/config"
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func New(cfg config.ClickHouseConfig) (clickhouse.Conn, error) {
	if cfg.DSN == "" {
		return nil, nil
	}

	opts, err := clickhouse.ParseDSN(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("clickhouse parse DSN: %w", err)
	}

	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("clickhouse open: %w", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}

	return conn, nil
}
