package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
)

type Auth struct {
	auth domain.AuthService
}

type registerResponse struct {
	Token     string `json:"token"`
	Pseudonym string `json:"pseudonym"`
}

type refreshResponse struct {
	Token string `json:"token"`
}

func NewAuth(auth domain.AuthService) *Auth {
	return &Auth{auth: auth}
}

func (a *Auth) Register(rg *gin.RouterGroup) {
	rg.POST("/register", a.register)
	rg.POST("/refresh", a.refresh)
}

func (a *Auth) register(c *gin.Context) {
	token, pseudonym, err := a.auth.Register(c.Request.Context())
	if err != nil {
		a.writeAuthError(c, err)
		return
	}

	c.JSON(http.StatusCreated, registerResponse{Token: token, Pseudonym: pseudonym})
}

func (a *Auth) refresh(c *gin.Context) {
	authorization := c.GetHeader("Authorization")
	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
		return
	}

	token, err := a.auth.Refresh(c.Request.Context(), parts[1])
	if err != nil {
		a.writeAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, refreshResponse{Token: token})
}

func (a *Auth) writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": domain.ErrConflict.Error()})
	case errors.Is(err, domain.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"error": domain.ErrUnauthorized.Error()})
	case errors.Is(err, domain.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}
