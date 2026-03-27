package telegram

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Sender struct {
	bot *tgbotapi.BotAPI
}

func NewSender(bot *tgbotapi.BotAPI) *Sender {
	return &Sender{bot: bot}
}

func (s *Sender) Send(_ context.Context, chatID int64, text string) error {
	_, err := s.bot.Send(tgbotapi.NewMessage(chatID, text))
	if err != nil {
		return fmt.Errorf("telegram send: %w", err)
	}
	return nil
}
