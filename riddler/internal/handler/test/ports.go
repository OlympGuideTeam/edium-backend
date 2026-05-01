package test

import (
	"context"

	"github.com/google/uuid"

	"riddler/internal/domain"
)

type testService interface {
	CreateTestCourseSession(ctx context.Context, authorID, quizTemplateID, moduleID uuid.UUID, p domain.CreateTestCourseSessionParams) (uuid.UUID, error)
}
