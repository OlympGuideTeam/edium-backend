package quiz

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
)

type quizTemplateAttachedPayload struct {
	QuizTemplateID uuid.UUID       `json:"quiz_template_id"`
	CourseID       uuid.UUID       `json:"course_id"`
	Title          string          `json:"title"`
	Payload        json.RawMessage `json:"payload,omitempty"`
}

// courseDraftPayload — содержимое черновика курса, совпадает по форме
// с payload у course_item (caesar). Хранится в course_draft.payload
// и отдаётся клиенту как есть через GET /courses/{id}.
type courseDraftPayload struct {
	Title                string     `json:"title"`
	Mode                 string     `json:"mode"`
	TotalTimeLimitSec    *int       `json:"total_time_limit_sec,omitempty"`
	QuestionTimeLimitSec *int       `json:"question_time_limit_sec,omitempty"`
	ShuffleQuestions     *bool      `json:"shuffle_questions,omitempty"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	FinishedAt           *time.Time `json:"finished_at,omitempty"`
}

func buildCourseDraftPayload(title string, s domain.QuizDefaultSettings) json.RawMessage {
	mode := domain.SessionModeTest
	if s.Mode != nil {
		mode = *s.Mode
	}
	p := courseDraftPayload{
		Title:                title,
		Mode:                 string(mode),
		TotalTimeLimitSec:    s.TotalTimeLimitSec,
		QuestionTimeLimitSec: s.QuestionTimeLimitSec,
		ShuffleQuestions:     s.ShuffleQuestions,
		StartedAt:            s.StartedAt,
		FinishedAt:           s.FinishedAt,
	}
	b, _ := json.Marshal(p)
	return b
}

func (s *Service) CreateQuiz(ctx context.Context, authorID uuid.UUID, title string, description *string, settings domain.QuizDefaultSettings, attachToCourse *uuid.UUID) (uuid.UUID, error) {
	if strings.TrimSpace(title) == "" {
		return uuid.Nil, apperr.ErrQuizEmptyTitle
	}

	source := domain.QuizSourceLibrary
	if attachToCourse != nil {
		source = domain.QuizSourceCourse
	}

	var id uuid.UUID
	err := s.txManager.WithTx(ctx, func(ctx context.Context) error {
		var innerErr error
		id, innerErr = s.quizzes.Create(ctx, authorID, title, description, settings, source, attachToCourse)
		if innerErr != nil {
			return fmt.Errorf("create quiz: %w", innerErr)
		}
		if attachToCourse != nil {
			payload, _ := json.Marshal(quizTemplateAttachedPayload{
				QuizTemplateID: id,
				CourseID:       *attachToCourse,
				Title:          title,
				Payload:        buildCourseDraftPayload(title, settings),
			})
			if err := s.tasks.Schedule(ctx, domain.TaskTypeQuizTemplateAttachedPublisher, payload); err != nil {
				return fmt.Errorf("schedule quiz_template.attached: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (s *Service) GetQuiz(ctx context.Context, id, userID uuid.UUID) (*domain.QuizDetail, error) {
	quiz, err := s.quizzes.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return nil, apperr.ErrQuizNotFound
	}
	if quiz.AuthorID != userID {
		return nil, apperr.ErrQuizForbidden
	}

	questions, err := s.quizzes.GetQuestionsWithOptions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get questions: %w", err)
	}

	return &domain.QuizDetail{QuizTemplate: *quiz, Questions: questions}, nil
}

func (s *Service) GetQuizForStudent(ctx context.Context, id uuid.UUID) (*domain.QuizStudentView, error) {
	quiz, err := s.quizzes.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return nil, apperr.ErrQuizNotFound
	}
	if !quiz.IsPublic {
		return nil, apperr.ErrQuizNotAvailable
	}

	return &domain.QuizStudentView{
		ID:                   quiz.ID,
		Title:                quiz.Title,
		Description:          quiz.Description,
		TotalTimeLimitSec:    quiz.DefaultSettings.TotalTimeLimitSec,
		QuestionTimeLimitSec: quiz.DefaultSettings.QuestionTimeLimitSec,
		QuestionCount:        quiz.QuestionCount,
		LibraryTestSessionID: quiz.LibrarySessionID,
	}, nil
}

func (s *Service) UpdateQuiz(ctx context.Context, id, authorID uuid.UUID, title, description *string) error {
	if title != nil && strings.TrimSpace(*title) == "" {
		return apperr.ErrQuizEmptyTitle
	}

	quiz, err := s.quizzes.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return apperr.ErrQuizNotFound
	}
	if quiz.AuthorID != authorID {
		return apperr.ErrQuizForbidden
	}

	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := s.quizzes.Update(ctx, id, title, description); err != nil {
			return fmt.Errorf("update quiz: %w", err)
		}
		if title != nil && quiz.CourseID != nil {
			payload, _ := json.Marshal(quizTemplateAttachedPayload{
				QuizTemplateID: id,
				CourseID:       *quiz.CourseID,
				Title:          *title,
				Payload:        buildCourseDraftPayload(*title, quiz.DefaultSettings),
			})
			if err := s.tasks.Schedule(ctx, domain.TaskTypeQuizTemplateAttachedPublisher, payload); err != nil {
				return fmt.Errorf("schedule quiz_template.attached: %w", err)
			}
		}
		return nil
	})
}

func (s *Service) PublishQuiz(ctx context.Context, id, authorID uuid.UUID) error {
	quiz, err := s.quizzes.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return apperr.ErrQuizNotFound
	}
	if quiz.AuthorID != authorID {
		return apperr.ErrQuizForbidden
	}

	hasSession, err := s.quizzes.HasSession(ctx, id)
	if err != nil {
		return fmt.Errorf("check session: %w", err)
	}
	if hasSession {
		return apperr.ErrQuizAlreadyPublished
	}

	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := s.quizzes.Publish(ctx, id); err != nil {
			return fmt.Errorf("publish quiz: %w", err)
		}
		if !quiz.NeedEvaluation {
			if err := s.createLibrarySession(ctx, quiz); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) CopyQuiz(ctx context.Context, quizID, authorID uuid.UUID, attachToCourse *uuid.UUID) (uuid.UUID, error) {
	quiz, err := s.quizzes.GetByID(ctx, quizID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("get quiz: %w", err)
	}
	if quiz == nil {
		return uuid.Nil, apperr.ErrQuizNotFound
	}

	source := domain.QuizSourceLibrary
	if attachToCourse != nil {
		source = domain.QuizSourceCourse
	}

	var newID uuid.UUID
	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		var innerErr error
		newID, innerErr = s.quizzes.Copy(ctx, quizID, authorID, source, attachToCourse)
		if innerErr != nil {
			return fmt.Errorf("copy quiz: %w", innerErr)
		}
		if attachToCourse != nil {
			payload, _ := json.Marshal(quizTemplateAttachedPayload{
				QuizTemplateID: newID,
				CourseID:       *attachToCourse,
				Title:          quiz.Title,
				Payload:        buildCourseDraftPayload(quiz.Title, quiz.DefaultSettings),
			})
			if err := s.tasks.Schedule(ctx, domain.TaskTypeQuizTemplateAttachedPublisher, payload); err != nil {
				return fmt.Errorf("schedule quiz_template.attached: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return newID, nil
}

func (s *Service) ListQuizzes(ctx context.Context, role domain.Role) ([]domain.QuizListItem, error) {
	items, err := s.quizzes.ListPublished(ctx, role == domain.RoleStudent)
	if err != nil {
		return nil, fmt.Errorf("list quizzes: %w", err)
	}
	return items, nil
}

func (s *Service) ListMyQuizzes(ctx context.Context, authorID uuid.UUID) ([]domain.QuizListItem, error) {
	items, err := s.quizzes.ListByAuthor(ctx, authorID)
	if err != nil {
		return nil, fmt.Errorf("list my quizzes: %w", err)
	}
	return items, nil
}
