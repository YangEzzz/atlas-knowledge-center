package middleware

import (
	"net/http"
	"strings"

	"ty-notebook-api/internal/api/response"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "Authorization header is required")
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "Invalid token format")
			c.Abort()
			return
		}

		// JWT parsing will be implemented with the auth module.
		if parts[1] == "" {
			response.Unauthorized(c, http.StatusText(http.StatusUnauthorized))
			c.Abort()
			return
		}

		c.Next()
	}
}
