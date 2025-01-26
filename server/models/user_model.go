package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username    string
	Password    string
	Email       string
	IsVerified  bool
	IsSuperuser bool
	Permissions []Permission `gorm:"many2many:user_permissions;"`
}

type Permission struct {
	gorm.Model
	Name        string
	DisplayName string
}
