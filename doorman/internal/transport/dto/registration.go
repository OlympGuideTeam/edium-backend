package dto

type RegisterRequest struct {
	Phone   string `json:"phone" binding:"required,e164,startswith=+7"`
	Name    string `json:"name" binding:"required"`
	Surname string `json:"surname" binding:"required"`
}
