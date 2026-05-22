package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
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
	users          domain.UserRepository
	generator      pseudonym.Generator
	redis          *redis.Client
	secret         string
	ttl            time.Duration
	adminUsername  string
	adminPassword  string
	adminUserID    uuid.UUID
}

func NewAuth(
	users domain.UserRepository,
	generator pseudonym.Generator,
	redisClient *redis.Client,
	secret string,
	ttl time.Duration,
	adminUsername, adminPassword, adminUserIDRaw string,
) *Auth {
	adminUserID, err := uuid.Parse(strings.TrimSpace(adminUserIDRaw))
	if err != nil {
		adminUserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}

	return &Auth{
		users:         users,
		generator:     generator,
		redis:         redisClient,
		secret:        secret,
		ttl:           ttl,
		adminUsername: strings.TrimSpace(adminUsername),
		adminPassword: adminPassword,
		adminUserID:   adminUserID,
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

func (a *Auth) AdminLogin(ctx context.Context, username, password string) (string, error) {
	if a.adminPassword == "" {
		return "", domain.ErrUnauthorized
	}

	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(username)), []byte(a.adminUsername)) != 1 {
		return "", domain.ErrUnauthorized
	}
	if subtle.ConstantTimeCompare([]byte(password), []byte(a.adminPassword)) != 1 {
		return "", domain.ErrUnauthorized
	}

	now := time.Now()
	expiresAt := now.Add(a.ttl)
	pseudonymValue := "echo-admin"

	token, err := a.sign(a.adminUserID, pseudonymValue, true, expiresAt)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(tokenDigest(token)), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash token: %w", err)
	}

	_, err = a.users.GetByID(ctx, a.adminUserID)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		user := &domain.User{
			ID:        a.adminUserID,
			Pseudonym: pseudonymValue,
			TokenHash: string(hash),
			CreatedAt: now,
			ExpiresAt: expiresAt,
			IsAdmin:   true,
		}
		if err := a.users.Create(ctx, user); err != nil {
			return "", fmt.Errorf("create admin user: %w", err)
		}
	case err != nil:
		return "", fmt.Errorf("get admin user: %w", err)
	default:
		if err := a.users.UpdateToken(ctx, a.adminUserID, string(hash), expiresAt); err != nil {
			return "", fmt.Errorf("update admin token: %w", err)
		}
	}

	if err := a.storeSession(ctx, a.adminUserID, string(hash), expiresAt); err != nil {
		return "", fmt.Errorf("store session: %w", err)
	}

	return token, nil
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
