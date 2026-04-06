package telegram

import (
	"context"
	"herald/internal/domain"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.opentelemetry.io/otel"
)

type Handler struct {
	bot     *tgbotapi.BotAPI
	service OTPService
}

func NewHandler(bot *tgbotapi.BotAPI, service OTPService) *Handler {
	return &Handler{bot: bot, service: service}
}

func (h *Handler) Run(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := h.bot.GetUpdatesChan(u)
	slog.Info("tg-bot: запущен", "username", h.bot.Self.UserName)

	for {
		select {
		case <-ctx.Done():
			h.bot.StopReceivingUpdates()
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			h.handleUpdate(ctx, update)
		}
	}
}

func (h *Handler) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	msg := update.Message
	chatID := msg.Chat.ID

	if msg.Contact != nil {
		h.handleContact(ctx, chatID, msg.Contact)
		return
	}

	if msg.Text == "/start" {
		h.sendContactRequest(chatID)
		return
	}

	h.sendContactRequest(chatID)
}

func (h *Handler) handleContact(ctx context.Context, chatID int64, contact *tgbotapi.Contact) {
	ctx, span := otel.Tracer("herald").Start(ctx, "tg.handleContact")
	defer span.End()

	phone := contact.PhoneNumber
	if len(phone) > 0 && phone[0] != '+' {
		phone = "+" + phone
	}

	slog.InfoContext(ctx, "tg-bot: контакт получен", "chat_id", chatID, "phone", phone)

	if err := h.service.RequestOTP(ctx, chatID, phone, domain.ChannelTG); err != nil {
		slog.ErrorContext(ctx, "tg-bot: ошибка RequestOTP", "chat_id", chatID, "err", err)
		h.sendContactRequest(chatID)
		return
	}

	reply := tgbotapi.NewMessage(chatID, "Запрос отправлен. Ожидайте код...")
	reply.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	h.sendMsg(reply)
}

func (h *Handler) sendContactRequest(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Нажмите кнопку ниже, чтобы поделиться номером телефона и получить код входа в Edium.")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButtonContact("📱 Поделиться номером"),
		),
	)
	h.sendMsg(msg)
}

func (h *Handler) sendMsg(msg tgbotapi.Chattable) {
	if _, err := h.bot.Send(msg); err != nil {
		slog.Error("tg-bot: ошибка отправки", "err", err)
	}
}
