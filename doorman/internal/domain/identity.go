package domain

type IdentityStatus string

var (
	IdentityStatusActive  IdentityStatus = "active"
	IdentityStatusBlocked IdentityStatus = "blocked"
	IdentityStatusDeleted IdentityStatus = "deleted"
)

type Identity struct {
	ID     string
	Status IdentityStatus
	Phone  string
}
