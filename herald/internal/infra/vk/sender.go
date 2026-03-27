package vk

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/SevereCloud/vksdk/v2/api"
)

type Sender struct {
	vk *api.VK
}

func NewSender(vk *api.VK) *Sender {
	return &Sender{vk: vk}
}

func (s *Sender) Send(_ context.Context, chatID int64, text string) error {
	_, err := s.vk.MessagesSend(api.Params{
		"user_id":   chatID,
		"message":   text,
		"random_id": rand.Int63(),
	})
	if err != nil {
		return fmt.Errorf("vk send: %w", err)
	}
	return nil
}
