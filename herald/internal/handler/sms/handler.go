package smshandler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// Handler обслуживает HTTP-эндпоинты для Android SMS-шлюза.
// GET  /herald/v1/sms/tasks        — поллинг ожидающих задач
// POST /herald/v1/sms/tasks/{id}/ack — подтверждение отправки
type Handler struct {
	tasks  SMSTaskRepository
	apiKey string
}

func NewHandler(tasks SMSTaskRepository, apiKey string) *Handler {
	return &Handler{tasks: tasks, apiKey: apiKey}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /herald/v1/sms/tasks", h.auth(h.listTasks))
	mux.HandleFunc("POST /herald/v1/sms/tasks/{id}/ack", h.auth(h.ackTask))
}

func (h *Handler) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" || token != h.apiKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

type smsTaskResponse struct {
	ID    string `json:"id"`
	Phone string `json:"phone"`
	Text  string `json:"text"`
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.tasks.ListPending(r.Context(), 10)
	if err != nil {
		slog.ErrorContext(r.Context(), "sms-handler: ListPending", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := make([]smsTaskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp = append(resp, smsTaskResponse{
			ID:    t.ID.String(),
			Phone: t.Phone,
			Text:  t.Text,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(r.Context(), "sms-handler: encode response", "err", err)
	}
}

type ackRequest struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (h *Handler) ackTask(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req ackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := h.tasks.Ack(r.Context(), id, req.Success, req.Error); err != nil {
		slog.ErrorContext(r.Context(), "sms-handler: Ack", "task_id", id, "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
