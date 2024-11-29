package tasks

import (
	"encoding/json"
	"io"
	"net/http"

	"proteggo_api/notifications"
	"proteggo_api/tools"
	"proteggo_api/types"

	_ "image/jpeg"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"
	"cloud.google.com/go/storage"
	"firebase.google.com/go/messaging"
	"github.com/gin-gonic/gin"
)

func ImageProcessingTaskHandler(logger *logging.Logger, messageClient *messaging.Client, storage *storage.Client, firestoreClient *firestore.Client) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Maybe use oidc to authenticate the request

		// Get the UploadImageToStorageModel instance from the request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// Unmarshal the body into an UploadImageToStorageModel instance
		var upload types.UploadImageToStorageModel
		err = json.Unmarshal(body, &upload)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// Download the image from the GCS
		img, err := tools.GetImageFromStorage(upload.FilePath, storage, c)

		// Apply the orientation correction
		correctedImg, err := tools.CorrectImageOrientation(logger, img, upload.Orientation)

		// Detect faces in the image
		faces, err := tools.DetectFacesInImage(c, firestoreClient, storage, correctedImg, 10) // TODO: Adjust maxResults as needed
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// Extract vertices from the faces
		facesVertices := []types.FaceVertices{}
		for _, face := range faces {
			facesVertices = append(facesVertices, types.FaceVertices{
				Id:       face.Id,
				ImageId:  upload.Id,
				Vertices: face.Vertices,
			})
		}

		// Get image dimensions
		width, height := tools.GetImageDimensions(correctedImg)

		// Draw borders around the faces
		overlayUrl := ""
		overlayStoragePath := ""
		if len(faces) > 0 {
			overlayStoragePath = types.FIREBASE_STORAGE_FACES_OVERLAY_FOLDER + upload.Id + ".png"
			overlayUrl, err = tools.DrawBordersAroundFaces(c, firestoreClient, storage, upload.Id, width, height, facesVertices)
			if err != nil {
				tools.LogError(logger, c, err)
				return
			}
		}

		// Iterate over the faces
		facesIds := []string{}
		facesUrls := []string{}
		facesStoragePaths := []string{}

		for _, face := range faces {
			err = tools.SetFirestoreDocument(c, firestoreClient, types.FIREBASE_FACES_COLLECTION, face.Id, map[string]interface{}{
				types.FIREBASE_FACES_FIELDS_ID:           face.Id,
				types.FIREBASE_FACES_FIELDS_EMOTION:      face.Emotion,
				types.FIREBASE_FACES_FIELDS_VERTICES:     face.Vertices,
				types.FIREBASE_FACES_FIELDS_LANDMARKS:    face.Landmarks,
				types.FIREBASE_FACES_FIELDS_ROLL_ANGLE:   face.RollAngle,
				types.FIREBASE_FACES_FIELDS_PAN_ANGLE:    face.PanAngle,
				types.FIREBASE_FACES_FIELDS_TILT_ANGLE:   face.TiltAngle,
				types.FIREBASE_FACES_FIELDS_STORAGE_PATH: face.StoragePath,
				types.FIREBASE_FACES_FIELDS_URL:          face.Url,
				types.FIREBASE_FACES_FIELDS_IMAGE_ID:     upload.Id,
				types.FIREBASE_FACES_FIELDS_CREATED_AT:   firestore.ServerTimestamp,
				types.FIREBASE_FACES_FIELDS_POST_ID:      nil,
			})

			if err != nil {
				tools.LogError(logger, c, err)
				return
			}

			facesIds = append(facesIds, face.Id)
			facesUrls = append(facesUrls, face.Url)
			facesStoragePaths = append(facesStoragePaths, face.StoragePath)
		}

		facesIdsField := len(facesIds) > 0
		var facesIdsValue interface{}
		if facesIdsField {
			facesIdsValue = facesIds
		} else {
			facesIdsValue = nil
		}

		facesUrlsField := len(facesUrls) > 0
		var facesUrlsValue interface{}
		if facesUrlsField {
			facesUrlsValue = facesUrls
		} else {
			facesUrlsValue = nil
		}

		facesStoragePathsField := len(facesStoragePaths) > 0
		var facesStoragePathsValue interface{}
		if facesStoragePathsField {
			facesStoragePathsValue = facesStoragePaths
		} else {
			facesStoragePathsValue = nil
		}

		// Encode the image to WebP format
		webpData, err := tools.EncodeWebP(logger, correctedImg, 95) // Adjust quality as needed
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// Generate random file name
		randomName, err := tools.GenerateRandomName()
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// Generate the URL for the image
		storagePath := types.FIREBASE_STORAGE_IMAGES_FOLDER + randomName + ".webp"
		url, err := tools.GenerateImageUrl(c, firestoreClient, storage, webpData, types.FIREBASE_STORAGE_BUCKET, storagePath)

		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// Save the URL to Firestore
		err = tools.SetFirestoreDocument(c, firestoreClient, types.FIREBASE_IMAGES_COLLECTION, upload.Id, map[string]interface{}{
			types.FIREBASE_IMAGES_FIELDS_ID:                         upload.Id,
			types.FIREBASE_IMAGES_FIELDS_STORAGE_PATH:               storagePath,
			types.FIREBASE_IMAGES_FIELDS_URL:                        url,
			types.FIREBASE_IMAGES_FIELDS_CREATED_AT:                 firestore.ServerTimestamp,
			types.FIREBASE_IMAGES_FIELDS_WIDTH:                      width,
			types.FIREBASE_IMAGES_FIELDS_HEIGHT:                     height,
			types.FIREBASE_IMAGES_FIELDS_POST_ID:                    nil,
			types.FIREBASE_IMAGES_FIELDS_FACES_IDS:                  facesIdsValue,
			types.FIREBASE_IMAGES_FIELDS_FACES_URLS:                 facesUrlsValue,
			types.FIREBASE_IMAGES_FIELDS_FACES_STORAGE_PATHS:        facesStoragePathsValue,
			types.FIREBASE_IMAGES_FIELDS_FACES_OVERLAY_URL:          overlayUrl,
			types.FIREBASE_IMAGES_FIELDS_FACES_OVERLAY_STORAGE_PATH: overlayStoragePath,
		})

		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// Delete temp image from GCS
		err = tools.DeleteObjectFromStorage(c, upload.FilePath, storage)

		if err != nil {
			tools.LogError(logger, c, err)
		} else {
			// Send notification to the user
			notifications.SendNotificationToClient(c, messageClient, firestoreClient, logger, types.NotificationMessage{
				ImageId:           upload.Id,
				ImageStoragePath:  storagePath,
				ImageUrl:          url,
				FacesIds:          facesIds,
				FacesUrls:         facesUrls,
				FacesStoragePaths: facesStoragePaths,
			})

			c.JSON(http.StatusOK, gin.H{types.FIREBASE_IMAGES_FIELDS_URL: url})
		}
	}
}
