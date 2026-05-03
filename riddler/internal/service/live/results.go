package live

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/pkg/apperr"
	"riddler/internal/repository"
)

type StudentResults struct {
	MyPosition        int
	TotalParticipants int
	MyScore           float64
	MaxScore          float64
	CorrectCount      int
	QuestionsCount    int
	Top               []domain.LiveParticipantResult
}

type TeacherResults struct {
	Questions   []QuestionResult
	Leaderboard []domain.LiveParticipantResult
}

type QuestionResult struct {
	QuestionID    uuid.UUID
	OrderIndex    int
	Text          string
	Type          string
	CorrectRate   float64
	AnsweredCount int
	CorrectCount  int
	AvgTimesMs    *int
	Distribution  []OptionStat
}

type OptionStat struct {
	OptionID  uuid.UUID
	Count     int
	IsCorrect bool
}

func (s *Service) GetLiveResultsStudent(ctx context.Context, sessionID, attemptID uuid.UUID) (*StudentResults, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if session == nil || session.Mode != domain.SessionModeLive {
		return nil, apperr.ErrSessionNotFound
	}
	if session.Status != domain.SessionStatusFinished {
		return nil, apperr.ErrLiveNotCompleted
	}

	leaderboard, err := s.attempts.GetLiveLeaderboard(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get leaderboard: %w", err)
	}

	var me *domain.LiveParticipantResult
	for i := range leaderboard {
		if leaderboard[i].AttemptID == attemptID {
			me = &leaderboard[i]
			break
		}
	}
	if me == nil {
		return nil, apperr.ErrAttemptNotFound
	}

	questions, err := s.quizzes.GetQuestionsWithOptions(ctx, session.QuizTemplateID)
	if err != nil {
		return nil, fmt.Errorf("get questions: %w", err)
	}

	top := buildTop(leaderboard, attemptID)

	return &StudentResults{
		MyPosition:        me.Position,
		TotalParticipants: len(leaderboard),
		MyScore:           me.Score,
		MaxScore:          float64(session.MaxScore),
		CorrectCount:      me.CorrectCount,
		QuestionsCount:    len(questions),
		Top:               top,
	}, nil
}

func (s *Service) GetLiveResultsTeacher(ctx context.Context, sessionID, callerID uuid.UUID) (*TeacherResults, error) {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if session == nil || session.Mode != domain.SessionModeLive {
		return nil, apperr.ErrSessionNotFound
	}
	if session.Status != domain.SessionStatusFinished {
		return nil, apperr.ErrLiveNotCompleted
	}

	switch session.Source {
	case domain.LiveSourceCourse:
		quiz, err := s.quizzes.GetByID(ctx, session.QuizTemplateID)
		if err != nil {
			return nil, fmt.Errorf("get quiz: %w", err)
		}
		if quiz == nil || quiz.AuthorID != callerID {
			return nil, apperr.ErrQuizForbidden
		}
	case domain.LiveSourceLibrary:
		quiz, err := s.quizzes.GetByID(ctx, session.QuizTemplateID)
		if err != nil {
			return nil, fmt.Errorf("get quiz: %w", err)
		}
		if quiz == nil {
			return nil, apperr.ErrQuizForbidden
		}
		allowed := quiz.AuthorID == callerID
		if session.LiveHostUserID != nil && *session.LiveHostUserID == callerID {
			allowed = true
		}
		if !allowed {
			return nil, apperr.ErrQuizForbidden
		}
	}

	leaderboard, err := s.attempts.GetLiveLeaderboard(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get leaderboard: %w", err)
	}

	questions, err := s.quizzes.GetQuestionsWithOptions(ctx, session.QuizTemplateID)
	if err != nil {
		return nil, fmt.Errorf("get questions: %w", err)
	}

	answers, err := s.attempts.GetLiveSessionAnswers(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session answers: %w", err)
	}

	qResults := buildQuestionResults(questions, answers)

	return &TeacherResults{
		Questions:   qResults,
		Leaderboard: leaderboard,
	}, nil
}

func buildTop(leaderboard []domain.LiveParticipantResult, myAttemptID uuid.UUID) []domain.LiveParticipantResult {
	top := make([]domain.LiveParticipantResult, 0, 4)
	myInTop := false
	for i, p := range leaderboard {
		if i < 3 {
			top = append(top, p)
			if p.AttemptID == myAttemptID {
				myInTop = true
			}
		} else {
			break
		}
	}
	if !myInTop {
		for _, p := range leaderboard {
			if p.AttemptID == myAttemptID {
				top = append(top, p)
				break
			}
		}
	}
	return top
}

func buildQuestionResults(questions []domain.QuestionWithOptions, answers []repository.LiveSessionAnswer) []QuestionResult {
	type questionStats struct {
		answered  int
		correct   int
		totalTime int
		timeCount int
		optCounts map[string]int
	}

	statsMap := make(map[uuid.UUID]*questionStats)
	for _, a := range answers {
		st, ok := statsMap[a.QuestionID]
		if !ok {
			st = &questionStats{optCounts: make(map[string]int)}
			statsMap[a.QuestionID] = st
		}
		st.answered++
		if a.FinalScore > 0 {
			st.correct++
		}
		if a.TimeTakenMs != nil {
			st.totalTime += *a.TimeTakenMs
			st.timeCount++
		}
		if id, ok := a.AnswerData["selected_option_id"].(string); ok {
			st.optCounts[id]++
		}
		if ids, ok := a.AnswerData["selected_option_ids"].([]any); ok {
			for _, v := range ids {
				if id, ok := v.(string); ok {
					st.optCounts[id]++
				}
			}
		}
	}

	results := make([]QuestionResult, 0, len(questions))
	for i := range questions {
		q := &questions[i]
		st := statsMap[q.ID]
		qr := QuestionResult{
			QuestionID: q.ID,
			OrderIndex: q.OrderIndex,
			Text:       q.Text,
			Type:       string(q.Type),
		}
		if st != nil {
			qr.AnsweredCount = st.answered
			qr.CorrectCount = st.correct
			if st.answered > 0 {
				qr.CorrectRate = float64(st.correct) / float64(st.answered)
			}
			if st.timeCount > 0 {
				avg := st.totalTime / st.timeCount
				qr.AvgTimesMs = &avg
			}
			if q.Type == domain.QuestionTypeSingleChoice || q.Type == domain.QuestionTypeMultipleChoice {
				for _, opt := range q.Options {
					qr.Distribution = append(qr.Distribution, OptionStat{
						OptionID:  opt.ID,
						Count:     st.optCounts[opt.ID.String()],
						IsCorrect: opt.IsCorrect,
					})
				}
			}
		}
		results = append(results, qr)
	}
	return results
}
