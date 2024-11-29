package tools

import (
	"context"

	"cloud.google.com/go/storage"
)

// Updates the download token for a file in Firebase Storage
func UpdateFirebaseStorageDownloadToken(ctx context.Context, obj *storage.ObjectHandle, bucket, token string) error {
	// Define the new metadata
	newMetadata := map[string]string{
		"firebaseStorageDownloadTokens": token,
	}

	// Update the object with the new metadata
	_, err := obj.Update(ctx, storage.ObjectAttrsToUpdate{
		Metadata: newMetadata,
	})
	return err
}
