package test

const defaultTotalTimeLimitPerQuestion = 60

type Service struct {
	quizzes  quizRepository
	sessions sessionRepository
	tasks    taskScheduler
	tx       txManager
}

func NewService(
	quizzes quizRepository,
	sessions sessionRepository,
	tasks taskScheduler,
	tx txManager,
) *Service {
	return &Service{
		quizzes:  quizzes,
		sessions: sessions,
		tasks:    tasks,
		tx:       tx,
	}
}
