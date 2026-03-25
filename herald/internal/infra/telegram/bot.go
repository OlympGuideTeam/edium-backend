package telegram

import (
	"fmt"
	"herald/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func New(cfg config.TelegramConfig) (*tgbotapi.BotAPI, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}
	return bot, nil
}
