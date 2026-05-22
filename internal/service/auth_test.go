package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/echo-app/echo/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type userRepoStub struct {
	create      func(ctx context.Context, user *domain.User) error
	getByID     func(ctx context.Context, id uuid.UUID) (*domain.User, error)
	updateToken func(ctx context.Context, id uuid.UUID, tokenHash string, expiresAt time.Time) error
}

func (s *userRepoStub) Create(ctx context.Context, user *domain.User) error {
	if s.create != nil {
		return s.create(ctx, user)
	}
	return nil
}

func (s *userRepoStub) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if s.getByID != nil {
		return s.getByID(ctx, id)
	}
	return nil, domain.ErrNotFound
}

func (s *userRepoStub) UpdateToken(ctx context.Context, id uuid.UUID, tokenHash string, expiresAt time.Time) error {
	if s.updateToken != nil {
		return s.updateToken(ctx, id, tokenHash, expiresAt)
	}
	return nil
}

type generatorStub struct {
	value string
}

func (g generatorStub) Generate() string {
	return g.value
}

func TestAuth_Register(t *testing.T) {
	r, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: r.Addr()})

	var savedUser *domain.User
	stubRepo := &userRepoStub{}
	stubRepo.create = func(ctx context.Context, user *domain.User) error {
		savedUser = user
		return nil
	}

	a := NewAuth(stubRepo, generatorStub{value: "greedy-owl-123"}, redisClient, "secret", time.Hour)

	token, pseudo, err := a.Register(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("expected token")
	}
	if pseudo != "greedy-owl-123" {
		t.Fatalf("expected pseudonym %q, got %q", "greedy-owl-123", pseudo)
	}
	if savedUser == nil {
		t.Fatal("expected user to be created")
	}

	if got, err := r.Get("session:" + savedUser.ID.String()); err != nil || got == "" {
		t.Fatalf("expected stored session in redis, got %q err=%v", got, err)
	}
}

func TestAuth_RegisterConflict(t *testing.T) {
	r, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: r.Addr()})
	conflicts := 0
	stubRepo := &userRepoStub{}
	stubRepo.create = func(ctx context.Context, user *domain.User) error {
		conflicts++
		return domain.ErrConflict
	}

	a := NewAuth(stubRepo, generatorStub{value: "dry-hill-123"}, redisClient, "secret", time.Hour)

	_, _, err = a.Register(context.Background())
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if conflicts != 8 {
		t.Fatalf("expected 8 conflict attempts, got %d", conflicts)
	}
}

func TestAuth_Refresh(t *testing.T) {
	r, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: r.Addr()})
	userID := uuid.New()
	user := &domain.User{ID: userID, Pseudonym: "spark-owl", IsAdmin: false}

	a := NewAuth(&userRepoStub{}, generatorStub{value: "spark-owl"}, redisClient, "secret", time.Hour)

	oldToken, err := a.sign(userID, user.Pseudonym, user.IsAdmin, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	oldHash, err := bcrypt.GenerateFromPassword([]byte(tokenDigest(oldToken)), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	user.TokenHash = string(oldHash)

	if err := redisClient.Set(context.Background(), "session:"+userID.String(), string(oldHash), time.Hour).Err(); err != nil {
		t.Fatal(err)
	}

	updatedTokenHash := ""
	stubRepo := &userRepoStub{}
	stubRepo.getByID = func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
		if id != userID {
			return nil, domain.ErrNotFound
		}
		return user, nil
	}
	stubRepo.updateToken = func(ctx context.Context, id uuid.UUID, tokenHash string, expiresAt time.Time) error {
		if id != userID {
			return domain.ErrNotFound
		}
		updatedTokenHash = tokenHash
		return nil
	}
	a.users = stubRepo

	newToken, err := a.Refresh(context.Background(), oldToken)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if newToken == "" {
		t.Fatal("expected new token")
	}
	if updatedTokenHash == "" {
		t.Fatal("expected UpdateToken to be called")
	}
}

func TestAuth_RefreshInvalidToken(t *testing.T) {
	r, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: r.Addr()})
	a := NewAuth(&userRepoStub{}, generatorStub{value: "x"}, redisClient, "secret", time.Hour)

	_, err = a.Refresh(context.Background(), "not-a-jwt")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}
