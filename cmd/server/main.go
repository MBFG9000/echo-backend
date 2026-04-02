package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/echo-app/echo/internal/config"
	"github.com/echo-app/echo/internal/handler"
	"github.com/echo-app/echo/internal/hub"
	"github.com/echo-app/echo/internal/middleware"
	"github.com/echo-app/echo/internal/repository"
	"github.com/echo-app/echo/internal/service"
	"github.com/echo-app/echo/pkg/pseudonym"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := openDB(cfg.DB)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	redisClient, err := openRedis(cfg.Redis)
	if err != nil {
		return fmt.Errorf("open redis: %w", err)
	}

	userRepo := repository.NewUser(db)
	postRepo := repository.NewPost(db)
	feedRepo := repository.NewFeed(db)
	reportRepo := repository.NewReport(db)
	feedHub := hub.NewHub()

	hubCtx, hubCancel := context.WithCancel(context.Background())
	defer hubCancel()
	go feedHub.Run(hubCtx)

	authService := service.NewAuth(userRepo, pseudonym.NewRandom(time.Now().UnixNano()), redisClient, cfg.JWT.Secret, cfg.JWT.TTL)
	postService := service.NewPost(postRepo, feedHub)
	feedService := service.NewFeed(feedRepo, redisClient)
	reportService := service.NewReport(reportRepo, postRepo, redisClient, cfg.Moderation.AutoHideThreshold)

	authHandler := handler.NewAuth(authService)
	postHandler := handler.NewPost(postService)
	feedHandler := handler.NewFeed(feedService)
	replyHandler := handler.NewReply(postService)
	reactionHandler := handler.NewReaction(postService)
	reportHandler := handler.NewReport(reportService)
	adminHandler := handler.NewAdmin(reportService)
	wsHandler := handler.NewWS(feedHub)
	healthHandler := handler.NewHealth(db, redisClient)

	authMiddleware := middleware.NewAuth(cfg.JWT.Secret, redisClient)
	adminMiddleware := middleware.NewAdmin()
	postCreateLimit := cfg.Server.RateLimitRequests / 6
	if postCreateLimit < 1 {
		postCreateLimit = 1
	}
	rateLimitMiddleware := middleware.NewRateLimit(
		redisClient,
		cfg.Server.RateLimitRequests,
		cfg.Server.RateLimitWindow,
		postCreateLimit,
		cfg.Server.RateLimitWindow,
	)
	corsMiddleware := middleware.NewCORS(cfg.CORS.AllowedOrigins)
	loggerMiddleware := middleware.NewLogger(slog.Default())

	router := gin.New()
	router.Use(
		middleware.NoIP(!cfg.IsProduction()),
		gin.Recovery(),
		middleware.Security(cfg),
		middleware.Tor(cfg),
		rateLimitMiddleware.General(),
		loggerMiddleware.Handler(),
		corsMiddleware.Handler(),
	)
	healthHandler.Register(router)

	authRoutes := router.Group("/auth")
	authHandler.Register(authRoutes)

	publicPosts := router.Group("/posts")
	postHandler.RegisterPublic(publicPosts)
	replyHandler.RegisterPublic(publicPosts)

	privatePosts := router.Group("/posts")
	privatePosts.Use(authMiddleware.Handler())
	postHandler.RegisterPrivate(privatePosts, rateLimitMiddleware.PostCreate())
	replyHandler.RegisterPrivate(privatePosts)
	reactionHandler.Register(privatePosts)
	reportHandler.RegisterPrivate(privatePosts)

	feedRoutes := router.Group("/feed")
	feedHandler.Register(feedRoutes)

	wsRoutes := router.Group("/ws")
	wsHandler.Register(wsRoutes)

	adminRoutes := router.Group("/admin")
	adminRoutes.Use(authMiddleware.Handler(), adminMiddleware.Handler())
	adminHandler.Register(adminRoutes)

	server := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("start server: %w", err)
	case <-signals:
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	hubCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	return nil
}

func openDB(cfg config.DB) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func openRedis(cfg config.Redis) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}
