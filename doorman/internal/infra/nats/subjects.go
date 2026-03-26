package nats

const (
	SubjectOTPRequest  = "herald.otp.requested"
	SubjectOTPSent     = "doorman.otp.sent"
	SubjectUserCreated = "doorman.user.created"
	SubjectUserDeleted = "caesar.user.deleted"

	QueueOTPRequest  = "doorman-otp-consumers"
	QueueUserDeleted = "doorman-user-deleted-consumers"
)
