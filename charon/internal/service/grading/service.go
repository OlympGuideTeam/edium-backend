package grading

import (
	"charon/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// maxCharsPerBatch — порог символов (вопрос + ответы) для одного LLM-запроса.
// Примерно соответствует ~2500 токенам входных данных.
const maxCharsPerBatch = 10000

const systemPrompt = `You are an educational grading assistant. Evaluate student answers to a question.

Grading rules:
- Grade on a scale from 0 to 10 (integers only)
- Consider the relative quality of all answers: if all students answered poorly, the best answer should receive a moderate score, not 10
- If the overall level is low, be lenient — partial knowledge deserves credit relative to peers
- If the overall level is high, be strict — minor mistakes should reduce scores
- Return ONLY a valid JSON array, no markdown fences, no extra text`

type llmGrade struct {
	StudentID string `json:"student_id"`
	Score     int    `json:"score"`
	Comment   string `json:"comment"`
}

type Service struct {
	llm         LLMClient
	usage       UsageLogger
	rateLimiter RateLimiter
	tasks       TaskScheduler
	model       string
}

func NewService(llm LLMClient, usage UsageLogger, rateLimiter RateLimiter, tasks TaskScheduler, model string) *Service {
	return &Service{llm: llm, usage: usage, rateLimiter: rateLimiter, tasks: tasks, model: model}
}

func (s *Service) ProcessGrade(ctx context.Context, req domain.QuizGradeRequest) error {
	allowed, err := s.rateLimiter.Allow(ctx, "riddler")
	if err != nil {
		slog.ErrorContext(ctx, "grading: rate limiter error", "err", err)
	}
	if !allowed {
		return s.scheduleResponse(ctx, domain.QuizGradeResponse{
			RequestID: req.RequestID,
			Error:     "RATE_LIMIT_EXCEEDED",
		})
	}

	batches := s.splitIntoBatches(req.Question, req.Answers)

	var allGrades []domain.AnswerGrade
	for _, batch := range batches {
		grades, err := s.gradeBatch(ctx, req.RequestID, req.Question, batch)
		if err != nil {
			return err
		}
		allGrades = append(allGrades, grades...)
	}

	return s.scheduleResponse(ctx, domain.QuizGradeResponse{
		RequestID: req.RequestID,
		Grades:    allGrades,
	})
}

func (s *Service) gradeBatch(ctx context.Context, requestID, question string, answers []domain.StudentAnswer) ([]domain.AnswerGrade, error) {
	completionReq := domain.CompletionRequest{
		RequestID: requestID,
		Service:   "riddler",
		Model:     s.model,
		Messages: []domain.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: buildUserMessage(question, answers)},
		},
	}

	start := time.Now()
	resp, err := s.llm.Complete(ctx, completionReq)
	duration := time.Since(start)

	record := domain.UsageRecord{
		Timestamp:  start,
		RequestID:  requestID,
		Service:    "riddler",
		Model:      s.model,
		DurationMs: uint32(duration.Milliseconds()),
		Status:     "ok",
	}

	if err != nil {
		record.Status = "error"
		record.Error = err.Error()
		s.logUsage(ctx, record)
		return nil, fmt.Errorf("llm complete: %w", err)
	}

	record.PromptTokens = uint32(resp.Usage.PromptTokens)
	record.CompletionTokens = uint32(resp.Usage.CompletionTokens)
	record.TotalTokens = uint32(resp.Usage.TotalTokens)
	s.logUsage(ctx, record)

	return parseLLMGrades(resp.Content)
}

// splitIntoBatches разбивает ответы на батчи, если суммарный объём текста
// превышает maxCharsPerBatch. Каждый батч оценивается в отдельном LLM-запросе.
func (s *Service) splitIntoBatches(question string, answers []domain.StudentAnswer) [][]domain.StudentAnswer {
	if len(answers) == 0 {
		return nil
	}

	answerBudget := maxCharsPerBatch - len(question)
	if answerBudget < 500 {
		answerBudget = 500
	}

	var batches [][]domain.StudentAnswer
	var current []domain.StudentAnswer
	currentLen := 0

	for _, a := range answers {
		aLen := len(a.StudentID) + len(a.Text) + 4
		if len(current) > 0 && currentLen+aLen > answerBudget {
			batches = append(batches, current)
			current = nil
			currentLen = 0
		}
		current = append(current, a)
		currentLen += aLen
	}
	if len(current) > 0 {
		batches = append(batches, current)
	}
	return batches
}

func buildUserMessage(question string, answers []domain.StudentAnswer) string {
	var sb strings.Builder
	sb.WriteString("Question: ")
	sb.WriteString(question)
	sb.WriteString("\n\nStudent answers:\n")
	for _, a := range answers {
		fmt.Fprintf(&sb, "[%s]: %s\n", a.StudentID, a.Text)
	}
	sb.WriteString("\nReturn a JSON array:\n")
	sb.WriteString(`[{"student_id": "...", "score": 0, "comment": "..."}]`)
	return sb.String()
}

func parseLLMGrades(content string) ([]domain.AnswerGrade, error) {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		start := strings.Index(content, "\n")
		end := strings.LastIndex(content, "```")
		if start != -1 && end > start {
			content = strings.TrimSpace(content[start+1 : end])
		}
	}

	var raw []llmGrade
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON from LLM: %w", err)
	}

	grades := make([]domain.AnswerGrade, len(raw))
	for i, g := range raw {
		grades[i] = domain.AnswerGrade{
			StudentID: g.StudentID,
			Score:     g.Score,
			Comment:   g.Comment,
		}
	}
	return grades, nil
}

func (s *Service) scheduleResponse(ctx context.Context, resp domain.QuizGradeResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	return s.tasks.Schedule(ctx, domain.QuizGradeCompleted, data)
}

func (s *Service) logUsage(ctx context.Context, record domain.UsageRecord) {
	if err := s.usage.LogUsage(ctx, record); err != nil {
		slog.ErrorContext(ctx, "grading: failed to log usage", "err", err)
	}
}
