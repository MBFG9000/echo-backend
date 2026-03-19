package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/echo-app/echo/internal/domain"
	"github.com/echo-app/echo/pkg/pseudonym"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	users     domain.UserRepository
	generator pseudonym.Generator
	secret    string
	ttl       time.Duration
}

func NewAuth(users domain.UserRepository, generator pseudonym.Generator, secret string, ttl time.Duration) *Auth {
	return &Auth{
		users:     users,
		generator: generator,
		secret:    secret,
		ttl:       ttl,
	}
}

func (a *Auth) Register(ctx context.Context, email, password string) (string, *domain.User, error) {
	_, err := a.users.GetByEmail(ctx, email)
	if err == nil {
		return "", nil, domain.ErrConflict
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return "", nil, fmt.Errorf("check user by email: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: string(hash),
		Pseudonym:    a.generator.Generate(),
	}

	if err := a.users.Create(ctx, user); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return "", nil, domain.ErrConflict
		}
		return "", nil, fmt.Errorf("create user: %w", err)
	}

	token, err := a.sign(user.ID)
	if err != nil {
		return "", nil, fmt.Errorf("sign token: %w", err)
	}

	return token, user, nil
}

func (a *Auth) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	user, err := a.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", nil, domain.ErrUnauthorized
		}
		return "", nil, fmt.Errorf("get user by email: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, domain.ErrUnauthorized
	}

	token, err := a.sign(user.ID)
	if err != nil {
		return "", nil, fmt.Errorf("sign token: %w", err)
	}

	return token, user, nil
}

func (a *Auth) sign(userID uint) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatUint(uint64(userID), 10),
		ExpiresAt: jwt.NewNumericDate(now.Add(a.ttl)),
		IssuedAt:  jwt.NewNumericDate(now),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(a.secret))
	if err != nil {
		return "", err
	}

	return signed, nil
}
