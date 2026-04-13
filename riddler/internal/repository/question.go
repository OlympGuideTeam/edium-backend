package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"riddler/internal/domain"
	"riddler/internal/infra/db"
	"riddler/internal/pkg/apperr"
)

func (r *PgQuizRepository) AddQuestion(ctx context.Context, params domain.AddQuestionParams) (uuid.UUID, int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var orderIndex int
	err = tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(order_index), 0) + 1 FROM question WHERE quiz_template_id = $1`,
		params.QuizTemplateID,
	).Scan(&orderIndex)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("get order index: %w", err)
	}

	var metadataJSON []byte
	if params.Metadata != nil {
		metadataJSON, err = json.Marshal(params.Metadata)
		if err != nil {
			return uuid.Nil, 0, fmt.Errorf("marshal metadata: %w", err)
		}
	}

	var questionID uuid.UUID
	err = tx.QueryRowContext(ctx,
		`INSERT INTO question (quiz_template_id, type, text, image_link, order_index, metadata, max_score)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		params.QuizTemplateID, params.Type, params.Text, params.ImageLink,
		orderIndex, nullableBytes(metadataJSON), params.MaxScore,
	).Scan(&questionID)
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("insert question: %w", err)
	}

	for _, opt := range params.Options {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO answer_option (question_id, text, is_correct) VALUES ($1, $2, $3)`,
			questionID, opt.Text, opt.IsCorrect,
		)
		if err != nil {
			return uuid.Nil, 0, fmt.Errorf("insert answer_option: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return uuid.Nil, 0, fmt.Errorf("commit: %w", err)
	}

	return questionID, orderIndex, nil
}

func (r *PgQuizRepository) DeleteQuestion(ctx context.Context, quizID, questionID uuid.UUID) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	res, err := exec.ExecContext(ctx,
		`DELETE FROM question WHERE id = $1 AND quiz_template_id = $2`,
		questionID, quizID,
	)
	if err != nil {
		return fmt.Errorf("delete question: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperr.ErrQuestionNotFound
	}
	return nil
}

func (r *PgQuizRepository) ReorderQuestions(ctx context.Context, quizID uuid.UUID, questionIDs []uuid.UUID) error {
	ids := make([]string, len(questionIDs))
	for i, id := range questionIDs {
		ids[i] = id.String()
	}

	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE question
		 SET order_index = idx.pos
		 FROM (
		     SELECT u.id, u.pos
		     FROM unnest($1::uuid[]) WITH ORDINALITY AS u(id, pos)
		 ) AS idx
		 WHERE question.id = idx.id AND question.quiz_template_id = $2`,
		"{"+strings.Join(ids, ",")+"}", quizID,
	)
	if err != nil {
		return fmt.Errorf("reorder questions: %w", err)
	}
	return nil
}

func (r *PgQuizRepository) GetQuestionsWithOptions(ctx context.Context, quizID uuid.UUID) ([]domain.QuestionWithOptions, error) {
	exec := db.ExecutorFromContext(ctx, r.db)

	rows, err := exec.QueryContext(ctx,
		`SELECT id, type, text, image_link, order_index, metadata, max_score
		 FROM question
		 WHERE quiz_template_id = $1
		 ORDER BY order_index`,
		quizID,
	)
	if err != nil {
		return nil, fmt.Errorf("query questions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var questions []domain.QuestionWithOptions
	var questionIDs []uuid.UUID

	for rows.Next() {
		var q domain.QuestionWithOptions
		var metadataJSON []byte
		if err := rows.Scan(&q.ID, &q.Type, &q.Text, &q.ImageLink,
			&q.OrderIndex, &metadataJSON, &q.MaxScore); err != nil {
			return nil, fmt.Errorf("scan question: %w", err)
		}
		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &q.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}
		q.QuizTemplateID = quizID
		questions = append(questions, q)
		questionIDs = append(questionIDs, q.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("questions rows: %w", err)
	}
	if len(questions) == 0 {
		return questions, nil
	}

	ids := make([]string, len(questionIDs))
	for i, id := range questionIDs {
		ids[i] = id.String()
	}

	optRows, err := exec.QueryContext(ctx,
		`SELECT id, question_id, text, is_correct
		 FROM answer_option
		 WHERE question_id = ANY($1::uuid[])
		 ORDER BY question_id`,
		"{"+strings.Join(ids, ",")+"}")
	if err != nil {
		return nil, fmt.Errorf("query options: %w", err)
	}
	defer func() { _ = optRows.Close() }()

	optsByQuestion := make(map[uuid.UUID][]domain.AnswerOption)
	for optRows.Next() {
		var o domain.AnswerOption
		if err := optRows.Scan(&o.ID, &o.QuestionID, &o.Text, &o.IsCorrect); err != nil {
			return nil, fmt.Errorf("scan option: %w", err)
		}
		optsByQuestion[o.QuestionID] = append(optsByQuestion[o.QuestionID], o)
	}
	if err := optRows.Err(); err != nil {
		return nil, fmt.Errorf("options rows: %w", err)
	}

	for i := range questions {
		questions[i].Options = optsByQuestion[questions[i].ID]
	}

	return questions, nil
}

func (r *PgQuizRepository) SetNeedEvaluation(ctx context.Context, quizID uuid.UUID, value bool) error {
	exec := db.ExecutorFromContext(ctx, r.db)
	_, err := exec.ExecContext(ctx,
		`UPDATE quiz_template SET need_evaluation = $1 WHERE id = $2`,
		value, quizID,
	)
	return err
}

func (r *PgQuizRepository) HasFreeAnswerQuestions(ctx context.Context, quizID uuid.UUID) (bool, error) {
	exec := db.ExecutorFromContext(ctx, r.db)
	var count int
	err := exec.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM question WHERE quiz_template_id = $1 AND type = 'with_free_answer'`,
		quizID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("count free_answer questions: %w", err)
	}
	return count > 0, nil
}
