package domain

import (
	"encoding/json"

	"github.com/google/uuid"
)

type CourseItemType string

const CourseItemTypeQuiz CourseItemType = "quiz"

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
	Items        []CourseModuleItem
}

type CourseItem struct {
	ID         uuid.UUID
	ModuleID   uuid.UUID
	RefID      uuid.UUID
	Type       CourseItemType
	OrderIndex int
	Settings   json.RawMessage
}

// CourseModuleItem — элемент модуля с прогрессом текущего пользователя.
type CourseModuleItem struct {
	ID         uuid.UUID
	ModuleID   uuid.UUID
	RefID      uuid.UUID
	Type       CourseItemType
	OrderIndex int
	AttemptID  *uuid.UUID
	Score      *float64
}

type CourseUserItemProgress struct {
	ID           uuid.UUID
	CourseItemID uuid.UUID
	UserID       uuid.UUID
	AttemptID    *uuid.UUID
	Score        *float64
}

// CourseSheetItem — элемент курса для колонки ведомости.
type CourseSheetItem struct {
	ID    uuid.UUID
	RefID uuid.UUID
}

// UserItemScore — оценка пользователя за один элемент курса.
type UserItemScore struct {
	UserID uuid.UUID
	ItemID uuid.UUID
	Score  *float64
}

// SheetScore — ячейка ведомости.
type SheetScore struct {
	ItemID uuid.UUID
	Score  *float64
}

// SheetRow — строка ведомости по одному студенту.
type SheetRow struct {
	StudentID   uuid.UUID
	StudentName string
	Scores      []SheetScore
}

// CourseSheet — ведомость курса.
type CourseSheet struct {
	Items    []CourseSheetItem
	Students []SheetRow
}
