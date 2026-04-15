package nats

const (
	SubjectUserCreated = "doorman.user.created"
	SubjectUserDeleted = "caesar.user.deleted"

	SubjectQuizTemplateAttached = "riddler.quiz_template.attached"
	SubjectCourseSessionCreated = "riddler.course_session.created"
	SubjectAttemptCreated       = "riddler.attempt.created"
	SubjectAttemptScored        = "riddler.attempt.scored"

	QueueUserCreated          = "caesar-user-created-consumers"
	QueueQuizTemplateAttached = "caesar-quiz-template-attached-consumers"
	QueueCourseSessionCreated = "caesar-course-session-created-consumers"
	QueueAttemptCreated       = "caesar-attempt-created-consumers"
	QueueAttemptScored        = "caesar-attempt-scored-consumers"
)
