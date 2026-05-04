package quiz

import "riddler/internal/infra/db"

type Service struct {
	quizzes   quizRepository
	attempts  attemptAccessor
	sessions  sessionService
	tasks     taskScheduler
	txManager *db.TxManager
}

func NewService(quizzes quizRepository, attempts attemptAccessor, sessions sessionService, txManager *db.TxManager, tasks taskScheduler) *Service {
	return &Service{quizzes: quizzes, attempts: attempts, sessions: sessions, tasks: tasks, txManager: txManager}
}
