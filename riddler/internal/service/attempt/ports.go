package attempt

import (
	"context"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

type attemptRepository interface {
	Create(ctx context.Context, sessionID, userID uuid.UUID, questionOrder []uuid.UUID) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Attempt, error)
	UpsertAnswer(ctx context.Context, attemptID, questionID uuid.UUID, answerData map[string]any) (uuid.UUID, error)
	GetAnswers(ctx context.Context, attemptID uuid.UUID) ([]domain.AnswerSubmission, error)
	EvaluateSubmission(ctx context.Context, submissionID uuid.UUID, score float64, source domain.FinalSource, feedback *string) error
	Complete(ctx context.Context, attemptID uuid.UUID, score, grade float64) error
	FindExpiredInProgress(ctx context.Context) ([]domain.Attempt, error)
	SetGrading(ctx context.Context, attemptID uuid.UUID) error
	SetGraded(ctx context.Context, attemptID uuid.UUID, score float64) error
	InsertEvaluation(ctx context.Context, submissionID uuid.UUID, status domain.EvaluationStatus, score *float64, source domain.FinalSource, feedback *string) (uuid.UUID, error)
	CompleteEvaluation(ctx context.Context, evalID uuid.UUID, score float64, feedback *string) error
	FailEvaluation(ctx context.Context, evalID uuid.UUID, feedback string) error
	GetEvaluationByID(ctx context.Context, evalID uuid.UUID) (*domain.AnswerEvaluation, error)
	HasPendingLLMEvaluations(ctx context.Context, attemptID uuid.UUID) (bool, error)
	UpdateSubmissionFinalScore(ctx context.Context, submissionID uuid.UUID, score float64, source domain.FinalSource, feedback *string) error
	SumScores(ctx context.Context, attemptID uuid.UUID) (float64, error)
	FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]domain.AttemptSummary, error)
	GetAnswersWithQuestion(ctx context.Context, attemptID uuid.UUID) ([]domain.AnswerWithQuestion, error)
	GetSubmissionByID(ctx context.Context, id uuid.UUID) (*domain.AnswerSubmission, error)
	UpdateScore(ctx context.Context, attemptID uuid.UUID, score float64) error
	CompleteGraded(ctx context.Context, attemptID uuid.UUID, score, grade float64) error
	GetUserStatistic(ctx context.Context, userID uuid.UUID) (*domain.UserStatistic, error)
}

type sessionReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizSession, error)
}

type quizReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.QuizTemplate, error)
	GetQuestionsWithOptions(ctx context.Context, quizID uuid.UUID) ([]domain.QuestionWithOptions, error)
}

type taskScheduler interface {
	Schedule(ctx context.Context, taskType domain.TaskType, payload []byte) error
}

type sessionGradingRepo interface {
	FindFinishedNeedingGrading(ctx context.Context) ([]domain.QuizSession, error)
	SetGradingSent(ctx context.Context, id uuid.UUID) error
	GetFreeAnswerSubmissionsForSession(ctx context.Context, sessionID uuid.UUID) ([]domain.FreeAnswerSubmission, error)
}

type questionReader interface {
	GetQuestionByID(ctx context.Context, id uuid.UUID) (*domain.Question, error)
}
