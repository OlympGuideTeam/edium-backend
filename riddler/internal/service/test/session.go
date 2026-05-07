package test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
)

func (s *Service) CreateTestCourseSession(ctx context.Context, authorID, quizTemplateID, moduleID uuid.UUID, p domain.CreateTestCourseSessionParams) (uuid.UUID, error) {
	quiz, err := s.quizzes.GetByID(ctx, quizTemplateID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return uuid.Nil, apperr.ErrQuizNotFound
	}
	if quiz.AuthorID != authorID {
		return uuid.Nil, apperr.ErrQuizForbidden
	}
	if quiz.Source != domain.QuizSourceCourse {
		return uuid.Nil, apperr.ErrQuizForbidden
	}

	params := s.buildParams(quizTemplateID, quiz, p)
	params.MaxScore = quiz.MaxScore

	var sessionID uuid.UUID
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		var innerErr error
		sessionID, innerErr = s.sessions.Create(ctx, params)
		if innerErr != nil {
			return fmt.Errorf("create session: %w", innerErr)
		}

		qtID := quizTemplateID
		payload, _ := json.Marshal(domain.CourseSessionCreatedPayload{
			SessionID:         sessionID,
			ModuleID:          moduleID,
			QuizTemplateID:    &qtID,
			CourseID:          quiz.CourseID,
			Title:             quiz.Title,
			Mode:              string(params.Mode),
			TotalTimeLimitSec: params.TotalTimeLimitSec,
			ShuffleQuestions:  params.ShuffleQuestions,
			StartedAt:         params.StartedAt,
			FinishedAt:        params.FinishedAt,
		})
		return s.tasks.Schedule(ctx, domain.TaskTypeCourseSessionCreatedPublisher, payload)
	})
	if err != nil {
		return uuid.Nil, err
	}
	return sessionID, nil
}

func (s *Service) FinishTestCourseSession(ctx context.Context, teacherID, sessionID uuid.UUID) error {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		return apperr.ErrSessionNotFound
	}
	if session.TeacherID != teacherID {
		return apperr.ErrQuizForbidden
	}
	if session.Source != domain.LiveSourceCourse || session.Mode != domain.SessionModeTest {
		return apperr.ErrQuizForbidden
	}
	if session.Status == domain.SessionStatusFinished {
		return apperr.ErrSessionCompleted
	}
	return s.sessions.UpdateStatus(ctx, sessionID, domain.SessionStatusFinished)
}

func (s *Service) buildParams(quizTemplateID uuid.UUID, quiz *domain.QuizTemplate, p domain.CreateTestCourseSessionParams) domain.CreateSessionParams {
	totalLimit := p.TotalTimeLimitSec
	if totalLimit == nil {
		totalLimit = quiz.DefaultSettings.TotalTimeLimitSec
	}
	if totalLimit == nil {
		v := quiz.QuestionCount * defaultTotalTimeLimitPerQuestion
		totalLimit = &v
	}

	shuffle := p.ShuffleQuestions
	if shuffle == nil {
		shuffle = quiz.DefaultSettings.ShuffleQuestions
	}

	return domain.CreateSessionParams{
		QuizTemplateID:    quizTemplateID,
		Mode:              domain.SessionModeTest,
		Source:            domain.LiveSourceCourse,
		TotalTimeLimitSec: totalLimit,
		ShuffleQuestions:  shuffle,
		Status:            domain.SessionStatusActive,
		StartedAt:         p.StartedAt,
		FinishedAt:        p.FinishedAt,
	}
}
