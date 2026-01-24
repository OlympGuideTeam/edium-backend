package kafka

import (
	"context"
	"github.com/segmentio/kafka-go"
	"log"
)

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, groupID, topic string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			GroupID: groupID,
			Topic:   topic,
		}),
	}
}

func (c *Consumer) Consume(ctx context.Context, handler func([]byte) error) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return err
		}

		if err = handler(msg.Value); err != nil {
			log.Fatal(err) // TODO: think about this
		}
	}
}
