package tools

import (
	"net/http"

	"cloud.google.com/go/logging"
	"github.com/gin-gonic/gin"
)

func LogError(logger *logging.Logger, c *gin.Context, err error) {
	logger.Log(logging.Entry{
		Severity: logging.Error,
		Payload:  err.Error(),
		Labels:   map[string]string{"status": "error"},
	})

	c.JSON(http.StatusBadRequest, gin.H{
		"error": err.Error(),
	})
}
