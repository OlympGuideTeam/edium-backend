package nats

const (
	SubjectCompletionRequested = "edium.completion.requested"
	SubjectCompletionCompleted = "charon.completion.completed"

	QueueCompletionRequested = "charon-completion-consumers"

	SubjectQuizGradeRequested = "riddler.quiz.grade.requested"
	SubjectQuizGradeCompleted = "charon.quiz.grade.completed"

	QueueQuizGradeRequested = "charon-quiz-grade-consumers"
)
