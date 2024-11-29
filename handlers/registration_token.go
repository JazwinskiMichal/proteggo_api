package handlers

import (
	"net/http"
	"proteggo_api/tools"
	"proteggo_api/types"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"
	"github.com/gin-gonic/gin"
)

// SetMessagingRegistrationToken sets the messaging registration token for a client
func SetMessagingRegistrationToken(logger *logging.Logger, client *firestore.Client) func(c *gin.Context) {
	return func(c *gin.Context) {

		// Get the client ID and the token from the request
		form, _ := c.MultipartForm()
		clientId := form.Value["clientId"][0]
		token := form.Value["token"][0]

		err := tools.SetFirestoreDocument(c, client, types.FIREBASE_MESSAGING_TOKEN_COLLECTION, types.FIREBASE_MESSAGING_TOKEN_DOCUMENT, map[string]interface{}{
			"clientId": clientId,
			"token":    token,
		})

		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	}
}
