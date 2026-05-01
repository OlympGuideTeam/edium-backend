package attempt

import (
	"riddler/internal/domain"
	"riddler/internal/pkg/grading"
)

func gradeAnswer(q domain.QuestionWithOptions, answerData map[string]any) float64 {
	return grading.GradeAnswer(q, answerData)
}

func computeGrade(score float64, maxScore int) float64 {
	return grading.ComputeGrade(score, maxScore)
}
