package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/dto"
	"github.com/sanda0/vps_pilot/internal/utils"
)

type UserService interface {
	Login(form dto.UserLoginDto) (*dto.UserLoginResponseDto, error)
	Profile(id int32) (*db.User, error)
	SaveGitHubToken(userID int32, token string) error
	GetGitHubToken(userID int32) (string, error)
	RemoveGitHubToken(userID int32) error
}

type userService struct {
	repo *db.Repo
	ctx  context.Context
}

// Profile implements UserService.
func (u *userService) Profile(id int32) (*db.User, error) {
	user, err := u.repo.Queries.FindUserById(u.ctx, int64(id))
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
		ID:       int32(user.ID),
		Email:    user.Email,
		Username: user.Username,
	}

	return response, nil
}

// SaveGitHubToken implements UserService.
func (u *userService) SaveGitHubToken(userID int32, token string) error {
	return u.repo.Queries.SaveGitHubToken(u.ctx, db.SaveGitHubTokenParams{
		GithubToken: sql.NullString{String: token, Valid: true},
		UpdatedAt:   sql.NullInt64{Int64: time.Now().Unix(), Valid: true},
		ID:          int64(userID),
	})
}

// GetGitHubToken implements UserService.
func (u *userService) GetGitHubToken(userID int32) (string, error) {
	token, err := u.repo.Queries.GetGitHubToken(u.ctx, int64(userID))
	if err != nil {
		return "", err
	}
	if !token.Valid || token.String == "" {
		return "", fmt.Errorf("GitHub token not found")
	}
	return token.String, nil
}

// RemoveGitHubToken implements UserService.
func (u *userService) RemoveGitHubToken(userID int32) error {
	return u.repo.Queries.RemoveGitHubToken(u.ctx, db.RemoveGitHubTokenParams{
		UpdatedAt: sql.NullInt64{Int64: time.Now().Unix(), Valid: true},
		ID:        int64(userID),
	})
}

func NewUserService(ctx context.Context, repo *db.Repo) UserService {
	return &userService{
		repo: repo,
		ctx:  ctx,
	}
}
