package quiz

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

const (
	defaultLiveQuestionTimeLimitSec      = 30
	defaultTestTotalTimeLimitPerQuestion = 60
)

type courseSessionCreatedPayload struct {
	SessionID uuid.UUID `json:"session_id"`
	ModuleID  uuid.UUID `json:"module_id"`
}

func (s *Service) CreateTestCourseSession(ctx context.Context, quizTemplateID, moduleID uuid.UUID, p domain.CreateTestCourseSessionParams) (uuid.UUID, error) {
	quiz, err := s.quizzes.GetByID(ctx, quizTemplateID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return uuid.Nil, fmt.Errorf("quiz not found")
	}

	var sessionID uuid.UUID
	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		var innerErr error
		sessionID, innerErr = s.sessions.Create(ctx, s.buildTestCourseSessionParams(quizTemplateID, quiz, p))
		if innerErr != nil {
			return fmt.Errorf("create session: %w", innerErr)
		}
		payload, _ := json.Marshal(courseSessionCreatedPayload{SessionID: sessionID, ModuleID: moduleID})
		if innerErr = s.tasks.Schedule(ctx, domain.TaskTypeCourseSessionCreatedPublisher, payload); innerErr != nil {
			return fmt.Errorf("schedule course_session.created: %w", innerErr)
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return sessionID, nil
}

func (s *Service) CreateLiveCourseSession(ctx context.Context, quizTemplateID, moduleID uuid.UUID, p domain.CreateLiveCourseSessionParams) (uuid.UUID, error) {
	quiz, err := s.quizzes.GetByID(ctx, quizTemplateID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return uuid.Nil, fmt.Errorf("quiz not found")
	}

	var sessionID uuid.UUID
	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		var innerErr error
		sessionID, innerErr = s.sessions.Create(ctx, s.buildLiveCourseSessionParams(quizTemplateID, quiz, p))
		if innerErr != nil {
			return fmt.Errorf("create session: %w", innerErr)
		}
		payload, _ := json.Marshal(courseSessionCreatedPayload{SessionID: sessionID, ModuleID: moduleID})
		if innerErr = s.tasks.Schedule(ctx, domain.TaskTypeCourseSessionCreatedPublisher, payload); innerErr != nil {
			return fmt.Errorf("schedule course_session.created: %w", innerErr)
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return sessionID, nil
}

func (s *Service) createLibrarySession(ctx context.Context, quiz *domain.QuizTemplate) error {
	sessionID, err := s.sessions.Create(ctx, s.buildLibrarySessionParams(quiz))
	if err != nil {
		return fmt.Errorf("create library session: %w", err)
	}
	if err := s.quizzes.SetLibrarySession(ctx, quiz.ID, sessionID); err != nil {
		return fmt.Errorf("set library session: %w", err)
	}
	return nil
}

func (s *Service) buildTestCourseSessionParams(quizTemplateID uuid.UUID, quiz *domain.QuizTemplate, p domain.CreateTestCourseSessionParams) domain.CreateSessionParams {
	totalLimit := p.TotalTimeLimitSec
	if totalLimit == nil {
		totalLimit = quiz.DefaultSettings.TotalTimeLimitSec
	}
	if totalLimit == nil {
		v := quiz.QuestionCount * defaultTestTotalTimeLimitPerQuestion
		totalLimit = &v
	}

	shuffle := p.ShuffleQuestions
	if shuffle == nil {
		shuffle = quiz.DefaultSettings.ShuffleQuestions
	}

	return domain.CreateSessionParams{
		QuizTemplateID:    quizTemplateID,
		Mode:              domain.SessionModeTest,
		TotalTimeLimitSec: totalLimit,
		ShuffleQuestions:  shuffle,
		Status:            domain.SessionStatusDraft,
		StartedAt:         p.StartedAt,
		FinishedAt:        p.FinishedAt,
	}
}

func (s *Service) buildLiveCourseSessionParams(quizTemplateID uuid.UUID, quiz *domain.QuizTemplate, p domain.CreateLiveCourseSessionParams) domain.CreateSessionParams {
	questionLimit := p.QuestionTimeLimitSec
	if questionLimit == nil {
		questionLimit = quiz.DefaultSettings.QuestionTimeLimitSec
	}
	if questionLimit == nil {
		v := defaultLiveQuestionTimeLimitSec
		questionLimit = &v
	}

	return domain.CreateSessionParams{
		QuizTemplateID:       quizTemplateID,
		Mode:                 domain.SessionModeLive,
		QuestionTimeLimitSec: questionLimit,
		Status:               domain.SessionStatusDraft,
	}
}

func (s *Service) buildLibrarySessionParams(quiz *domain.QuizTemplate) domain.CreateSessionParams {
	totalLimit := quiz.DefaultSettings.TotalTimeLimitSec
	if totalLimit == nil {
		v := quiz.QuestionCount * defaultTestTotalTimeLimitPerQuestion
		totalLimit = &v
	}

	return domain.CreateSessionParams{
		QuizTemplateID:    quiz.ID,
		Mode:              domain.SessionModeTest,
		TotalTimeLimitSec: totalLimit,
		ShuffleQuestions:  quiz.DefaultSettings.ShuffleQuestions,
		Status:            domain.SessionStatusActive,
	}
}
