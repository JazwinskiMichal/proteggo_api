package middlewares

import (
	"net/http"

	"cloud.google.com/go/logging"
	"firebase.google.com/go/auth"
	"github.com/gin-gonic/gin"
)

// Admin authorization middleware
func AdminAuthMiddleware(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Attempt to retrieve the decoded token from the context
		if user, exists := c.Get("user"); exists {
			// Assuming the decoded token is stored as a map
			decodedToken, ok := user.(*auth.Token)
			if !ok {
				logger.Log(logging.Entry{
					Severity: logging.Error,
					Payload:  "Failed to cast the user to *auth.Token",
				})

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Unexpected error occurred"})
				return
			}

			// Check for the admin role within the token's claims
			if admin, ok := decodedToken.Claims["admin"].(bool); ok && admin {
				// User is confirmed as admin, proceed with the request
				c.Next()
				return
			}
		}

		// If the code reaches this point, the user is either not authenticated or not an admin
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "You must be an admin to perform this action",
		})
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "You must be an admin to perform this action"})
	}
}
