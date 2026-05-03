package live

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"riddler/internal/middleware"
	"riddler/internal/pkg/apperr"
	"riddler/internal/pkg/httpx"
	"riddler/internal/transport/dto"
)

type Handler struct {
	svc liveService
	hub *Hub
}

func NewHandler(svc liveService) *Handler {
	return &Handler{svc: svc, hub: newHub()}
}

func userIDFromCtx(c *gin.Context) (uuid.UUID, bool) {
	id, ok := middleware.UserIDFromContext(c.Request.Context())
	if !ok {
		httpx.WriteError(c, apperr.ErrUnauthorized)
		c.Abort()
	}
	return id, ok
}

func (h *Handler) ListLibraryLiveSessions(c *gin.Context) {
	authorID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	sessions, err := h.svc.ListLibraryLiveSessions(c.Request.Context(), authorID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	items := make([]dto.LibraryLiveSessionItem, len(sessions))
	for i, s := range sessions {
		items[i] = dto.LibraryLiveSessionItem{
			SessionID:         s.SessionID.String(),
			QuizTemplateID:    s.QuizTemplateID.String(),
			QuizTitle:         s.QuizTitle,
			Status:            string(s.Status),
			Phase:             string(s.Phase),
			JoinCode:          s.JoinCode,
			ParticipantsCount: s.ParticipantsCount,
			CreatedAt:         s.CreatedAt.UTC().Format(time.RFC3339),
		}
	}

	c.JSON(http.StatusOK, dto.ListLibraryLiveSessionsResponse{Sessions: items})
}

func (h *Handler) JoinLiveSession(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	var req dto.JoinLiveSessionRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.WriteError(c, apperr.ErrBadRequest)
			return
		}
	}

	var userID *uuid.UUID
	if id, ok := middleware.UserIDFromContext(c.Request.Context()); ok {
		userID = &id
	}

	attemptID, wsToken, err := h.svc.JoinLiveSession(c.Request.Context(), sessionID, userID, req.Name)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.JoinLiveSessionResponse{
		AttemptID: attemptID.String(),
		WsToken:   wsToken,
	})
}

func (h *Handler) GetLiveResultsStudent(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}
	attemptID, err := uuid.Parse(c.Query("attempt_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	res, err := h.svc.GetLiveResultsStudent(c.Request.Context(), sessionID, attemptID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	top := make([]dto.LiveLeaderboardRow, len(res.Top))
	for i, p := range res.Top {
		row := dto.LiveLeaderboardRow{
			Position:  p.Position,
			AttemptID: p.AttemptID.String(),
			Score:     p.Score,
			IsMe:      p.AttemptID == attemptID,
		}
		if p.UserID != nil {
			s := p.UserID.String()
			row.UserID = &s
		}
		row.Name = p.Name
		top[i] = row
	}

	c.JSON(http.StatusOK, dto.LiveResultsStudentResponse{
		MyPosition:        res.MyPosition,
		TotalParticipants: res.TotalParticipants,
		MyScore:           res.MyScore,
		MaxScore:          res.MaxScore,
		CorrectCount:      res.CorrectCount,
		QuestionsCount:    res.QuestionsCount,
		Top:               top,
	})
}

func (h *Handler) GetLiveResultsTeacher(c *gin.Context) {
	authorID, ok := userIDFromCtx(c)
	if !ok {
		return
	}
	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	res, err := h.svc.GetLiveResultsTeacher(c.Request.Context(), sessionID, authorID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	questions := make([]dto.LiveQuestionResult, len(res.Questions))
	for i, q := range res.Questions {
		qr := dto.LiveQuestionResult{
			QuestionID:    q.QuestionID.String(),
			OrderIndex:    q.OrderIndex,
			Text:          q.Text,
			Type:          q.Type,
			CorrectRate:   q.CorrectRate,
			AnsweredCount: q.AnsweredCount,
			CorrectCount:  q.CorrectCount,
			AvgTimesMs:    q.AvgTimesMs,
		}
		for _, d := range q.Distribution {
			qr.Distribution = append(qr.Distribution, dto.LiveOptionStat{
				OptionID:  d.OptionID.String(),
				Count:     d.Count,
				IsCorrect: d.IsCorrect,
			})
		}
		questions[i] = qr
	}

	leaderboard := make([]dto.LiveResultsTeacherAttempt, len(res.Leaderboard))
	for i, p := range res.Leaderboard {
		row := dto.LiveResultsTeacherAttempt{
			Position:     p.Position,
			AttemptID:    p.AttemptID.String(),
			Score:        p.Score,
			CorrectCount: p.CorrectCount,
		}
		if p.UserID != nil {
			s := p.UserID.String()
			row.UserID = &s
		}
		row.Name = p.Name
		leaderboard[i] = row
	}

	c.JSON(http.StatusOK, dto.LiveResultsTeacherResponse{
		Questions:   questions,
		Leaderboard: leaderboard,
	})
}

func (h *Handler) ResolveLiveCode(c *gin.Context) {
	code := c.Param("code")

	meta, err := h.svc.ResolveLiveCode(c.Request.Context(), code)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ResolveLiveCodeResponse{
		SessionID:          meta.SessionID.String(),
		QuizTitle:          meta.QuizTitle,
		QuestionCount:      meta.QuestionCount,
		IsAnonymousAllowed: meta.IsAnonymousAllowed,
	})
}

func (h *Handler) StartLiveSession(c *gin.Context) {
	authorID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	wsToken, joinCode, err := h.svc.StartLiveSession(c.Request.Context(), sessionID, authorID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.StartLiveSessionResponse{WsToken: wsToken, JoinCode: joinCode})
}

func (h *Handler) CreateLiveLibrarySession(c *gin.Context) {
	authorID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	var req dto.CreateLiveLibrarySessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	quizTemplateID, err := uuid.Parse(req.QuizTemplateID)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	sessionID, err := h.svc.CreateLiveLibrarySession(c.Request.Context(), authorID, quizTemplateID, req.QuestionTimeLimitSec)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CreateLiveSessionResponse{SessionID: sessionID.String()})
}

func (h *Handler) CreateLiveCourseSession(c *gin.Context) {
	authorID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	var req dto.CreateLiveCourseSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	quizTemplateID, err := uuid.Parse(req.QuizTemplateID)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}
	moduleID, err := uuid.Parse(req.ModuleID)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	sessionID, err := h.svc.CreateLiveCourseSession(c.Request.Context(), authorID, quizTemplateID, moduleID, req.QuestionTimeLimitSec)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.CreateLiveSessionResponse{SessionID: sessionID.String()})
}
