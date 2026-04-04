package nats

import (
	"charon/internal/config"
	"fmt"

	"github.com/nats-io/nats.go"
)

func New(cfg config.NATSConfig) (*nats.Conn, error) {
	conn, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return conn, nil
}
