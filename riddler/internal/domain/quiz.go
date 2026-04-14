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
	TotalTimeLimitSec    *int  `json:"total_time_limit_sec"`
	QuestionTimeLimitSec *int  `json:"question_time_limit_sec"`
	ShuffleQuestions     *bool `json:"shuffle_questions"`
}

type QuizTemplate struct {
	ID              uuid.UUID
	AuthorID        uuid.UUID
	Title           string
	Description     *string
	DefaultSettings QuizDefaultSettings
	IsPublic        bool
	IsDraft         bool
	NeedEvaluation  bool
	QuestionCount   int
	CreatedAt       time.Time
	UpdatedAt       time.Time
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
	IsDraft         bool
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
