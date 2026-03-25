package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Health struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewHealth(db *gorm.DB, redisClient *redis.Client) *Health {
	return &Health{db: db, redis: redisClient}
}

func (h *Health) Register(r *gin.Engine) {
	r.GET("/health", h.check)
}

func (h *Health) check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "ok"
	redisStatus := "ok"

	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.PingContext(ctx) != nil {
		dbStatus = "error"
	}

	if h.redis.Ping(ctx).Err() != nil {
		redisStatus = "error"
	}

	status := "ok"
	httpStatus := http.StatusOK
	if dbStatus != "ok" || redisStatus != "ok" {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{"status": status, "db": dbStatus, "redis": redisStatus})
}
