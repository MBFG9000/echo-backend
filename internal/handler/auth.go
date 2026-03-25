package handler

import (
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
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, registerResponse{Token: token, Pseudonym: pseudonym})
}

func (a *Auth) refresh(c *gin.Context) {
	authorization := c.GetHeader("Authorization")
	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		writeDomainError(c, domain.ErrUnauthorized)
		return
	}

	token, err := a.auth.Refresh(c.Request.Context(), parts[1])
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, refreshResponse{Token: token})
}
