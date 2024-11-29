package tools

import (
	"context"
	"errors"
	"image"
	"io"
	"mime/multipart"
	"net/url"
	"proteggo_api/types"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"
	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
)

func DecodeImageInfo(fileInfo map[string]interface{}) (types.DecodedImageInfo, error) {
	fileInterface, exists := fileInfo["file"]
	if !exists {
		return types.DecodedImageInfo{}, errors.New("error getting file from context")
	}

	idInterface, exists := fileInfo["id"]
	if !exists {
		return types.DecodedImageInfo{}, errors.New("error getting id from context")
	}

	extensionRaw, exists := fileInfo["extension"]
	if !exists {
		return types.DecodedImageInfo{}, errors.New("error getting extension from context")
	}

	contentTypeRaw, exists := fileInfo["content_type"]
	if !exists {
		return types.DecodedImageInfo{}, errors.New("error getting content type from context")
	}

	file, ok := fileInterface.(multipart.File)
	if !ok {
		return types.DecodedImageInfo{}, errors.New("error casting file to multipart.File")
	}

	id, ok := idInterface.(string)
	if !ok {
		return types.DecodedImageInfo{}, errors.New("error casting id to string")
	}

	extension, ok := extensionRaw.(string)
	if !ok {
		return types.DecodedImageInfo{}, errors.New("error casting extension to string")
	}

	contentType, ok := contentTypeRaw.(string)
	if !ok {
		return types.DecodedImageInfo{}, errors.New("error casting content type to string")
	}

	return types.DecodedImageInfo{
		File:        file,
		Id:          id,
		Extension:   extension,
		ContentType: contentType,
	}, nil
}

func GenerateImageUrl(c *gin.Context, client *firestore.Client, storage *storage.Client, data []byte, bucket string, storagePath string) (string, error) {
	bucketHandle := storage.Bucket(bucket)

	// Get the object
	obj := bucketHandle.Object(storagePath)
	sw := obj.NewWriter(c)

	// Write the WebP data to GCS
	if _, err := sw.Write(data); err != nil {
		return "", err
	}

	if err := sw.Close(); err != nil {
		return "", err
	}

	// Generate a download token
	downloadToken, err := GenerateRandomName()
	if err != nil {
		return "", err
	}

	// Update Firebase storage download token
	err = UpdateFirebaseStorageDownloadToken(c, obj, bucket, downloadToken)
	if err != nil {
		return "", err
	}

	// Generate the URL
	url := "https://firebasestorage.googleapis.com/v0/b/" + bucket + "/o/" + url.PathEscape(storagePath) + "?alt=media&token=" + downloadToken

	return url, nil
}

func UpdateImageUrl(c context.Context, client *firestore.Client, storage *storage.Client, bucket string, storagePath string, downloadToken string) (string, error) {
	bucketHandle := storage.Bucket(bucket)

	// Get the object
	obj := bucketHandle.Object(storagePath)

	// Update Firebase storage download token
	err := UpdateFirebaseStorageDownloadToken(c, obj, bucket, downloadToken)
	if err != nil {
		return "", err
	}

	// Generate the URL
	url := "https://firebasestorage.googleapis.com/v0/b/" + bucket + "/o/" + url.PathEscape(storagePath) + "?alt=media&token=" + downloadToken

	return url, nil
}

func GetImageDimensions(img image.Image) (int, int) {
	return img.Bounds().Dx(), img.Bounds().Dy()
}

// TODO: What could spped up the process is, instead of making a POST request and then create task, maybe just upload the image from the client to the GCS, and somehow create the task from the gcs?
func UploadImageToStorage(context context.Context, file multipart.File, logger *logging.Logger, storage *storage.Client, id string, contentType string, fileExtension string) (*types.UploadImageToStorageModel, error) {

	// Get Exif orientation
	orientation, err := TryFindExifOrientation(logger, file)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error finding EXIF orientation",
			Labels:   map[string]string{"error": err.Error()},
		})
		return nil, err
	}

	bucket := types.FIREBASE_STORAGE_BUCKET
	bucketHandle := storage.Bucket(bucket)

	folderName := types.FIREBASE_STORAGE_TEMP_FOLDER

	// Generate random file name
	randomName, err := GenerateRandomName()
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error generating random name",
			Labels:   map[string]string{"error": err.Error()},
		})
		return nil, err
	}

	objectName := folderName + randomName + fileExtension
	obj := bucketHandle.Object(objectName)
	sw := obj.NewWriter(context)
	sw.ContentType = contentType

	if _, err := io.Copy(sw, file); err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error writing image to storage",
			Labels:   map[string]string{"error": err.Error()},
		})
		return nil, err
	}

	if err := sw.Close(); err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error closing storage writer",
			Labels:   map[string]string{"error": err.Error()},
		})
		return nil, err
	}

	return &types.UploadImageToStorageModel{
		Id:          id,
		FilePath:    objectName,
		Orientation: orientation,
	}, nil
}
