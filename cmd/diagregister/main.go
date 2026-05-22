package main

import (
	"context"
	"fmt"
	"time"

	"github.com/echo-app/echo/internal/config"
	"github.com/echo-app/echo/internal/repository"
	"github.com/echo-app/echo/internal/service"
	"github.com/echo-app/echo/pkg/pseudonym"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("config error:", err)
		return
	}

	db, err := gorm.Open(postgres.Open(cfg.DB.DSN()), &gorm.Config{})
	if err != nil {
		fmt.Println("db error:", err)
		return
	}

	rdb := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Println("redis ping error:", err)
		return
	}

	auth := service.NewAuth(
		repository.NewUser(db),
		pseudonym.NewRandom(time.Now().UnixNano()),
		rdb,
		cfg.JWT.Secret,
		cfg.JWT.TTL,
		cfg.Admin.Username,
		cfg.Admin.Password,
		cfg.Admin.UserID,
	)
	token, pseudonymValue, err := auth.Register(context.Background())
	if err != nil {
		fmt.Printf("register error: %T %v\n", err, err)
		return
	}

	fmt.Println("register ok", len(token), pseudonymValue)
}
