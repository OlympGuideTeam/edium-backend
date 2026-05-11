package quiz

type Service struct {
	quizzes   quizRepository
	attempts  attemptAccessor
	sessions  sessionService
	tasks     taskScheduler
	txManager txRunner
}

func NewService(quizzes quizRepository, attempts attemptAccessor, sessions sessionService, txManager txRunner, tasks taskScheduler) *Service {
	return &Service{quizzes: quizzes, attempts: attempts, sessions: sessions, tasks: tasks, txManager: txManager}
}
