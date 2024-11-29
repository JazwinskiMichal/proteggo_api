package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"proteggo_api/tools"
	"proteggo_api/types"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"
	"firebase.google.com/go/messaging"
)

func SendNotificationToClient(context context.Context, client *messaging.Client, db *firestore.Client, logger *logging.Logger, data types.NotificationMessage) error {
	// Get registration token from the firestore
	registrationToken, err := tools.GetFirestoreDocument(context, db, types.FIREBASE_MESSAGING_TOKEN_COLLECTION, types.FIREBASE_MESSAGING_TOKEN_DOCUMENT)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error getting registration token from firestore",
			Labels:   map[string]string{"error": err.Error()},
		})
		return err
	}

	tokenStr, ok := registrationToken["token"].(string)
	if !ok || tokenStr == "" {
		errorMsg := "registration token is empty or invalid"
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error sending message to client",
			Labels:   map[string]string{"error": errorMsg},
		})
		return errors.New(errorMsg)
	}

	// Convert the data struct to a JSON string
	dataJson, err := json.Marshal(data)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error converting data to JSON",
			Labels:   map[string]string{"error": err.Error()},
		})
		return fmt.Errorf("error converting data to JSON: %w", err)
	}

	message := &messaging.Message{
		Data:  map[string]string{"data": string(dataJson)},
		Token: tokenStr,
	}

	// Send a message to the device corresponding to the provided registration token.
	_, err = client.Send(context, message)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error sending message to client",
			Labels:   map[string]string{"error": err.Error()},
		})
		return fmt.Errorf("error sending message to client: %w", err)
	}

	return nil
}
