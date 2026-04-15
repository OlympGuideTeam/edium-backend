package domain

import "github.com/google/uuid"

type ClassMemberRole string

const (
	ClassMemberRoleTeacher ClassMemberRole = "teacher"
	ClassMemberRoleStudent ClassMemberRole = "student"
	ClassMemberRoleOwner   ClassMemberRole = "owner"
)

type Class struct {
	ID           uuid.UUID
	Title        string
	OwnerID      uuid.UUID
	StudentCount int
}

// ClassListItem — класс с данными для отображения в списке.
type ClassListItem struct {
	Class
	OwnerName string
	IsOwner   bool
}

// ClassDetail — класс с полным составом участников и курсами.
type ClassDetail struct {
	Class
	OwnerName string
	IsOwner   bool
	Teachers  []ClassMember
	Students  []ClassMember
	Courses   []CourseListItem
}

type ClassMember struct {
	ClassID uuid.UUID
	UserID  uuid.UUID
	Name    string
	Surname string
	Role    ClassMemberRole
}

type ClassInvitation struct {
	ID      uuid.UUID
	ClassID uuid.UUID
	Role    ClassMemberRole
}

type InvitationDetail struct {
	ClassTitle        string
	ClassStudentCount int
	Role              ClassMemberRole
}
