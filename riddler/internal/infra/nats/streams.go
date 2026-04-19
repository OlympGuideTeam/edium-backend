package nats

import (
	"errors"
	"fmt"

	natsgo "github.com/nats-io/nats.go"
)

const (
	StreamGenerationRequested = "RIDDLER_GEN"
	StreamGenerationCompleted = "SPHINX_GEN"
)

func EnsureGenerationStream(js natsgo.JetStreamContext) error {
	return ensureStream(js, StreamGenerationRequested, SubjectGenerationRequested)
}

func EnsureGenerationCompletedStream(js natsgo.JetStreamContext) error {
	return ensureStream(js, StreamGenerationCompleted, SubjectGenerationCompleted)
}

func ensureStream(js natsgo.JetStreamContext, stream, subject string) error {
	_, err := js.StreamInfo(stream)
	if err == nil {
		return nil
	}
	if !errors.Is(err, natsgo.ErrStreamNotFound) {
		return fmt.Errorf("stream info %s: %w", stream, err)
	}
	_, err = js.AddStream(&natsgo.StreamConfig{
		Name:     stream,
		Subjects: []string{subject},
	})
	if err != nil {
		return fmt.Errorf("add stream %s: %w", stream, err)
	}
	return nil
}
