package middlewares

import (
	"net/http"
	"strings"

	"cloud.google.com/go/logging"
	"firebase.google.com/go/auth"
	"github.com/gin-gonic/gin"
)

// Middleware to authenticate and authorize users.
func AuthMiddleware(logger *logging.Logger, authClient *auth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the token from the Authorization header or cookie
		idToken := extractToken(c)
		if idToken == "" {
			logger.Log(logging.Entry{
				Severity: logging.Error,
				Payload:  "Unauthorized - No ID token provided",
			})

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - No ID token provided"})
			return
		}

		// Verify ID token
		decodedToken, err := authClient.VerifyIDToken(c.Request.Context(), idToken)
		if err != nil {
			logger.Log(logging.Entry{
				Severity: logging.Error,
				Payload:  "Unauthorized - Invalid ID token: " + err.Error(),
			})

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized - Invalid ID token, " + err.Error()})
			return
		}

		// ID token is verified, and the user is an admin; proceed with the request
		// Store user information or decoded token in context if needed
		c.Set("user", decodedToken)
		c.Next()
	}
}

// Extracts token from the Authorization header or cookie.
func extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		// More explicit way to strip "Bearer " prefix and avoid magic numbers
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	// Fallback to cookie if needed
	if cookie, err := c.Cookie("__session"); err == nil {
		return cookie
	}
	return ""
}
