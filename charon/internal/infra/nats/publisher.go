package nats

import (
	"context"
	"fmt"

	natsgo "github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/propagation"
)

var propagator = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
)

type natsHeaderCarrier struct {
	header natsgo.Header
}

func (c natsHeaderCarrier) Get(key string) string      { return c.header.Get(key) }
func (c natsHeaderCarrier) Set(key string, val string) { c.header.Set(key, val) }
func (c natsHeaderCarrier) Keys() []string             { return nil }

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
	propagator.Inject(ctx, natsHeaderCarrier{header: msg.Header})
	if err := p.conn.PublishMsg(msg); err != nil {
		return fmt.Errorf("publish to %s: %w", subject, err)
	}
	return nil
}
