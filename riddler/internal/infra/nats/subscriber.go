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

// QueueSubscribe балансирует между репликами сервиса; блокируется до отмены ctx.
func (s *Subscriber) QueueSubscribe(ctx context.Context, subject, queue string, handler MsgHandler) error {
	sub, err := s.conn.QueueSubscribe(subject, queue, func(msg *natsgo.Msg) {
		msgCtx := propagator.Extract(ctx, natsHeaderCarrier{header: msg.Header})
		if err := handler(msgCtx, msg.Data); err != nil {
			slog.ErrorContext(msgCtx, "ошибка обработки сообщения", "subject", subject, "err", err)
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe to %s: %w", subject, err)
	}
	<-ctx.Done()
	return sub.Drain()
}

// JetStreamSubscriber подписывается через JetStream durable consumer.
// Ack отправляется при успехе обработки, Nak — при ошибке (NATS переотправит).
type JetStreamSubscriber struct {
	js natsgo.JetStreamContext
}

func NewJetStreamSubscriber(js natsgo.JetStreamContext) *JetStreamSubscriber {
	return &JetStreamSubscriber{js: js}
}

func (s *JetStreamSubscriber) Subscribe(ctx context.Context, subject, stream, consumer string, handler MsgHandler) error {
	sub, err := s.js.Subscribe(subject, func(msg *natsgo.Msg) {
		msgCtx := propagator.Extract(ctx, natsHeaderCarrier{header: msg.Header})
		if err := handler(msgCtx, msg.Data); err != nil {
			slog.ErrorContext(msgCtx, "ошибка обработки JS-сообщения", "subject", subject, "err", err)
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
	}, natsgo.Durable(consumer), natsgo.ManualAck(), natsgo.BindStream(stream))
	if err != nil {
		return fmt.Errorf("js subscribe to %s: %w", subject, err)
	}
	<-ctx.Done()
	return sub.Drain()
}
