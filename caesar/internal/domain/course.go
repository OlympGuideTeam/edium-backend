package domain

import "github.com/google/uuid"

type Course struct {
	ID           uuid.UUID
	ClassID      uuid.UUID
	OwnerID      uuid.UUID
	Title        string
	ModuleCount  int
	ElementCount int
}

// CourseListItem — курс для отображения в списке.
type CourseListItem struct {
	Course
	TeacherName string
	IsTeacher   bool
}

// CourseDetail — курс с модулями.
type CourseDetail struct {
	Course
	TeacherName string
	IsTeacher   bool
	Modules     []CourseModule
}

type CourseModule struct {
	ID           uuid.UUID
	CourseID     uuid.UUID
	Title        string
	ElementCount int
}
