package handler

import (
	"net/http"

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

type refreshRequest struct {
	Token string `json:"token" binding:"required"`
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
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeValidationError(c, err)
		return
	}

	token, err := a.auth.Refresh(c.Request.Context(), req.Token)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, refreshResponse{Token: token})
}
