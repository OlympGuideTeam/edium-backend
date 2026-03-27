package vk

import (
	"context"
	"encoding/json"
	"herald/internal/domain"
	"log/slog"
	"math/rand"
	"regexp"
	"strings"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/events"
	lp "github.com/SevereCloud/vksdk/v2/longpoll-bot"
	"go.opentelemetry.io/otel"
)

var phoneRe = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)

type Handler struct {
	lp      *lp.LongPoll
	vk      *api.VK
	service OTPService
}

func NewHandler(longPoll *lp.LongPoll, vk *api.VK, service OTPService) *Handler {
	h := &Handler{lp: longPoll, vk: vk, service: service}
	longPoll.MessageNew(func(ctx context.Context, obj events.MessageNewObject) {
		h.handleMessage(ctx, obj)
	})
	return h
}

func (h *Handler) Run(ctx context.Context) error {
	slog.Info("vk-bot: запущен")
	go func() {
		<-ctx.Done()
		h.lp.Shutdown()
	}()
	return h.lp.Run()
}

func (h *Handler) handleMessage(ctx context.Context, obj events.MessageNewObject) {
	userID := int64(obj.Message.FromID)
	text := strings.TrimSpace(obj.Message.Text)
	payload := obj.Message.Payload

	if text == "/start" || isStartEvent(payload) {
		h.sendWelcome(userID)
		return
	}

	if isSendButton(payload) {
		h.sendPhoneRequest(userID)
		return
	}

	if phoneRe.MatchString(text) {
		h.handlePhone(ctx, userID, text)
		return
	}

	h.sendWithButton(userID, "Нажмите кнопку ниже, чтобы получить код входа.")
}

func (h *Handler) handlePhone(ctx context.Context, userID int64, phone string) {
	ctx, span := otel.Tracer("herald").Start(ctx, "vk.handlePhone")
	defer span.End()

	if len(phone) > 0 && phone[0] != '+' {
		phone = "+" + phone
	}

	slog.InfoContext(ctx, "vk-bot: телефон получен", "user_id", userID)

	if err := h.service.RequestOTP(ctx, userID, phone, domain.ChannelVK); err != nil {
		slog.ErrorContext(ctx, "vk-bot: ошибка RequestOTP", "user_id", userID, "err", err)
		h.sendMsg(userID, "Произошла ошибка. Попробуйте ещё раз.")
		return
	}

	h.sendMsg(userID, "Запрос отправлен. Ожидайте код...")
}

func (h *Handler) sendWelcome(userID int64) {
	h.sendWithButton(userID, "Привет! Я помогу войти в приложение Edium — отправлю код подтверждения прямо сюда.\n\nНажмите кнопку ниже, чтобы получить код.")
}

func (h *Handler) sendPhoneRequest(userID int64) {
	h.sendMsg(userID, "Введите номер телефона в формате +7XXXXXXXXXX:")
}

func (h *Handler) sendWithButton(userID int64, text string) {
	_, err := h.vk.MessagesSend(api.Params{
		"user_id":   userID,
		"message":   text,
		"random_id": rand.Int63(),
		"keyboard":  buildKeyboard(),
	})
	if err != nil {
		slog.Error("vk-bot: ошибка отправки", "user_id", userID, "err", err)
	}
}

func (h *Handler) sendMsg(userID int64, text string) {
	_, err := h.vk.MessagesSend(api.Params{
		"user_id":   userID,
		"message":   text,
		"random_id": rand.Int63(),
	})
	if err != nil {
		slog.Error("vk-bot: ошибка отправки", "user_id", userID, "err", err)
	}
}

func buildKeyboard() string {
	kb := map[string]any{
		"one_time": true,
		"buttons": [][]map[string]any{
			{
				{
					"action": map[string]any{
						"type":    "text",
						"label":   "📲 Получить код",
						"payload": `{"cmd":"send"}`,
					},
					"color": "primary",
				},
			},
		},
	}
	b, _ := json.Marshal(kb)
	return string(b)
}

func isSendButton(payload string) bool {
	return strings.Contains(payload, `"cmd":"send"`)
}

func isStartEvent(payload string) bool {
	return strings.Contains(payload, `"command":"start"`)
}
