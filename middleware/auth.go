package middleware

import (
	"os"

	"github.com/gin-gonic/gin"
)

// RequireAdminAuth is a middleware that checks for admin authentication
// For development: checks a simple API key from environment variable
// For production: should use proper authentication (JWT, OAuth, etc.)
func RequireAdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get admin API key from environment variable
		adminKey := os.Getenv("ADMIN_API_KEY")

		// If no admin key is set, allow access (development mode)
		if adminKey == "" {
			c.Next()
			return
		}

		// Check Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Check if the provided key matches
		// Expected format: "Bearer <api-key>"
		expectedAuth := "Bearer " + adminKey
		if authHeader != expectedAuth {
			c.JSON(403, gin.H{
				"error": "Invalid admin credentials",
			})
			c.Abort()
			return
		}

		// Authentication successful
		c.Next()
	}
}
