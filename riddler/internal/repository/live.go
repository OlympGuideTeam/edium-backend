package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"riddler/internal/domain"
)

const wsTokenTTL = time.Minute

type LiveRepository struct {
	rdb *redis.Client
}

func NewLiveRepository(rdb *redis.Client) *LiveRepository {
	return &LiveRepository{rdb: rdb}
}

type WsTokenPayload struct {
	Role      domain.Role `json:"role"`
	UserID    string      `json:"user_id,omitempty"`
	AttemptID string      `json:"attempt_id,omitempty"`
}

func keyWsToken(sessionID uuid.UUID, token string) string {
	return fmt.Sprintf("live:%s:ws_token:%s", sessionID, token)
}

func keyCode(code string) string               { return fmt.Sprintf("live:code:%s", code) }
func keyMeta(sid uuid.UUID) string             { return fmt.Sprintf("live:%s:meta", sid) }
func keyPhase(sid uuid.UUID) string            { return fmt.Sprintf("live:%s:phase", sid) }
func keyParticipants(sid uuid.UUID) string     { return fmt.Sprintf("live:%s:participants", sid) }
func keyCurrentQIdx(sid uuid.UUID) string      { return fmt.Sprintf("live:%s:current_q_idx", sid) }
func keyCurrentQDeadline(sid uuid.UUID) string { return fmt.Sprintf("live:%s:current_q_deadline", sid) }
func keyAnswers(sid, qid uuid.UUID) string     { return fmt.Sprintf("live:%s:answers:%s", sid, qid) }

type sessionMeta struct {
	AuthorID             string `json:"author_id"`
	QuizTemplateID       string `json:"quiz_template_id"`
	QuizTitle            string `json:"quiz_title"`
	QuestionCount        int    `json:"question_count"`
	Source               string `json:"source"`
	QuestionTimeLimitSec int    `json:"question_time_limit_sec"`
	JoinCode             string `json:"join_code"`
}

const sessionTTL = 24 * time.Hour

func (r *LiveRepository) InitSession(ctx context.Context, sessionID uuid.UUID, quiz *domain.QuizTemplate, timeLimitSec int, source domain.LiveSource, authorID uuid.UUID) (joinCode string, err error) {
	code := fmt.Sprintf("%06d", rand.IntN(1_000_000))
	meta := sessionMeta{
		AuthorID:             authorID.String(),
		QuizTemplateID:       quiz.ID.String(),
		QuizTitle:            quiz.Title,
		QuestionCount:        quiz.QuestionCount,
		Source:               string(source),
		QuestionTimeLimitSec: timeLimitSec,
		JoinCode:             code,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("marshal meta: %w", err)
	}

	pipe := r.rdb.Pipeline()
	pipe.Set(ctx, keyMeta(sessionID), metaJSON, sessionTTL)
	pipe.Set(ctx, keyPhase(sessionID), string(domain.LivePhaseLobby), sessionTTL)
	pipe.Set(ctx, keyCode(code), sessionID.String(), sessionTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("init session: %w", err)
	}

	return code, nil
}

func (r *LiveRepository) ResolveCode(ctx context.Context, code string) (uuid.UUID, error) {
	val, err := r.rdb.Get(ctx, keyCode(code)).Result()
	if err == redis.Nil {
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve code: %w", err)
	}
	id, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse session_id: %w", err)
	}
	return id, nil
}

func (r *LiveRepository) GetSessionMeta(ctx context.Context, sessionID uuid.UUID) (*domain.LiveSessionMeta, error) {
	metaJSON, err := r.rdb.Get(ctx, keyMeta(sessionID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get meta: %w", err)
	}

	var m sessionMeta
	if err := json.Unmarshal(metaJSON, &m); err != nil {
		return nil, fmt.Errorf("unmarshal meta: %w", err)
	}

	phase, err := r.rdb.Get(ctx, keyPhase(sessionID)).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("get phase: %w", err)
	}

	participantsCount, err := r.rdb.HLen(ctx, keyParticipants(sessionID)).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("get participants count: %w", err)
	}

	authorID, _ := uuid.Parse(m.AuthorID)
	quizTemplateID, _ := uuid.Parse(m.QuizTemplateID)
	source := domain.LiveSource(m.Source)

	result := &domain.LiveSessionMeta{
		SessionID:            sessionID,
		AuthorID:             authorID,
		QuizTemplateID:       quizTemplateID,
		QuizTitle:            m.QuizTitle,
		QuestionCount:        m.QuestionCount,
		Source:               source,
		Phase:                domain.LivePhase(phase),
		QuestionTimeLimitSec: m.QuestionTimeLimitSec,
		IsAnonymousAllowed:   source == domain.LiveSourceLibrary,
		ParticipantsCount:    int(participantsCount),
	}
	if m.JoinCode != "" {
		result.JoinCode = &m.JoinCode
	}
	return result, nil
}

func (r *LiveRepository) GetPhase(ctx context.Context, sessionID uuid.UUID) (domain.LivePhase, error) {
	val, err := r.rdb.Get(ctx, keyPhase(sessionID)).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get phase: %w", err)
	}
	return domain.LivePhase(val), nil
}

type participantEntry struct {
	UserID string `json:"user_id,omitempty"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status"`
}

func (r *LiveRepository) IsKicked(ctx context.Context, sessionID, attemptID uuid.UUID) (bool, error) {
	raw, err := r.rdb.HGet(ctx, keyParticipants(sessionID), attemptID.String()).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("hget participant: %w", err)
	}
	var entry participantEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		return false, nil
	}
	return entry.Status == "kicked", nil
}

func (r *LiveRepository) AddParticipant(ctx context.Context, sessionID uuid.UUID, p domain.LiveParticipant) error {
	entry := participantEntry{Status: p.Status}
	if p.UserID != nil {
		entry.UserID = p.UserID.String()
	}
	if p.Name != nil {
		entry.Name = *p.Name
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal participant: %w", err)
	}
	return r.rdb.HSet(ctx, keyParticipants(sessionID), p.AttemptID.String(), data).Err()
}

func (r *LiveRepository) IssueWsToken(ctx context.Context, sessionID uuid.UUID, payload WsTokenPayload) (string, error) {
	token := uuid.New().String()
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal ws_token payload: %w", err)
	}
	if err := r.rdb.Set(ctx, keyWsToken(sessionID, token), data, wsTokenTTL).Err(); err != nil {
		return "", fmt.Errorf("set ws_token: %w", err)
	}
	return token, nil
}

func (r *LiveRepository) ConsumeWsToken(ctx context.Context, sessionID uuid.UUID, token string) (*WsTokenPayload, error) {
	data, err := r.rdb.GetDel(ctx, keyWsToken(sessionID, token)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getdel ws_token: %w", err)
	}
	var payload WsTokenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal ws_token: %w", err)
	}
	return &payload, nil
}

func (r *LiveRepository) SetPhase(ctx context.Context, sessionID uuid.UUID, phase domain.LivePhase) error {
	if err := r.rdb.Set(ctx, keyPhase(sessionID), string(phase), sessionTTL).Err(); err != nil {
		return fmt.Errorf("set phase: %w", err)
	}
	return nil
}

func (r *LiveRepository) GetParticipants(ctx context.Context, sessionID uuid.UUID) ([]domain.LiveParticipant, error) {
	all, err := r.rdb.HGetAll(ctx, keyParticipants(sessionID)).Result()
	if err != nil {
		return nil, fmt.Errorf("hgetall participants: %w", err)
	}
	result := make([]domain.LiveParticipant, 0, len(all))
	for attemptIDStr, raw := range all {
		attemptID, err := uuid.Parse(attemptIDStr)
		if err != nil {
			continue
		}
		var entry participantEntry
		if err := json.Unmarshal([]byte(raw), &entry); err != nil {
			continue
		}
		p := domain.LiveParticipant{AttemptID: attemptID, Status: entry.Status}
		if entry.UserID != "" {
			id, _ := uuid.Parse(entry.UserID)
			p.UserID = &id
		}
		if entry.Name != "" {
			p.Name = &entry.Name
		}
		result = append(result, p)
	}
	return result, nil
}

func (r *LiveRepository) SetParticipantStatus(ctx context.Context, sessionID, attemptID uuid.UUID, status string) error {
	raw, err := r.rdb.HGet(ctx, keyParticipants(sessionID), attemptID.String()).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("hget participant: %w", err)
	}
	var entry participantEntry
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		return fmt.Errorf("unmarshal participant: %w", err)
	}
	entry.Status = status
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal participant: %w", err)
	}
	return r.rdb.HSet(ctx, keyParticipants(sessionID), attemptID.String(), data).Err()
}

type liveAnswer struct {
	AnswerData  map[string]any `json:"answer_data"`
	IsCorrect   bool           `json:"is_correct"`
	Score       float64        `json:"score"`
	TimeTakenMs int64          `json:"time_taken_ms"`
}

func (r *LiveRepository) SaveAnswer(ctx context.Context, sessionID, questionID, attemptID uuid.UUID, ans domain.LiveAnswer) error {
	entry := liveAnswer{
		AnswerData:  ans.AnswerData,
		IsCorrect:   ans.IsCorrect,
		Score:       ans.Score,
		TimeTakenMs: ans.TimeTakenMs,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal answer: %w", err)
	}
	return r.rdb.HSet(ctx, keyAnswers(sessionID, questionID), attemptID.String(), data).Err()
}

func (r *LiveRepository) GetAnswer(ctx context.Context, sessionID, questionID, attemptID uuid.UUID) (*domain.LiveAnswer, error) {
	raw, err := r.rdb.HGet(ctx, keyAnswers(sessionID, questionID), attemptID.String()).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("hget answer: %w", err)
	}
	var entry liveAnswer
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		return nil, fmt.Errorf("unmarshal answer: %w", err)
	}
	return &domain.LiveAnswer{
		AnswerData:  entry.AnswerData,
		IsCorrect:   entry.IsCorrect,
		Score:       entry.Score,
		TimeTakenMs: entry.TimeTakenMs,
	}, nil
}

func (r *LiveRepository) GetAllAnswers(ctx context.Context, sessionID, questionID uuid.UUID) (map[uuid.UUID]domain.LiveAnswer, error) {
	all, err := r.rdb.HGetAll(ctx, keyAnswers(sessionID, questionID)).Result()
	if err != nil {
		return nil, fmt.Errorf("hgetall answers: %w", err)
	}
	result := make(map[uuid.UUID]domain.LiveAnswer, len(all))
	for attemptIDStr, raw := range all {
		attemptID, err := uuid.Parse(attemptIDStr)
		if err != nil {
			continue
		}
		var entry liveAnswer
		if err := json.Unmarshal([]byte(raw), &entry); err != nil {
			continue
		}
		result[attemptID] = domain.LiveAnswer{
			AnswerData:  entry.AnswerData,
			IsCorrect:   entry.IsCorrect,
			Score:       entry.Score,
			TimeTakenMs: entry.TimeTakenMs,
		}
	}
	return result, nil
}

func (r *LiveRepository) GetAnsweredCount(ctx context.Context, sessionID, questionID uuid.UUID) (int, error) {
	n, err := r.rdb.HLen(ctx, keyAnswers(sessionID, questionID)).Result()
	if err != nil {
		return 0, fmt.Errorf("hlen answers: %w", err)
	}
	return int(n), nil
}

func (r *LiveRepository) SetCurrentQuestion(ctx context.Context, sessionID uuid.UUID, idx int, deadline time.Time) error {
	pipe := r.rdb.Pipeline()
	pipe.Set(ctx, keyCurrentQIdx(sessionID), idx, sessionTTL)
	pipe.Set(ctx, keyCurrentQDeadline(sessionID), deadline.UTC().Format(time.RFC3339), sessionTTL)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("set current question: %w", err)
	}
	return nil
}

func (r *LiveRepository) GetCurrentQuestion(ctx context.Context, sessionID uuid.UUID) (int, time.Time, error) {
	pipe := r.rdb.Pipeline()
	idxCmd := pipe.Get(ctx, keyCurrentQIdx(sessionID))
	deadlineCmd := pipe.Get(ctx, keyCurrentQDeadline(sessionID))
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return 0, time.Time{}, fmt.Errorf("get current question: %w", err)
	}

	idx := 0
	if v, err := idxCmd.Int(); err == nil {
		idx = v
	}
	var deadline time.Time
	if v, err := deadlineCmd.Result(); err == nil {
		deadline, _ = time.Parse(time.RFC3339, v)
	}
	return idx, deadline, nil
}

func (r *LiveRepository) DeleteAll(ctx context.Context, sessionID uuid.UUID, code string) error {
	var cursor uint64
	pattern := fmt.Sprintf("live:%s:*", sessionID)
	for {
		keys, next, err := r.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("scan live keys: %w", err)
		}
		if len(keys) > 0 {
			if err := r.rdb.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("del live keys: %w", err)
			}
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	if code != "" {
		_ = r.rdb.Del(ctx, keyCode(code)).Err()
	}
	return nil
}
