package nats

import (
	"doorman/internal/config"
	"fmt"

	"github.com/nats-io/nats.go"
)

// New создаёт подключение к NATS.
func New(cfg config.NATSConfig) (*nats.Conn, error) {
	conn, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("подключение к NATS: %w", err)
	}
	return conn, nil
}
