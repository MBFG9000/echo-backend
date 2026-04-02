package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/echo-app/echo/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	db *gorm.DB
}

func NewUser(db *gorm.DB) *User {
	return &User{db: db}
}

func (u *User) Create(ctx context.Context, user *domain.User) error {
	err := u.db.WithContext(ctx).Create(user).Error
	if err == nil {
		return nil
	}

	if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return domain.ErrConflict
	}

	return err
}

func (u *User) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	err := u.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if err == nil {
		return &user, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}

	return nil, err
}

func (u *User) UpdateToken(ctx context.Context, id uuid.UUID, tokenHash string, expiresAt time.Time) error {
	result := u.db.WithContext(ctx).Model(&domain.User{}).
		Where("id = ?", id).
		Updates(map[string]any{"token_hash": tokenHash, "expires_at": expiresAt})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}
