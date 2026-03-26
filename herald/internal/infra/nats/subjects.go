package nats

const (
	SubjectOTPRequest = "herald.otp.requested"
	SubjectOTPSent    = "doorman.otp.sent"
	QueueOTPSent      = "herald-otp-sent-consumers"
)
