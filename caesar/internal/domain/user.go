package domain

import "github.com/google/uuid"

type UserStatus string

var (
	UserStatusActive  UserStatus = "active"
	UserStatusBlocked UserStatus = "blocked"
	UserStatusDeleted UserStatus = "deleted"
)

type User struct {
	ID      uuid.UUID
	Name    string
	Surname string
	Phone   string
	Status  UserStatus
}

type UserStatistic struct {
	ClassTeacherCount  int
	StudentCount       int
	CourseTeacherCount int
	CourseStudentCount int
	QuizCountPassed    int
	AvgQuizScore       float64
}
