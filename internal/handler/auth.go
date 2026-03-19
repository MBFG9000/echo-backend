package handler

import (
	"errors"
	"net/http"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
)

type Auth struct {
	auth domain.AuthService
}

type authRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  domain.User `json:"user"`
}

func NewAuth(auth domain.AuthService) *Auth {
	return &Auth{auth: auth}
}

func (a *Auth) Register(rg *gin.RouterGroup) {
	rg.POST("/register", a.register)
	rg.POST("/login", a.login)
}

func (a *Auth) register(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}

	token, user, err := a.auth.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		a.writeAuthError(c, err)
		return
	}

	c.JSON(http.StatusCreated, authResponse{Token: token, User: *user})
}

func (a *Auth) login(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": domain.ErrInvalidInput.Error()})
		return
	}

	token, user, err := a.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		a.writeAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, authResponse{Token: token, User: *user})
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
