package live


const defaultQuestionTimeLimitSec = 30

type Service struct {
	quizzes  quizRepository
	sessions sessionRepository
	attempts attemptRepository
	live     liveRepository
	tasks    taskScheduler
	tx       txManager
}

func NewService(
	quizzes quizRepository,
	sessions sessionRepository,
	attempts attemptRepository,
	live liveRepository,
	tasks taskScheduler,
	tx txManager,
) *Service {
	return &Service{
		quizzes:  quizzes,
		sessions: sessions,
		attempts: attempts,
		live:     live,
		tasks:    tasks,
		tx:       tx,
	}
}
