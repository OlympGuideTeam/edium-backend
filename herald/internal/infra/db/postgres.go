package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"herald/internal/config"
)

func NewDB(cfg config.PostgresConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
