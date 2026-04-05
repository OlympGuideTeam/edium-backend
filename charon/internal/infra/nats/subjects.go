package nats

const (
	SubjectCompletionRequested = "charon.completion.requested"
	SubjectCompletionCompleted = "charon.completion.completed"

	QueueCompletionRequested = "charon-completion-consumers"

	SubjectQuizGradeRequested = "charon.quiz.grade.requested"
	SubjectQuizGradeCompleted = "charon.quiz.grade.completed"

	QueueQuizGradeRequested = "charon-quiz-grade-consumers"
)
