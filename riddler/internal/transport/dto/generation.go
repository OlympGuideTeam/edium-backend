package dto

type GenerateQuestionsRequest struct {
	Text string `json:"text" binding:"required"`
}

type GenerateQuestionsResponse struct {
	JobID string `json:"job_id"`
}
