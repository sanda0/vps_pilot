package services

import (
	"context"
	"fmt"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/dto"
	"github.com/sanda0/vps_pilot/internal/utils"
)

type UserService interface {
	Login(form dto.UserLoginDto) (*dto.UserLoginResponseDto, error)
	Profile(id int32) (*db.User, error)
}

type userService struct {
	repo *db.Repo
	ctx  context.Context
}

// Profile implements UserService.
func (u *userService) Profile(id int32) (*db.User, error) {
	user, err := u.repo.Queries.FindUserById(u.ctx, int32(id))
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Login implements UserService.
func (u *userService) Login(form dto.UserLoginDto) (*dto.UserLoginResponseDto, error) {
	user, err := u.repo.Queries.FindUserByEmail(u.ctx, form.Email)
	if err != nil {
		return nil, err
	}

	if err := utils.VerifyPassword(form.Password, user.PasswordHash); err != nil {
		return nil, err

	}

	fmt.Println("user", user)

	response := &dto.UserLoginResponseDto{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
	}

	return response, nil
}

func NewUserService(ctx context.Context, repo *db.Repo) UserService {
	return &userService{
		repo: repo,
		ctx:  ctx,
	}
}
