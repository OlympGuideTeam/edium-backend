package quiz

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
)

const (
	defaultLiveQuestionTimeLimitSec      = 30
	defaultTestTotalTimeLimitPerQuestion = 60
)

func (s *Service) CreateTestCourseSessionInline(ctx context.Context, authorID uuid.UUID, p domain.CreateTestCourseSessionInlineParams) (uuid.UUID, uuid.UUID, error) {
	var quizTemplateID, sessionID uuid.UUID

	var inlineMaxScore int
	for _, q := range p.Questions {
		inlineMaxScore += q.MaxScore
	}

	err := s.txManager.WithTx(ctx, func(ctx context.Context) error {
		var innerErr error
		quizTemplateID, innerErr = s.quizzes.Create(ctx, authorID, p.Title, p.Description, domain.QuizDefaultSettings{}, domain.QuizSourceCourse, &p.CourseID)
		if innerErr != nil {
			return fmt.Errorf("create quiz: %w", innerErr)
		}

		needEvaluation := false
		for i := range p.Questions {
			p.Questions[i].QuizTemplateID = quizTemplateID
			if err := validateQuestion(p.Questions[i].Type, p.Questions[i].Metadata, p.Questions[i].Options); err != nil {
				return err
			}
			if _, _, innerErr = s.quizzes.AddQuestion(ctx, p.Questions[i]); innerErr != nil {
				return fmt.Errorf("add question: %w", innerErr)
			}
			if p.Questions[i].Type == domain.QuestionTypeWithFreeAnswer {
				needEvaluation = true
			}
		}
		if needEvaluation {
			if innerErr = s.quizzes.SetNeedEvaluation(ctx, quizTemplateID, true); innerErr != nil {
				return fmt.Errorf("set need_evaluation: %w", innerErr)
			}
		}

		totalLimit := p.TotalTimeLimitSec
		if totalLimit == nil {
			v := len(p.Questions) * defaultTestTotalTimeLimitPerQuestion
			totalLimit = &v
		}
		sessionParams := domain.CreateSessionParams{
			QuizTemplateID:    quizTemplateID,
			Mode:              domain.SessionModeTest,
			Source:            domain.LiveSourceCourse,
			MaxScore:          inlineMaxScore,
			TotalTimeLimitSec: totalLimit,
			ShuffleQuestions:  p.ShuffleQuestions,
			Status:            domain.SessionStatusActive,
			StartedAt:         p.StartedAt,
			FinishedAt:        p.FinishedAt,
		}
		sessionID, innerErr = s.sessions.Create(ctx, sessionParams)
		if innerErr != nil {
			return fmt.Errorf("create session: %w", innerErr)
		}

		attachedPayload, _ := json.Marshal(quizTemplateAttachedPayload{
			QuizTemplateID: quizTemplateID,
			CourseID:       p.CourseID,
			Title:          p.Title,
			Payload:        buildCourseDraftPayload(p.Title, domain.QuizDefaultSettings{}),
		})
		if innerErr = s.tasks.Schedule(ctx, domain.TaskTypeQuizTemplateAttachedPublisher, attachedPayload); innerErr != nil {
			return fmt.Errorf("schedule quiz_template.attached: %w", innerErr)
		}

		qtID := quizTemplateID
		cID := p.CourseID
		sessionCreatedPayload, _ := json.Marshal(domain.CourseSessionCreatedPayload{
			SessionID:         sessionID,
			ModuleID:          p.ModuleID,
			QuizTemplateID:    &qtID,
			CourseID:          &cID,
			Title:             p.Title,
			Mode:              string(domain.SessionModeTest),
			TotalTimeLimitSec: totalLimit,
			ShuffleQuestions:  p.ShuffleQuestions,
			StartedAt:         p.StartedAt,
			FinishedAt:        p.FinishedAt,
		})
		if innerErr = s.tasks.Schedule(ctx, domain.TaskTypeCourseSessionCreatedPublisher, sessionCreatedPayload); innerErr != nil {
			return fmt.Errorf("schedule course_session.created: %w", innerErr)
		}

		return nil
	})
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return quizTemplateID, sessionID, nil
}

type courseSessionCanceledPayload struct {
	SessionID      uuid.UUID       `json:"session_id"`
	QuizTemplateID uuid.UUID       `json:"quiz_template_id"`
	CourseID       *uuid.UUID      `json:"course_id,omitempty"`
	Title          string          `json:"title"`
	Payload        json.RawMessage `json:"payload,omitempty"`
}

func (s *Service) DeleteCourseSession(ctx context.Context, sessionID, authorID uuid.UUID) error {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		return apperr.ErrSessionNotFound
	}

	quiz, err := s.quizzes.GetByID(ctx, session.QuizTemplateID)
	if err != nil {
		return fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return apperr.ErrQuizNotFound
	}
	if quiz.AuthorID != authorID {
		return apperr.ErrQuizForbidden
	}

	if session.Mode == domain.SessionModeLive && session.Status == domain.SessionStatusRunning {
		return apperr.ErrSessionAlreadyStarted
	}

	hasAttempts, err := s.sessions.HasAttempts(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("check attempts: %w", err)
	}
	if hasAttempts {
		return apperr.ErrSessionHasAttempts
	}

	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		if innerErr := s.sessions.Delete(ctx, sessionID); innerErr != nil {
			return fmt.Errorf("delete session: %w", innerErr)
		}
		eventPayload, _ := json.Marshal(courseSessionCanceledPayload{
			SessionID:      sessionID,
			QuizTemplateID: quiz.ID,
			CourseID:       quiz.CourseID,
			Title:          quiz.Title,
			Payload:        buildCourseDraftPayload(quiz.Title, quiz.DefaultSettings),
		})
		if innerErr := s.tasks.Schedule(ctx, domain.TaskTypeCourseSessionCanceledPublisher, eventPayload); innerErr != nil {
			return fmt.Errorf("schedule course_session.canceled: %w", innerErr)
		}
		return nil
	})
}

func (s *Service) createLibrarySession(ctx context.Context, quiz *domain.QuizTemplate) error {
	params := s.buildLibrarySessionParams(quiz)
	params.MaxScore = quiz.MaxScore
	sessionID, err := s.sessions.Create(ctx, params)
	if err != nil {
		return fmt.Errorf("create library session: %w", err)
	}
	if err := s.quizzes.SetLibrarySession(ctx, quiz.ID, sessionID); err != nil {
		return fmt.Errorf("set library session: %w", err)
	}
	return nil
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
		Source:            domain.LiveSourceLibrary,
		TotalTimeLimitSec: totalLimit,
		ShuffleQuestions:  quiz.DefaultSettings.ShuffleQuestions,
		Status:            domain.SessionStatusActive,
	}
}
