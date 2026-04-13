package domain

import (
	"time"

	"github.com/google/uuid"
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
	TotalTimeLimitSec    *int `json:"total_time_limit_sec"`
	QuestionTimeLimitSec *int `json:"question_time_limit_sec"`
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
