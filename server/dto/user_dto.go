package dto

type UserLoginDto struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type UserLoginResponseDto struct {
	ID       int32  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}
