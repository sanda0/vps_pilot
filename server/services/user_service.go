package services

import (
	"fmt"

	"github.com/sanda0/vps_pilot/dto"
	"github.com/sanda0/vps_pilot/models"
	"github.com/sanda0/vps_pilot/repositories"
	"github.com/sanda0/vps_pilot/utils"
)

type UserService interface {
	Login(form dto.UserLoginDto) (*dto.UserLoginResponseDto, error)
	Profile(id uint) (*models.User, error)
}

type userService struct {
	userRepo repositories.UserRepo
}

// Profile implements UserService.
func (u *userService) Profile(id uint) (*models.User, error) {
	user, err := u.userRepo.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Login implements UserService.
func (u *userService) Login(form dto.UserLoginDto) (*dto.UserLoginResponseDto, error) {
	user, err := u.userRepo.GetByEmail(form.Email)
	if err != nil {
		return nil, err
	}

	if err := utils.VerifyPassword(form.Password, user.Password); err != nil {
		return nil, err

	}

	fmt.Println("user", user)

	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		return nil, err
	}

	response := &dto.UserLoginResponseDto{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
		Token:    token,
	}

	return response, nil
}

func NewUserService(userRepo repositories.UserRepo) UserService {
	return &userService{
		userRepo: userRepo,
	}
}
