package nats

import (
	"context"
	"doorman/internal/pkg/correlation"
	"fmt"

	natsgo "github.com/nats-io/nats.go"
)

// Publisher публикует сообщения в NATS.
// Correlation ID прокидывается через заголовок X-Correlation-Id.
type Publisher struct {
	conn *natsgo.Conn
}

func NewPublisher(conn *natsgo.Conn) *Publisher {
	return &Publisher{conn: conn}
}

func (p *Publisher) Publish(ctx context.Context, subject string, data []byte) error {
	msg := &natsgo.Msg{
		Subject: subject,
		Data:    data,
		Header:  natsgo.Header{},
	}

	if id := correlation.IDFromContext(ctx); id != "" {
		msg.Header.Set("X-Correlation-Id", id)
	}

	if err := p.conn.PublishMsg(msg); err != nil {
		return fmt.Errorf("публикация в %s: %w", subject, err)
	}
	return nil
}
