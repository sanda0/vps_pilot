package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username    string       `json:"username"`
	Password    string       `json:"-"`
	Email       string       `json:"email"`
	IsVerified  bool         `json:"is_verified"`
	IsSuperuser bool         `json:"is_superuser"`
	Permissions []Permission `json:"permissions"  gorm:"many2many:user_permissions;"`
}

type Permission struct {
	gorm.Model
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}
