package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Auth struct {
	secret string
}

func NewAuth(secret string) *Auth {
	return &Auth{secret: secret}
}

func (a *Auth) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if authorization == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
			c.Abort()
			return
		}

		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
			c.Abort()
			return
		}

		token, err := jwt.ParseWithClaims(parts[1], &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, domain.ErrUnauthorized
			}
			return []byte(a.secret), nil
		})
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*jwt.RegisteredClaims)
		if !ok || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
			c.Abort()
			return
		}

		userID, err := strconv.ParseUint(claims.Subject, 10, 64)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
			c.Abort()
			return
		}

		c.Set("userID", uint(userID))
		c.Next()
	}
}
