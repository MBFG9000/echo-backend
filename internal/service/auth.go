package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/echo-app/echo/internal/domain"
	"github.com/echo-app/echo/pkg/pseudonym"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	UserID    string `json:"user_id"`
	Pseudonym string `json:"pseudonym"`
	IsAdmin   bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

type Auth struct {
	users     domain.UserRepository
	generator pseudonym.Generator
	redis     *redis.Client
	secret    string
	ttl       time.Duration
}

func NewAuth(users domain.UserRepository, generator pseudonym.Generator, redisClient *redis.Client, secret string, ttl time.Duration) *Auth {
	return &Auth{
		users:     users,
		generator: generator,
		redis:     redisClient,
		secret:    secret,
		ttl:       ttl,
	}
}

func (a *Auth) Register(ctx context.Context) (string, string, error) {
	userID := uuid.New()
	now := time.Now()
	expiresAt := now.Add(a.ttl)

	var token string
	var tokenHash string
	var pseudonymValue string

	for range 8 {
		pseudonymValue = a.generator.Generate()
		signed, err := a.sign(userID, pseudonymValue, false, expiresAt)
		if err != nil {
			return "", "", fmt.Errorf("sign token: %w", err)
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(tokenDigest(signed)), bcrypt.DefaultCost)
		if err != nil {
			return "", "", fmt.Errorf("hash token: %w", err)
		}

		user := &domain.User{
			ID:        userID,
			Pseudonym: pseudonymValue,
			TokenHash: string(hash),
			CreatedAt: now,
			ExpiresAt: expiresAt,
		}

		if err := a.users.Create(ctx, user); err != nil {
			if errors.Is(err, domain.ErrConflict) {
				continue
			}
			return "", "", fmt.Errorf("create user: %w", err)
		}

		token = signed
		tokenHash = user.TokenHash
		break
	}

	if token == "" {
		return "", "", domain.ErrConflict
	}

	if err := a.storeSession(ctx, userID, tokenHash, expiresAt); err != nil {
		return "", "", fmt.Errorf("store session: %w", err)
	}

	return token, pseudonymValue, nil
}

func (a *Auth) Refresh(ctx context.Context, oldToken string) (string, error) {
	claims, err := a.parse(oldToken)
	if err != nil {
		return "", domain.ErrUnauthorized
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return "", domain.ErrUnauthorized
	}

	sessionHash, err := a.redis.Get(ctx, a.sessionKey(userID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", domain.ErrUnauthorized
		}
		return "", fmt.Errorf("get session: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(sessionHash), []byte(tokenDigest(oldToken))); err != nil {
		return "", domain.ErrUnauthorized
	}

	user, err := a.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", domain.ErrUnauthorized
		}
		return "", fmt.Errorf("get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.TokenHash), []byte(tokenDigest(oldToken))); err != nil {
		return "", domain.ErrUnauthorized
	}

	expiresAt := time.Now().Add(a.ttl)
	newToken, err := a.sign(user.ID, user.Pseudonym, user.IsAdmin, expiresAt)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(tokenDigest(newToken)), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash token: %w", err)
	}

	if err := a.users.UpdateToken(ctx, user.ID, string(newHash), expiresAt); err != nil {
		return "", fmt.Errorf("update token: %w", err)
	}

	if err := a.storeSession(ctx, user.ID, string(newHash), expiresAt); err != nil {
		return "", fmt.Errorf("store session: %w", err)
	}

	return newToken, nil
}

func (a *Auth) sign(userID uuid.UUID, pseudonym string, isAdmin bool, expiresAt time.Time) (string, error) {
	claims := Claims{
		UserID:    userID.String(),
		Pseudonym: pseudonym,
		IsAdmin:   isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(a.secret))
	if err != nil {
		return "", err
	}

	return signed, nil
}

func (a *Auth) parse(token string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrUnauthorized
		}
		return []byte(a.secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !parsed.Valid || strings.TrimSpace(claims.UserID) == "" || strings.TrimSpace(claims.Pseudonym) == "" {
		return nil, domain.ErrUnauthorized
	}

	return claims, nil
}

func (a *Auth) storeSession(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return domain.ErrUnauthorized
	}

	return a.redis.Set(ctx, a.sessionKey(userID), tokenHash, ttl).Err()
}

func (a *Auth) sessionKey(userID uuid.UUID) string {
	return "session:" + userID.String()
}

func tokenDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
