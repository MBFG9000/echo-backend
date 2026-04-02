package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Pseudonym string    `json:"pseudonym" gorm:"uniqueIndex;not null"`
	TokenHash string    `json:"-" gorm:"not null"`
	IsAdmin   bool      `json:"isAdmin" gorm:"not null;default:false"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt" gorm:"not null"`
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateToken(ctx context.Context, id uuid.UUID, tokenHash string, expiresAt time.Time) error
}

type AuthService interface {
	Register(ctx context.Context) (string, string, error)
	Refresh(ctx context.Context, token string) (string, error)
}
