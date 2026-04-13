package quiz

type Service struct {
	quizzes quizRepository
}

func NewService(quizzes quizRepository) *Service {
	return &Service{quizzes: quizzes}
}
