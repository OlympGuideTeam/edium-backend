package domain

import "time"

type UsageRecord struct {
	Timestamp        time.Time
	RequestID        string
	Service          string
	Model            string
	PromptTokens     uint32
	CompletionTokens uint32
	TotalTokens      uint32
	DurationMs       uint32
	Status           string
	Error            string
}
