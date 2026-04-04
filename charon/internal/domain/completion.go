package domain

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionOptions struct {
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
}

type CompletionRequest struct {
	RequestID string             `json:"request_id"`
	Service   string             `json:"service"`
	Model     string             `json:"model"`
	Messages  []Message          `json:"messages"`
	Options   *CompletionOptions `json:"options,omitempty"`
}

type CompletionResponse struct {
	RequestID string `json:"request_id"`
	Content   string `json:"content"`
	Model     string `json:"model"`
	Usage     Usage  `json:"usage"`
	Error     string `json:"error,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
