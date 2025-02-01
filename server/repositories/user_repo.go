package repositories

import (
	"github.com/sanda0/vps_pilot/models"
	"gorm.io/gorm"
)

type UserRepo interface {
	GetByEmail(email string) (*models.User, error)
	GetUserByID(id uint) (*models.User, error)
}

type userRepo struct {
	db *gorm.DB
}

// GetUserByID implements UserRepo.
func (u *userRepo) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	result := u.db.First(&user, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// GetByEmail implements UserRepo.
func (u *userRepo) GetByEmail(email string) (*models.User, error) {
	var user models.User
	result := u.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func NewUserRepo(db *gorm.DB) UserRepo {
	return &userRepo{
		db: db,
	}
}
