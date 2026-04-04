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

// ClassDetail — класс с полным составом участников.
type ClassDetail struct {
	Class
	OwnerName string
	IsOwner   bool
	Teachers  []ClassMember
	Students  []ClassMember
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
