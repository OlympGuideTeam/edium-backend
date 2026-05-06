package nats

const (
	SubjectOTPRequest = "herald.otp.requested"
	SubjectOTPSent    = "doorman.otp.sent"
	SubjectUserLogout = "doorman.user.logout"

	SubjectAttemptScored        = "riddler.attempt.scored"
	SubjectQuizGenerationNotify = "riddler.quiz.generation.completed"
	SubjectCourseSessionNotify  = "caesar.course_session.notify"

	QueueOTPSent              = "herald-otp-sent-consumers"
	QueueUserLogout           = "herald-user-logout-consumers"
	QueueAttemptScored        = "herald-attempt-scored-consumers"
	QueueQuizGenerationNotify = "herald-quiz-generation-notify-consumers"
	QueueCourseSessionNotify  = "herald-course-session-notify-consumers"
)
