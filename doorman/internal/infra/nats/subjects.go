package nats

// Subjects и queue groups для взаимодействия с Herald.

const (
	// SubjectOTPRequest — Herald публикует запрос на отправку OTP.
	SubjectOTPRequest = "doorman.otp.send"

	// SubjectOTPSent — Doorman публикует подтверждение отправки OTP.
	SubjectOTPSent = "herald.otp.sent"

	// SubjectUserCreated — Doorman публикует событие о создании нового пользователя.
	SubjectUserCreated = "doorman.user.created"

	// SubjectUserDeleted — Caesar публикует событие об удалении пользователя.
	SubjectUserDeleted = "caesar.user.deleted"

	// QueueOTPRequest — queue group для балансировки между репликами Doorman.
	QueueOTPRequest = "doorman-otp-consumers"

	// QueueUserDeleted — queue group для обработки удалений пользователей.
	QueueUserDeleted = "doorman-user-deleted-consumers"
)
