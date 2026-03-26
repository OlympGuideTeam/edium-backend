package nats

import (
	"context"
	"doorman/internal/pkg/correlation"
	"fmt"
	"log"

	natsgo "github.com/nats-io/nats.go"
)

type MsgHandler func(ctx context.Context, data []byte) error

type Subscriber struct {
	conn *natsgo.Conn
}

func NewSubscriber(conn *natsgo.Conn) *Subscriber {
	return &Subscriber{conn: conn}
}

// QueueSubscribe балансирует между репликами сервиса; блокируется до отмены ctx.
func (s *Subscriber) QueueSubscribe(ctx context.Context, subject, queue string, handler MsgHandler) error {
	sub, err := s.conn.QueueSubscribe(subject, queue, func(msg *natsgo.Msg) {
		msgCtx := ctx
		if id := msg.Header.Get("X-Correlation-Id"); id != "" {
			msgCtx = correlation.WithID(ctx, id)
		}
		if err := handler(msgCtx, msg.Data); err != nil {
			log.Printf("[nats] handler error subject=%s correlation_id=%s: %v",
				subject, msg.Header.Get("X-Correlation-Id"), err)
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe to %s: %w", subject, err)
	}
	<-ctx.Done()
	return sub.Drain()
}
