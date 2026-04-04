package dto

type UserProfileResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Surname string `json:"surname"`
}

type UpdateUserRequest struct {
	Name    *string `json:"name"`
	Surname *string `json:"surname"`
}
