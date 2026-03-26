package nats

const (
	SubjectOTPRequest = "doorman.otp.send"
	SubjectOTPSent = "herald.otp.sent"
	SubjectUserCreated = "doorman.user.created"
	SubjectUserDeleted = "caesar.user.deleted"

	QueueOTPRequest = "doorman-otp-consumers"
	QueueUserDeleted = "doorman-user-deleted-consumers"
)
