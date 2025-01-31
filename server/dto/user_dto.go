package dto

type UserLoginDto struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type UserLoginResponseDto struct {
	ID       uint   `json:"id"`
	Email    string `json:"email"`
	Token    string `json:"token"`
	Username string `json:"username"`
}
