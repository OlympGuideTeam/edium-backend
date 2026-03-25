package nats

const (
	SubjectOTPRequest = "doorman.otp.send"
	SubjectOTPSent    = "herald.otp.sent"
	QueueOTPSent      = "herald-otp-sent-consumers"
)
