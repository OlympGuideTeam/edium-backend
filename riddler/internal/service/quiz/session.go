package quiz

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
)

const (
	defaultLiveQuestionTimeLimitSec      = 30
	defaultTestTotalTimeLimitPerQuestion = 60
)

type courseSessionCreatedPayload struct {
	SessionID            uuid.UUID  `json:"session_id"`
	ModuleID             uuid.UUID  `json:"module_id"`
	QuizTemplateID       *uuid.UUID `json:"quiz_template_id,omitempty"`
	CourseID             *uuid.UUID `json:"course_id,omitempty"`
	Title                string     `json:"title"`
	Mode                 string     `json:"mode"`
	TotalTimeLimitSec    *int       `json:"total_time_limit_sec,omitempty"`
	QuestionTimeLimitSec *int       `json:"question_time_limit_sec,omitempty"`
	ShuffleQuestions     *bool      `json:"shuffle_questions,omitempty"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	FinishedAt           *time.Time `json:"finished_at,omitempty"`
}

func (s *Service) CreateTestCourseSession(ctx context.Context, quizTemplateID, moduleID uuid.UUID, p domain.CreateTestCourseSessionParams) (uuid.UUID, error) {
	quiz, err := s.quizzes.GetByID(ctx, quizTemplateID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return uuid.Nil, fmt.Errorf("quiz not found")
	}

	params := s.buildTestCourseSessionParams(quizTemplateID, quiz, p)

	var sessionID uuid.UUID
	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		var innerErr error
		sessionID, innerErr = s.sessions.Create(ctx, params)
		if innerErr != nil {
			return fmt.Errorf("create session: %w", innerErr)
		}
		qtID := quizTemplateID
		eventPayload, _ := json.Marshal(courseSessionCreatedPayload{
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
		if innerErr = s.tasks.Schedule(ctx, domain.TaskTypeCourseSessionCreatedPublisher, eventPayload); innerErr != nil {
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

	params := s.buildLiveCourseSessionParams(quizTemplateID, quiz, p)

	var sessionID uuid.UUID
	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		var innerErr error
		sessionID, innerErr = s.sessions.Create(ctx, params)
		if innerErr != nil {
			return fmt.Errorf("create session: %w", innerErr)
		}
		qtID := quizTemplateID
		eventPayload, _ := json.Marshal(courseSessionCreatedPayload{
			SessionID:            sessionID,
			ModuleID:             moduleID,
			QuizTemplateID:       &qtID,
			CourseID:             quiz.CourseID,
			Title:                quiz.Title,
			Mode:                 string(params.Mode),
			QuestionTimeLimitSec: params.QuestionTimeLimitSec,
		})
		if innerErr = s.tasks.Schedule(ctx, domain.TaskTypeCourseSessionCreatedPublisher, eventPayload); innerErr != nil {
			return fmt.Errorf("schedule course_session.created: %w", innerErr)
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return sessionID, nil
}

func (s *Service) CreateTestCourseSessionInline(ctx context.Context, authorID uuid.UUID, p domain.CreateTestCourseSessionInlineParams) (uuid.UUID, uuid.UUID, error) {
	var quizTemplateID, sessionID uuid.UUID

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
		sessionCreatedPayload, _ := json.Marshal(courseSessionCreatedPayload{
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
		Status:            domain.SessionStatusActive,
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
		Status:               domain.SessionStatusNotStarted,
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
