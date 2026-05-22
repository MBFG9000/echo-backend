package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/echo-app/echo/internal/domain"
	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupPostgres(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "secret",
			"POSTGRES_DB":       "echo_test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(90 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get port: %v", err)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=postgres password=secret dbname=echo_test sslmode=disable TimeZone=UTC", host, port.Port())
	var db *gorm.DB
	for i := 0; i < 10; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("failed to connect database: %v", err)
	}

	if err = db.AutoMigrate(
		&domain.User{},
		&domain.Post{},
		&domain.PostAttachment{},
		&domain.Reply{},
		&domain.ReplyReaction{},
		&domain.Reaction{},
	); err != nil {
		container.Terminate(ctx)
		t.Fatalf("auto migration failed: %v", err)
	}

	return db, func() {
		_ = container.Terminate(ctx)
	}
}

func TestUserRepository_CreateGetUpdate(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()

	repo := NewUser(db)
	user := &domain.User{ID: uuid.New(), Pseudonym: "test-user", TokenHash: "hash", ExpiresAt: time.Now().Add(time.Hour)}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	fetched, err := repo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}
	if fetched.Pseudonym != user.Pseudonym {
		t.Fatalf("expected pseudonym %s got %s", user.Pseudonym, fetched.Pseudonym)
	}

	if err := repo.UpdateToken(context.Background(), user.ID, "newhash", time.Now().Add(2*time.Hour)); err != nil {
		t.Fatalf("update token failed: %v", err)
	}

	fetched2, err := repo.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get user failed: %v", err)
	}
	if fetched2.TokenHash != "newhash" {
		t.Fatalf("expected token hash updated, got %s", fetched2.TokenHash)
	}
}

func TestUserRepository_Conflict(t *testing.T) {
	db, cleanup := setupPostgres(t)
	defer cleanup()

	repo := NewUser(db)
	user1 := &domain.User{ID: uuid.New(), Pseudonym: "duplicate-user", TokenHash: "x", ExpiresAt: time.Now().Add(time.Hour)}
	user2 := &domain.User{ID: uuid.New(), Pseudonym: "duplicate-user", TokenHash: "y", ExpiresAt: time.Now().Add(time.Hour)}

	if err := repo.Create(context.Background(), user1); err != nil {
		t.Fatalf("create user1 failed: %v", err)
	}

	if err := repo.Create(context.Background(), user2); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict got %v", err)
	}
}
