package domain

// QuizGradeRequest — запрос от Riddler на оценку свободных ответов.
// Публикуется в NATS subject: riddler.quiz.grade.requested
type QuizGradeRequest struct {
	RequestID string          `json:"request_id"`
	Question  string          `json:"question"`
	Answers   []StudentAnswer `json:"answers"`
}

type StudentAnswer struct {
	StudentID string `json:"student_id"`
	Text      string `json:"text"`
}

// QuizGradeResponse — результат оценки от Charon.
// Публикуется в NATS subject: charon.quiz.grade.completed
type QuizGradeResponse struct {
	RequestID string        `json:"request_id"`
	Grades    []AnswerGrade `json:"grades"`
	Error     string        `json:"error,omitempty"`
}

type AnswerGrade struct {
	StudentID string `json:"student_id"`
	Score     int    `json:"score"` // 0–10
	Comment   string `json:"comment"`
}
