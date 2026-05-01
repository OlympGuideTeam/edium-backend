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

func keyCode(code string) string         { return fmt.Sprintf("live:code:%s", code) }
func keyMeta(sid uuid.UUID) string       { return fmt.Sprintf("live:%s:meta", sid) }
func keyPhase(sid uuid.UUID) string      { return fmt.Sprintf("live:%s:phase", sid) }
func keyParticipants(sid uuid.UUID) string { return fmt.Sprintf("live:%s:participants", sid) }

type sessionMeta struct {
	AuthorID             string `json:"author_id"`
	QuizTemplateID       string `json:"quiz_template_id"`
	QuizTitle            string `json:"quiz_title"`
	QuestionCount        int    `json:"question_count"`
	Source               string `json:"source"`
	QuestionTimeLimitSec int    `json:"question_time_limit_sec"`
}

const sessionTTL = 24 * time.Hour

func (r *LiveRepository) InitSession(ctx context.Context, sessionID uuid.UUID, quiz *domain.QuizTemplate, timeLimitSec int, source domain.LiveSource, authorID uuid.UUID) (joinCode string, err error) {
	meta := sessionMeta{
		AuthorID:             authorID.String(),
		QuizTemplateID:       quiz.ID.String(),
		QuizTitle:            quiz.Title,
		QuestionCount:        quiz.QuestionCount,
		Source:               string(source),
		QuestionTimeLimitSec: timeLimitSec,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("marshal meta: %w", err)
	}

	code := fmt.Sprintf("%06d", rand.IntN(1_000_000))

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

	return &domain.LiveSessionMeta{
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
	}, nil
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
