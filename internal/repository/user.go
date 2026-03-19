package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/echo-app/echo/internal/domain"
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

func (u *User) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := u.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err == nil {
		return &user, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}

	return nil, err
}

func (u *User) GetByID(ctx context.Context, id uint) (*domain.User, error) {
	var user domain.User
	err := u.db.WithContext(ctx).First(&user, id).Error
	if err == nil {
		return &user, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}

	return nil, err
}
