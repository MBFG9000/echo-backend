package domain

import (
	"context"
	"time"
)

type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Email        string    `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"not null"`
	Pseudonym    string    `json:"pseudonym" gorm:"uniqueIndex;not null"`
	CreatedAt    time.Time `json:"createdAt"`
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uint) (*User, error)
}

type AuthService interface {
	Register(ctx context.Context, email, password string) (string, *User, error)
	Login(ctx context.Context, email, password string) (string, *User, error)
}
