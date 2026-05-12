package attempt

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/middleware"
	"riddler/internal/pkg/apperr"
	"riddler/internal/pkg/httpx"
	"riddler/internal/transport/dto"
)

func (h *Handler) CreateAttempt(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	attempt, questions, err := h.service.Create(c.Request.Context(), sessionID, userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.CreateAttemptResponse{
		AttemptID: attempt.ID.String(),
		Questions: toQuestionsResponse(questions),
	})
}

func (h *Handler) SubmitAnswer(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	attemptID, err := uuid.Parse(c.Param("attempt_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	var req dto.SubmitAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	questionID, err := uuid.Parse(req.QuestionID)
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	if err := h.service.SubmitAnswer(c.Request.Context(), attemptID, userID, questionID, req.AnswerData); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) Finish(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	attemptID, err := uuid.Parse(c.Param("attempt_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	if err := h.service.Finish(c.Request.Context(), attemptID, userID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) ListSessionAttempts(c *gin.Context) {
	teacherID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	attempts, err := h.service.ListSessionAttempts(c.Request.Context(), sessionID, teacherID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	resp := make([]dto.AttemptSummaryResponse, len(attempts))
	for i, a := range attempts {
		item := dto.AttemptSummaryResponse{
			AttemptID: a.ID.String(),
			Status:    string(a.Status),
			Score:     a.Grade,
		}
		if a.UserID != uuid.Nil {
			item.UserID = a.UserID.String()
		}
		resp[i] = item
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetAttemptReview(c *gin.Context) {
	var callerID *uuid.UUID
	if id, ok := middleware.UserIDFromContext(c.Request.Context()); ok {
		callerID = &id
	}

	attemptID, err := uuid.Parse(c.Param("attempt_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	attempt, answers, enriched, err := h.service.GetAttemptReview(c.Request.Context(), attemptID, callerID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	var finishedAt *string
	if attempt.FinishedAt != nil {
		s := attempt.FinishedAt.Format("2006-01-02T15:04:05Z")
		finishedAt = &s
	}

	reviewAnswers := make([]dto.AnswerReviewResponse, len(answers))
	for i := range answers {
		a := answers[i]
		var src *string
		if a.FinalSource != nil {
			s := string(*a.FinalSource)
			src = &s
		}
		var opts []dto.AnswerOptionTeacherResponse
		var studentOpts []dto.AnswerOptionForStudentResponse
		if len(a.Options) > 0 {
			studentOpts = make([]dto.AnswerOptionForStudentResponse, len(a.Options))
			for j, o := range a.Options {
				studentOpts[j] = dto.AnswerOptionForStudentResponse{
					ID:   o.ID.String(),
					Text: o.Text,
				}
			}
			if enriched {
				opts = make([]dto.AnswerOptionTeacherResponse, len(a.Options))
				for j, o := range a.Options {
					opts[j] = dto.AnswerOptionTeacherResponse{
						ID:        o.ID.String(),
						Text:      o.Text,
						IsCorrect: o.IsCorrect,
					}
				}
			}
		}
		var metadata map[string]any
		if enriched {
			metadata = a.Metadata
		} else {
			metadata = sanitizeMetadataForStudent(a.QuestionType, a.Metadata)
		}
		reviewAnswers[i] = dto.AnswerReviewResponse{
			SubmissionID:   a.SubmissionID.String(),
			QuestionID:     a.QuestionID.String(),
			QuestionType:   a.QuestionType,
			QuestionText:   a.QuestionText,
			AnswerData:     a.AnswerData,
			FinalScore:     a.FinalScore,
			FinalSource:    src,
			FinalFeedback:  a.FinalFeedback,
			Options:        opts,
			StudentOptions: studentOpts,
			Metadata:       metadata,
		}
	}

	resp := dto.AttemptReviewResponse{
		AttemptID:  attempt.ID.String(),
		Status:     string(attempt.Status),
		Score:      attempt.Grade,
		StartedAt:  attempt.StartedAt.Format("2006-01-02T15:04:05Z"),
		FinishedAt: finishedAt,
		Answers:    reviewAnswers,
	}
	if attempt.UserID != uuid.Nil {
		resp.UserID = attempt.UserID.String()
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetStudentDashboard(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	dashboard, err := h.service.GetStudentDashboard(c.Request.Context(), userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	grades := make([]dto.RecentGradeItemDTO, len(dashboard.RecentGrades))
	for i, g := range dashboard.RecentGrades {
		var finishedAt *string
		if g.FinishedAt != nil {
			s := g.FinishedAt.UTC().Format(time.RFC3339)
			finishedAt = &s
		}
		grades[i] = dto.RecentGradeItemDTO{
			SessionID:      g.SessionID.String(),
			QuizTemplateID: g.QuizTemplateID.String(),
			QuizTitle:      g.QuizTitle,
			AttemptID:      g.AttemptID.String(),
			Score:          g.Score,
			Status:         string(g.Status),
			FinishedAt:     finishedAt,
		}
	}

	active := make([]dto.ActiveTestItemDTO, len(dashboard.ActiveTests))
	for i, t := range dashboard.ActiveTests {
		var sessionStartedAt, sessionFinishedAt *string
		if t.SessionStartedAt != nil {
			s := t.SessionStartedAt.UTC().Format(time.RFC3339)
			sessionStartedAt = &s
		}
		if t.SessionFinishedAt != nil {
			s := t.SessionFinishedAt.UTC().Format(time.RFC3339)
			sessionFinishedAt = &s
		}
		var attemptID *string
		if t.AttemptID != nil {
			s := t.AttemptID.String()
			attemptID = &s
		}
		var attemptStatus *string
		if t.AttemptStatus != nil {
			s := string(*t.AttemptStatus)
			attemptStatus = &s
		}
		active[i] = dto.ActiveTestItemDTO{
			SessionID:         t.SessionID.String(),
			QuizTemplateID:    t.QuizTemplateID.String(),
			QuizTitle:         t.QuizTitle,
			TotalTimeLimitSec: t.TotalTimeLimitSec,
			SessionStartedAt:  sessionStartedAt,
			SessionFinishedAt: sessionFinishedAt,
			AttemptID:         attemptID,
			AttemptStatus:     attemptStatus,
		}
	}

	c.JSON(http.StatusOK, dto.StudentDashboardResponse{
		RecentGrades: grades,
		ActiveTests:  active,
	})
}

func (h *Handler) ListAwaitingReview(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	sessions, err := h.service.ListAwaitingReview(c.Request.Context(), userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	items := make([]dto.AwaitingReviewSessionItem, len(sessions))
	for i, s := range sessions {
		items[i] = dto.AwaitingReviewSessionItem{
			SessionID:      s.SessionID.String(),
			QuizTemplateID: s.QuizTemplateID.String(),
			QuizTitle:      s.QuizTitle,
			GradingCount:   s.GradingCount,
			GradedCount:    s.GradedCount,
			CompletedCount: s.CompletedCount,
		}
	}

	c.JSON(http.StatusOK, dto.AwaitingReviewResponse{Sessions: items})
}

func (h *Handler) GradeAttempt(c *gin.Context) {
	teacherID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	attemptID, err := uuid.Parse(c.Param("attempt_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	var req dto.GradeAttemptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	grades := make([]domain.GradeItem, len(req.Grades))
	for i, g := range req.Grades {
		submissionID, err := uuid.Parse(g.SubmissionID)
		if err != nil {
			httpx.WriteError(c, apperr.ErrBadID)
			return
		}
		grades[i] = domain.GradeItem{
			SubmissionID: submissionID,
			Score:        g.Score,
			Feedback:     g.Feedback,
		}
	}

	if err := h.service.GradeAttempt(c.Request.Context(), attemptID, teacherID, grades); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) PublishSession(c *gin.Context) {
	teacherID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	if err := h.service.PublishSession(c.Request.Context(), sessionID, teacherID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}
