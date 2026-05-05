package nats

const (
	SubjectQuizTemplateAttached = "riddler.quiz_template.attached"
	SubjectCourseSessionCreated = "riddler.course_session.created"
	SubjectAttemptCreated       = "riddler.attempt.created"
	SubjectAttemptScored        = "riddler.attempt.scored"

	SubjectGenerationRequested = "riddler.generation.requested"
	SubjectGenerationCompleted = "sphinx.generation.completed"

	SubjectCourseSessionDeleted  = "caesar.course_session.deleted"
	SubjectCourseSessionCanceled = "riddler.course_session.canceled"

	SubjectQuizGradeRequested = "riddler.quiz.grade.requested"
	SubjectQuizGradeCompleted = "charon.quiz.grade.completed"
)
