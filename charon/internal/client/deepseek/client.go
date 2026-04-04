package deepseek

import (
	"bytes"
	"charon/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

type chatRequest struct {
	Model       string           `json:"model"`
	Messages    []domain.Message `json:"messages"`
	Temperature *float64         `json:"temperature,omitempty"`
	MaxTokens   *int             `json:"max_tokens,omitempty"`
	TopP        *float64         `json:"top_p,omitempty"`
}

type chatResponse struct {
	ID      string   `json:"id"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

type choice struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type apiError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (c *Client) Complete(ctx context.Context, req domain.CompletionRequest) (*domain.CompletionResponse, error) {
	body := chatRequest{
		Model:    req.Model,
		Messages: req.Messages,
	}
	if req.Options != nil {
		body.Temperature = req.Options.Temperature
		body.MaxTokens = req.Options.MaxTokens
		body.TopP = req.Options.TopP
	}

	var resp *domain.CompletionResponse
	var err error

	for attempt := 0; attempt < 3; attempt++ {
		resp, err = c.doRequest(ctx, body)
		if err == nil {
			return resp, nil
		}

		if !isRetryable(err) {
			return nil, err
		}

		backoff := time.Duration(1<<uint(attempt)) * time.Second
		slog.WarnContext(ctx, "deepseek request retry",
			"attempt", attempt+1, "backoff", backoff, "err", err)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}

	return nil, fmt.Errorf("deepseek: max retries exceeded: %w", err)
}

func (c *Client) doRequest(ctx context.Context, body chatRequest) (*domain.CompletionResponse, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, &retryableError{err: fmt.Errorf("http request: %w", err)}
	}
	defer func() { _ = httpResp.Body.Close() }()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode == http.StatusTooManyRequests ||
		httpResp.StatusCode >= http.StatusInternalServerError {
		return nil, &retryableError{
			err: fmt.Errorf("deepseek API status %d: %s", httpResp.StatusCode, string(respBody)),
		}
	}

	if httpResp.StatusCode != http.StatusOK {
		var apiErr apiError
		_ = json.Unmarshal(respBody, &apiErr)
		return nil, fmt.Errorf("deepseek API error %d: %s", httpResp.StatusCode, apiErr.Error.Message)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	content := ""
	if len(chatResp.Choices) > 0 {
		content = chatResp.Choices[0].Message.Content
	}

	return &domain.CompletionResponse{
		Content: content,
		Model:   body.Model,
		Usage: domain.Usage{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		},
	}, nil
}

type retryableError struct {
	err error
}

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

func isRetryable(err error) bool {
	_, ok := err.(*retryableError)
	return ok
}
