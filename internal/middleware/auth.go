package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type httpError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

type Claims struct {
	UserID    string `json:"user_id"`
	Pseudonym string `json:"pseudonym"`
	IsAdmin   bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

type Auth struct {
	secret string
	redis  *redis.Client
}

func NewAuth(secret string, redisClient *redis.Client) *Auth {
	return &Auth{secret: secret, redis: redisClient}
}

func (a *Auth) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if authorization == "" {
			c.JSON(http.StatusUnauthorized, httpError{Error: "unauthorized", Code: "ERR_UNAUTHORIZED"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, httpError{Error: "unauthorized", Code: "ERR_UNAUTHORIZED"})
			c.Abort()
			return
		}

		token, err := jwt.ParseWithClaims(parts[1], &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, domain.ErrUnauthorized
			}
			return []byte(a.secret), nil
		})
		if err != nil {
			c.JSON(http.StatusUnauthorized, httpError{Error: "unauthorized", Code: "ERR_UNAUTHORIZED"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok || !token.Valid {
			c.JSON(http.StatusUnauthorized, httpError{Error: "unauthorized", Code: "ERR_UNAUTHORIZED"})
			c.Abort()
			return
		}

		if strings.TrimSpace(claims.UserID) == "" || strings.TrimSpace(claims.Pseudonym) == "" {
			c.JSON(http.StatusUnauthorized, httpError{Error: "unauthorized", Code: "ERR_UNAUTHORIZED"})
			c.Abort()
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, httpError{Error: "unauthorized", Code: "ERR_UNAUTHORIZED"})
			c.Abort()
			return
		}

		hash, err := a.redis.Get(c.Request.Context(), a.sessionKey(userID)).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				c.JSON(http.StatusUnauthorized, httpError{Error: "unauthorized", Code: "ERR_UNAUTHORIZED"})
				c.Abort()
				return
			}
			c.JSON(http.StatusInternalServerError, httpError{Error: "internal error", Code: "ERR_INTERNAL"})
			c.Abort()
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(tokenDigest(parts[1]))); err != nil {
			c.JSON(http.StatusUnauthorized, httpError{Error: "unauthorized", Code: "ERR_UNAUTHORIZED"})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Set("user_id", userID)
		c.Set("pseudonym", claims.Pseudonym)
		c.Set("isAdmin", claims.IsAdmin)
		c.Next()
	}
}

func (a *Auth) sessionKey(userID uuid.UUID) string {
	return "session:" + userID.String()
}

func tokenDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
