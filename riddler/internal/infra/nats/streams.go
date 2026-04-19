package nats

import (
	"errors"
	"fmt"

	natsgo "github.com/nats-io/nats.go"
)

const StreamGenerationRequested = "RIDDLER_GEN"

// EnsureGenerationStream создаёт JetStream-стрим для субъекта riddler.generation.requested,
// если он ещё не существует. Вызывается при старте Riddler, чтобы стрим был до первой публикации.
func EnsureGenerationStream(js natsgo.JetStreamContext) error {
	_, err := js.StreamInfo(StreamGenerationRequested)
	if err == nil {
		return nil
	}
	if !errors.Is(err, natsgo.ErrStreamNotFound) {
		return fmt.Errorf("stream info: %w", err)
	}
	_, err = js.AddStream(&natsgo.StreamConfig{
		Name:     StreamGenerationRequested,
		Subjects: []string{SubjectGenerationRequested},
	})
	if err != nil {
		return fmt.Errorf("add stream: %w", err)
	}
	return nil
}
