package tools

import (
	"context"
	"fmt"
	"image"
	"proteggo_api/types"
	"strings"

	"cloud.google.com/go/firestore"
	gcs "cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Sets a document in a Firestore collection
func SetFirestoreDocument(c context.Context, client *firestore.Client, collection, documentName string, data map[string]interface{}) error {
	// Create a reference to the document you want to create
	docRef := client.Collection(collection).Doc(documentName)

	// Set the document with the data
	_, err := docRef.Set(c, data)

	return err
}

func UpdateFirestoreDocument(c context.Context, client *firestore.Client, collection, documentName string, data map[string]interface{}) error {
	// Create a reference to the document you want to update
	docRef := client.Collection(collection).Doc(documentName)

	// Update the document with the data
	_, err := docRef.Set(c, data, firestore.MergeAll)

	return err
}

func AddFirestoreDocument(c context.Context, client *firestore.Client, collection string, data map[string]interface{}) error {
	// Create a reference to the document you want to create
	_, _, err := client.Collection(collection).Add(c, data)

	if err != nil {
		return err
	}

	return nil
}

// Gets a document from a Firestore collection
func GetFirestoreDocument(c context.Context, client *firestore.Client, collection, documentName string) (map[string]interface{}, error) {
	// Create a reference to the document you want to create
	docRef := client.Collection(collection).Doc(documentName)

	// Get the document
	doc, err := docRef.Get(c)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return doc.Data(), nil
}

func GetFirestoreDocuments(c context.Context, client *firestore.Client, collection string) ([]map[string]interface{}, error) {
	// Create a reference to the document you want to create
	iter := client.Collection(collection).Documents(c)
	defer iter.Stop()

	var result []map[string]interface{}

	// Iterate over the documents in the collection
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		result = append(result, doc.Data())
	}

	return result, nil
}

func DeleteFirestoreDocument(c context.Context, client *firestore.Client, collection, documentName string) error {
	// Create a reference to the document you want to delete
	docRef := client.Collection(collection).Doc(documentName)

	// Delete the document
	_, err := docRef.Delete(c)
	if err != nil {
		return err
	}

	return nil
}

func CheckIfImageExistsInStorage(c context.Context, path string, storage *gcs.Client) (bool, error) {
	// Get the bucket and object.
	bucket := storage.Bucket(types.FIREBASE_STORAGE_BUCKET)
	obj := bucket.Object(path)

	// Try to get the object's attributes.
	_, err := obj.Attrs(c)
	if err == gcs.ErrObjectNotExist {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("object.Attrs: %v", err)
	}

	return true, nil
}

func MoveObjectInStorage(c context.Context, srcPath, dstPath string, storage *gcs.Client) error {
	bucket := storage.Bucket(types.FIREBASE_STORAGE_BUCKET)
	srcObj := bucket.Object(srcPath)
	dstObj := bucket.Object(dstPath)

	_, err := dstObj.CopierFrom(srcObj).Run(c)
	if err != nil {
		return err
	}

	if err := srcObj.Delete(c); err != nil {
		return err
	}

	return nil
}

func DeleteObjectFromStorage(c context.Context, path string, storage *gcs.Client) error {
	bucket := types.FIREBASE_STORAGE_BUCKET
	bucketHandle := storage.Bucket(bucket)

	obj := bucketHandle.Object(path)
	if err := obj.Delete(c); err != nil {
		return err
	}

	return nil
}

func DeleteObjectsFromTempFolderStorage(c context.Context, storage *gcs.Client) error {
	bucket := types.FIREBASE_STORAGE_BUCKET
	bucketHandle := storage.Bucket(bucket)

	it := bucketHandle.Objects(c, nil)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		if strings.HasPrefix(attrs.Name, types.FIREBASE_STORAGE_TEMP_FOLDER) {
			obj := bucketHandle.Object(attrs.Name)
			if err := obj.Delete(c); err != nil {
				return err
			}
		}
	}

	return nil
}

func GetImageFromStorage(filePath string, storage *gcs.Client, c context.Context) (image.Image, error) {
	// Download the image from the GCS
	rc, err := storage.Bucket(types.FIREBASE_STORAGE_BUCKET).Object(filePath).NewReader(c)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	// Decode the image
	img, _, err := image.Decode(rc)
	if err != nil {
		return nil, err
	}

	return img, nil
}
