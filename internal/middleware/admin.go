package middleware

import (
	"net/http"

	"github.com/echo-app/echo/internal/domain"
	"github.com/gin-gonic/gin"
)

type Admin struct{}

func NewAdmin() *Admin {
	return &Admin{}
}

func (a *Admin) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdminValue, ok := c.Get("isAdmin")
		if !ok {
			c.JSON(http.StatusForbidden, httpError{Error: domain.ErrForbidden.Error(), Code: "ERR_FORBIDDEN"})
			c.Abort()
			return
		}

		isAdmin, ok := isAdminValue.(bool)
		if !ok || !isAdmin {
			c.JSON(http.StatusForbidden, httpError{Error: domain.ErrForbidden.Error(), Code: "ERR_FORBIDDEN"})
			c.Abort()
			return
		}

		c.Next()
	}
}
