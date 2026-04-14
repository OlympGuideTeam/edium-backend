package quiz

import "riddler/internal/infra/db"

type Service struct {
	quizzes   quizRepository
	sessions  sessionService
	txManager *db.TxManager
}

func NewService(quizzes quizRepository, sessions sessionService, txManager *db.TxManager) *Service {
	return &Service{quizzes: quizzes, sessions: sessions, txManager: txManager}
}
