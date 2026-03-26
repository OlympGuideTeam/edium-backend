package domain

import "github.com/google/uuid"

type IdentityStatus string

var (
	IdentityStatusActive  IdentityStatus = "active"
	IdentityStatusBlocked IdentityStatus = "blocked"
	IdentityStatusDeleted IdentityStatus = "deleted"
)

type Identity struct {
	ID     uuid.UUID
	Status IdentityStatus
	Phone  string
}
