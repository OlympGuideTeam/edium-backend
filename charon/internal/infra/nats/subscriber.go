package nats

import (
	"context"
	"fmt"
	"log/slog"

	natsgo "github.com/nats-io/nats.go"
)

type MsgHandler func(ctx context.Context, data []byte) error

type Subscriber struct {
	conn *natsgo.Conn
}

func NewSubscriber(conn *natsgo.Conn) *Subscriber {
	return &Subscriber{conn: conn}
}

func (s *Subscriber) QueueSubscribe(ctx context.Context, subject, queue string, handler MsgHandler) error {
	sub, err := s.conn.QueueSubscribe(subject, queue, func(msg *natsgo.Msg) {
		msgCtx := propagator.Extract(ctx, natsHeaderCarrier{header: msg.Header})
		if err := handler(msgCtx, msg.Data); err != nil {
			slog.ErrorContext(msgCtx, "message handling error", "subject", subject, "err", err)
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe to %s: %w", subject, err)
	}
	<-ctx.Done()
	return sub.Drain()
}
