package nats

const (
	SubjectUserCreated = "doorman.user.created"
	SubjectUserDeleted = "caesar.user.deleted"

	SubjectQuizTemplateAttached  = "riddler.quiz_template.attached"
	SubjectCourseSessionCreated  = "riddler.course_session.created"
	SubjectCourseSessionDeleted  = "caesar.course_session.deleted"
	SubjectCourseSessionCanceled = "riddler.course_session.canceled"
	SubjectAttemptCreated        = "riddler.attempt.created"
	SubjectAttemptScored         = "riddler.attempt.scored"

	QueueUserCreated           = "caesar-user-created-consumers"
	QueueQuizTemplateAttached  = "caesar-quiz-template-attached-consumers"
	QueueCourseSessionCreated  = "caesar-course-session-created-consumers"
	QueueAttemptCreated        = "caesar-attempt-created-consumers"
	QueueAttemptScored         = "caesar-attempt-scored-consumers"
	QueueCourseSessionCanceled = "caesar-course-session-canceled-consumers"

)
