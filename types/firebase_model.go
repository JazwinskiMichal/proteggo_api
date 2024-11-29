package types

import (
	"context"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"
	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"firebase.google.com/go/messaging"
)

type FirebaseApp struct {
	Context       context.Context
	Admin         *firebase.App
	DB            *firestore.Client
	Storage       *storage.Client
	Auth          *auth.Client
	Logger        *logging.Logger
	MessageClient *messaging.Client
	TaskClient    *cloudtasks.Client
}
