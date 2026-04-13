package quiz

type Handler struct {
	service quizService
}

func NewHandler(service quizService) *Handler {
	return &Handler{service: service}
}
