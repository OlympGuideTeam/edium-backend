package telegram

import (
	"context"
	"herald/internal/pkg/correlation"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
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
	log.Printf("[tg-bot] started, username=@%s", h.bot.Self.UserName)

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

	ctx = correlation.WithID(ctx, uuid.New().String())
	msg := update.Message
	chatID := msg.Chat.ID

	if msg.Contact != nil {
		h.handleContact(ctx, chatID, msg.Contact)
		return
	}

	h.sendContactRequest(chatID)
}

func (h *Handler) handleContact(ctx context.Context, chatID int64, contact *tgbotapi.Contact) {
	phone := contact.PhoneNumber
	if len(phone) > 0 && phone[0] != '+' {
		phone = "+" + phone
	}

	log.Printf("[tg-bot] contact received correlation_id=%s chat_id=%d", correlation.IDFromContext(ctx), chatID)

	if err := h.service.RequestOTP(ctx, chatID, phone); err != nil {
		log.Printf("[tg-bot] RequestOTP error correlation_id=%s: %v", correlation.IDFromContext(ctx), err)
		h.sendMsg(tgbotapi.NewMessage(chatID, "Произошла ошибка. Попробуйте ещё раз."))
		return
	}

	reply := tgbotapi.NewMessage(chatID, "Запрос отправлен. Ожидайте код...")
	reply.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	h.sendMsg(reply)
}

func (h *Handler) sendContactRequest(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Привет! Нажмите кнопку ниже, чтобы поделиться номером телефона и получить код входа.")
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButtonContact("📱 Поделиться номером"),
		),
	)
	h.sendMsg(msg)
}

func (h *Handler) sendMsg(msg tgbotapi.Chattable) {
	if _, err := h.bot.Send(msg); err != nil {
		log.Printf("[tg-bot] send error: %v", err)
	}
}
