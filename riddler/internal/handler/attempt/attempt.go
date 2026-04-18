package attempt

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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
		resp[i] = dto.AttemptSummaryResponse{
			AttemptID: a.ID.String(),
			UserID:    a.UserID.String(),
			Status:    string(a.Status),
			Score:     a.Score,
		}
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetAttemptReview(c *gin.Context) {
	userID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	attemptID, err := uuid.Parse(c.Param("attempt_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	attempt, answers, err := h.service.GetAttemptReview(c.Request.Context(), attemptID, userID)
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
		if len(a.Options) > 0 {
			opts = make([]dto.AnswerOptionTeacherResponse, len(a.Options))
			for j, o := range a.Options {
				opts[j] = dto.AnswerOptionTeacherResponse{
					ID:        o.ID.String(),
					Text:      o.Text,
					IsCorrect: o.IsCorrect,
				}
			}
		}
		reviewAnswers[i] = dto.AnswerReviewResponse{
			SubmissionID:  a.SubmissionID.String(),
			QuestionID:    a.QuestionID.String(),
			QuestionType:  a.QuestionType,
			QuestionText:  a.QuestionText,
			AnswerData:    a.AnswerData,
			FinalScore:    a.FinalScore,
			FinalSource:   src,
			FinalFeedback: a.FinalFeedback,
			Options:       opts,
			Metadata:      a.Metadata,
		}
	}

	c.JSON(http.StatusOK, dto.AttemptReviewResponse{
		AttemptID:  attempt.ID.String(),
		UserID:     attempt.UserID.String(),
		Status:     string(attempt.Status),
		Score:      attempt.Score,
		StartedAt:  attempt.StartedAt.Format("2006-01-02T15:04:05Z"),
		FinishedAt: finishedAt,
		Answers:    reviewAnswers,
	})
}

func (h *Handler) GradeSubmission(c *gin.Context) {
	teacherID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	attemptID, err := uuid.Parse(c.Param("attempt_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	submissionID, err := uuid.Parse(c.Param("submission_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	var req dto.GradeSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, apperr.ErrBadRequest)
		return
	}

	if err := h.service.GradeSubmission(c.Request.Context(), attemptID, submissionID, teacherID, req.Score, req.Feedback); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) CompleteAttempt(c *gin.Context) {
	teacherID, ok := userIDFromCtx(c)
	if !ok {
		return
	}

	attemptID, err := uuid.Parse(c.Param("attempt_id"))
	if err != nil {
		httpx.WriteError(c, apperr.ErrBadID)
		return
	}

	if err := h.service.CompleteAttempt(c.Request.Context(), attemptID, teacherID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusOK)
}
