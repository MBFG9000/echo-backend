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

type refreshRequest struct {
	Token string `json:"token"`
}

func NewAuth(auth domain.AuthService) *Auth {
	return &Auth{auth: auth}
}

func (a *Auth) Register(rg *gin.RouterGroup) {
	rg.POST("/register", a.register)
	rg.POST("/refresh", a.refresh)
}

// @Summary Register anonymous session
// @Tags auth
// @Produce json
// @Success 201 {object} registerResponse
// @Failure 500 {object} errorResponse
// @Router /auth/register [post]
func (a *Auth) register(c *gin.Context) {
	token, pseudonym, err := a.auth.Register(c.Request.Context())
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusCreated, registerResponse{Token: token, Pseudonym: pseudonym})
}

// @Summary Refresh session token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body refreshRequest true "Refresh token"
// @Success 200 {object} refreshResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /auth/refresh [post]
func (a *Auth) refresh(c *gin.Context) {
	token := refreshTokenFromRequest(c)
	if token == "" {
		writeDomainError(c, domain.ErrUnauthorized)
		return
	}

	newToken, err := a.auth.Refresh(c.Request.Context(), token)
	if err != nil {
		writeDomainError(c, err)
		return
	}

	c.JSON(http.StatusOK, refreshResponse{Token: newToken})
}

func refreshTokenFromRequest(c *gin.Context) string {
	var req refreshRequest
	_ = c.ShouldBindJSON(&req)
	if trimmed := strings.TrimSpace(req.Token); trimmed != "" {
		return trimmed
	}

	authorization := c.GetHeader("Authorization")
	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}
