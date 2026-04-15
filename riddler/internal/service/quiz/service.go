package quiz

import "riddler/internal/infra/db"

type Service struct {
	quizzes   quizRepository
	sessions  sessionService
	tasks     taskScheduler
	txManager *db.TxManager
}

func NewService(quizzes quizRepository, sessions sessionService, txManager *db.TxManager, tasks taskScheduler) *Service {
	return &Service{quizzes: quizzes, sessions: sessions, tasks: tasks, txManager: txManager}
}
