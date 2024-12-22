package models

import (
	"gorm.io/gorm"
	"golang.org/x/crypto/bcrypt"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleUser   Role = "user"
	RoleViewer Role = "viewer"
)

type User struct {
	gorm.Model
	Username     string `gorm:"uniqueIndex;not null" json:"username"`
	Password     string `gorm:"not null" json:"-"`
	Role         Role   `gorm:"not null" json:"role"`
	Email        string `gorm:"uniqueIndex" json:"email"`
	ApiKey       string `gorm:"uniqueIndex" json:"-"`
	IsActive     bool   `gorm:"default:true" json:"is_active"`
}

func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

func (u *User) HasPermission(action string) bool {
	switch u.Role {
	case RoleAdmin:
		return true
	case RoleUser:
		return action != "manage_users" && action != "system_config"
	case RoleViewer:
		return action == "view_containers" || action == "view_alerts"
	default:
		return false
	}
}
