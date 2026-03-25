package nats

// Subjects и queue groups для взаимодействия с Herald.

const (
	// SubjectOTPRequest — Herald публикует запрос на отправку OTP.
	SubjectOTPRequest = "doorman.otp.send"

	// SubjectOTPSent — Doorman публикует подтверждение отправки OTP.
	SubjectOTPSent = "herald.otp.sent"

	// QueueOTPRequest — queue group для балансировки между репликами Doorman.
	QueueOTPRequest = "doorman-otp-consumers"
)
