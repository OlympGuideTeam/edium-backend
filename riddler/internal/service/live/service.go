package live

const defaultQuestionTimeLimitSec = 30

type Service struct {
	quizzes      quizRepository
	sessions     sessionRepository
	attempts     attemptRepository
	liveSession  liveSessionStore
	liveParticip liveParticipantStore
	liveAnswers  liveAnswerStore
	liveTokens   liveTokenStore
	tasks        taskScheduler
	tx           txManager
	notifier     studentNotifier
}

func NewService(
	quizzes quizRepository,
	sessions sessionRepository,
	attempts attemptRepository,
	liveSession liveSessionStore,
	liveParticip liveParticipantStore,
	liveAnswers liveAnswerStore,
	liveTokens liveTokenStore,
	tasks taskScheduler,
	tx txManager,
	notifier studentNotifier,
) *Service {
	return &Service{
		quizzes:      quizzes,
		sessions:     sessions,
		attempts:     attempts,
		liveSession:  liveSession,
		liveParticip: liveParticip,
		liveAnswers:  liveAnswers,
		liveTokens:   liveTokens,
		tasks:        tasks,
		tx:           tx,
		notifier:     notifier,
	}
}
