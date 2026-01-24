package kafka

import (
	"context"
	"doorman/internal/config"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(cfg config.KafkaConfig) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(cfg.Brokers...),
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *Producer) Publish(
	ctx context.Context,
	topic string,
	key string,
	value []byte,
) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	})
}
