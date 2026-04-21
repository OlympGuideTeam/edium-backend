package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleTeacher Role = "teacher"
	RoleStudent Role = "student"
)

type QuestionType string

const (
	QuestionTypeSingleChoice    QuestionType = "single_choice"
	QuestionTypeMultipleChoice  QuestionType = "multiple_choice"
	QuestionTypeWithGivenAnswer QuestionType = "with_given_answer"
	QuestionTypeWithFreeAnswer  QuestionType = "with_free_answer"
	QuestionTypeDrag            QuestionType = "drag"
	QuestionTypeConnection      QuestionType = "connection"
)

type QuizDefaultSettings struct {
	Mode                 *SessionMode `json:"mode,omitempty"`
	TotalTimeLimitSec    *int         `json:"total_time_limit_sec,omitempty"`
	QuestionTimeLimitSec *int         `json:"question_time_limit_sec,omitempty"`
	ShuffleQuestions     *bool        `json:"shuffle_questions,omitempty"`
	StartedAt            *time.Time   `json:"started_at,omitempty"`
	FinishedAt           *time.Time   `json:"finished_at,omitempty"`
}

type QuizSource string

const (
	QuizSourceLibrary QuizSource = "library"
	QuizSourceCourse  QuizSource = "course"
)

type QuizTemplate struct {
	ID               uuid.UUID
	AuthorID         uuid.UUID
	Title            string
	Description      *string
	DefaultSettings  QuizDefaultSettings
	IsPublic         bool
	Source           QuizSource
	NeedEvaluation   bool
	QuestionCount    int
	CourseID         *uuid.UUID
	LibrarySessionID *uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Question struct {
	ID             uuid.UUID
	QuizTemplateID uuid.UUID
	Type           QuestionType
	Text           string
	ImageLink      *string
	OrderIndex     int
	Metadata       map[string]any
	MaxScore       int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AnswerOption struct {
	ID         uuid.UUID
	QuestionID uuid.UUID
	Text       string
	IsCorrect  bool
	CreatedAt  time.Time
}

type QuestionWithOptions struct {
	Question
	Options []AnswerOption
}

type QuizDetail struct {
	QuizTemplate
	Questions []QuestionWithOptions
}

type AnswerOptionForStudent struct {
	ID   uuid.UUID
	Text string
}

type QuestionForStudent struct {
	ID        uuid.UUID
	Type      QuestionType
	Text      string
	ImageLink *string
	MaxScore  int
	Options   []AnswerOptionForStudent
	Metadata  map[string]any
}

type QuizStudentView struct {
	ID                   uuid.UUID
	Title                string
	Description          *string
	TotalTimeLimitSec    *int
	QuestionTimeLimitSec *int
	QuestionCount        int
	LibraryTestSessionID *uuid.UUID
}

type QuizListItem struct {
	ID              uuid.UUID
	Title           string
	Description     *string
	DefaultSettings QuizDefaultSettings
	IsPublic        bool
	Source          QuizSource
	NeedEvaluation  bool
	QuestionCount   int
}

type AddOptionParams struct {
	Text      string
	IsCorrect bool
}

type AddQuestionParams struct {
	QuizTemplateID uuid.UUID
	Type           QuestionType
	Text           string
	ImageLink      *string
	Metadata       map[string]any
	MaxScore       int
	Options        []AddOptionParams
}
