package domain

import (
	"time"

	"github.com/google/uuid"
)

type AttemptStatus string

const (
	AttemptStatusInProgress AttemptStatus = "in_progress"
	AttemptStatusGrading    AttemptStatus = "grading"
	AttemptStatusGraded     AttemptStatus = "graded"
	AttemptStatusCompleted  AttemptStatus = "completed"
)

type FinalSource string

const (
	FinalSourceAuto    FinalSource = "auto"
	FinalSourceTeacher FinalSource = "teacher"
	FinalSourceLLM     FinalSource = "llm"
)

type EvaluationStatus string

const (
	EvaluationStatusPending   EvaluationStatus = "pending"
	EvaluationStatusCompleted EvaluationStatus = "completed"
	EvaluationStatusFailed    EvaluationStatus = "failed"
)

type AnswerEvaluation struct {
	ID           uuid.UUID
	SubmissionID uuid.UUID
	AttemptID    uuid.UUID
	QuestionID   uuid.UUID
	Status       EvaluationStatus
	Score        *float64
	Source       FinalSource
	Feedback     *string
}

type Attempt struct {
	ID            uuid.UUID
	SessionID     uuid.UUID
	UserID        uuid.UUID
	Status        AttemptStatus
	Score         *float64
	QuestionOrder []uuid.UUID
	StartedAt     time.Time
	FinishedAt    *time.Time
}

type AnswerSubmission struct {
	ID            uuid.UUID
	AttemptID     uuid.UUID
	QuestionID    uuid.UUID
	AnswerData    map[string]any
	FinalScore    *float64
	FinalSource   *FinalSource
	FinalFeedback *string
}

type AttemptResult struct {
	Attempt
	Answers []AnswerSubmission
}

type AttemptSummary struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Status AttemptStatus
	Score  *float64
}

type AnswerWithQuestion struct {
	SubmissionID  uuid.UUID
	QuestionID    uuid.UUID
	QuestionType  string
	QuestionText  string
	AnswerData    map[string]any
	FinalScore    *float64
	FinalSource   *FinalSource
	FinalFeedback *string
	Options       []AnswerOption
	Metadata      map[string]any
}

type FreeAnswerSubmission struct {
	SubmissionID uuid.UUID
	AttemptID    uuid.UUID
	QuestionID   uuid.UUID
	QuestionText string
	AnswerText   string
}
