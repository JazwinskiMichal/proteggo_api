package firebase

import (
	"context"
	"fmt"

	"proteggo_api/types"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/logging"
	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go"
)

func InitFirebaseApp() (*types.FirebaseApp, error) {
	// Initialize logging client
	ctx := context.Background()
	loggingClient, err := logging.NewClient(ctx, types.FIREBASE_PROJECT_ID)
	if err != nil {
		return nil, fmt.Errorf("error initializing logging client: %v", err)
	}
	logger := loggingClient.Logger("go-todo-app")

	logger.Log(logging.Entry{
		Severity: logging.Info,
		Payload:  "Logging client initialized successfully",
		Labels:   map[string]string{"status": "success"},
	})

	// Initialize the app without any options.
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error initializing Firebase app",
			Labels:   map[string]string{"error": err.Error()},
		})
		return nil, err
	} else {
		logger.Log(logging.Entry{
			Severity: logging.Info,
			Payload:  "Firebase app initialized successfully",
			Labels:   map[string]string{"status": "success"},
		})
	}

	// Initialize the Firestore client
	db, err := app.Firestore(ctx)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error initializing Firestore client",
			Labels:   map[string]string{"error": err.Error()},
		})
		return nil, err
	} else {
		logger.Log(logging.Entry{
			Severity: logging.Info,
			Payload:  "Firestore client initialized successfully",
			Labels:   map[string]string{"status": "success"},
		})
	}

	// Initialize the Storage client
	gcs, err := storage.NewClient(ctx)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error initializing Goole Cloud Storage client",
			Labels:   map[string]string{"error": err.Error()},
		})
		return nil, err
	} else {
		logger.Log(logging.Entry{
			Severity: logging.Info,
			Payload:  "Storage client initialized successfully",
			Labels:   map[string]string{"status": "success"},
		})
	}

	// Initialize the Auth client
	auth, err := app.Auth(ctx)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error initializing Auth client",
			Labels:   map[string]string{"error": err.Error()},
		})
		return nil, err
	} else {
		logger.Log(logging.Entry{
			Severity: logging.Info,
			Payload:  "Auth client initialized successfully",
			Labels:   map[string]string{"status": "success"},
		})
	}

	// Initialize the Messaging client
	messagingClient, err := app.Messaging(ctx)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error initializing Messaging client",
			Labels:   map[string]string{"error": err.Error()},
		})
		return nil, err
	} else {
		logger.Log(logging.Entry{
			Severity: logging.Info,
			Payload:  "Messaging client initialized successfully",
			Labels:   map[string]string{"status": "success"},
		})
	}

	// Initialize the Cloud Tasks client
	taskClient, err := cloudtasks.NewClient(ctx)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error initializing Cloud Tasks client",
			Labels:   map[string]string{"error": err.Error()},
		})
		return nil, err
	} else {
		logger.Log(logging.Entry{
			Severity: logging.Info,
			Payload:  "Cloud Tasks client initialized successfully",
			Labels:   map[string]string{"status": "success"},
		})
	}

	return &types.FirebaseApp{
		Context:       ctx,
		Admin:         app,
		DB:            db,
		Storage:       gcs,
		Auth:          auth,
		Logger:        logger,
		MessageClient: messagingClient,
		TaskClient:    taskClient,
	}, nil
}
