package middlewares

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"cloud.google.com/go/logging"
	"github.com/gin-gonic/gin"
)

// Validate file type middleware, only allow images with max 5mb per image
func ImageValidationMiddleware(logger *logging.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the multipart form data from the request
		form, err := c.MultipartForm()
		if err != nil {
			logger.Log(logging.Entry{
				Severity: logging.Error,
				Payload:  "Failed to get the multipart form data from the request: " + err.Error(),
			})

			c.JSON(http.StatusBadRequest, gin.H{"error": "No files are received"})
			c.Abort()
			return
		}

		// Get the files from the form data
		files := form.File
		var fileInfos []map[string]interface{}

		if len(files) == 0 {
			logger.Log(logging.Entry{
				Severity: logging.Error,
				Payload:  "No file is received",
			})

			c.JSON(http.StatusBadRequest, gin.H{"error": "No file is received"})
			c.Abort()
			return
		}

		for id, fileHeaders := range files {
			for _, file := range fileHeaders {
				fileInfos, err = validateAndProcessFile(file, id, fileInfos)
				if err != nil {
					logger.Log(logging.Entry{
						Severity: logging.Error,
						Payload:  "Failed to validate and process file: " + err.Error(),
					})

					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					c.Abort()
					return
				}
			}
		}

		c.Set("fileInfos", fileInfos)
		c.Next()
	}
}

func validateAndProcessFile(file *multipart.FileHeader, id string, fileInfos []map[string]interface{}) ([]map[string]interface{}, error) {
	if file.Size > 5*1024*1024 {
		return nil, fmt.Errorf("file too large: maximum size 5MB")
	}

	f, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer f.Close()

	// Allocate a buffer to read only the first 512 bytes to detect content type
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	// Detect content type
	contentType := http.DetectContentType(buf[:n])
	if contentType != "image/jpeg" && contentType != "image/png" {
		return nil, fmt.Errorf("unsupported file type: %v", contentType)
	}

	// Reset file pointer to the beginning of the file for subsequent operations
	if _, err = f.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("error seeking file: %v", err)
	}

	fileInfo := map[string]interface{}{
		"id":           id,
		"file":         f,
		"content_type": contentType,
		"extension":    filepath.Ext(file.Filename),
	}

	fileInfos = append(fileInfos, fileInfo)

	return fileInfos, nil
}
