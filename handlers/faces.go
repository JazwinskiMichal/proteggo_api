package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"proteggo_api/tools"
	"proteggo_api/types"
	"strconv"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"
	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
)

func SetObscuredOverlayHandler(logger *logging.Logger, firestoreClient *firestore.Client, storage *storage.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the data
		form, err := c.MultipartForm()
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		imagesIds := form.Value["imagesIds"]
		obscuredStoragePaths := form.Value["obscuredStoragePaths"]

		var failedIds []string
		var obscuredOverlays []types.ObscuredOverlay

		// Iterate over the images and set the obscured overlay
		for i, imageId := range imagesIds {
			tempObscuredStoragePath := obscuredStoragePaths[i]

			// Check if the temp obscured image exists
			exists, err := tools.CheckIfImageExistsInStorage(c, tempObscuredStoragePath, storage)
			if err != nil {
				failedIds = append(failedIds, imageId)
				tools.LogError(logger, c, err)
				continue
			}

			if !exists {
				failedIds = append(failedIds, imageId)
				tools.LogError(logger, c, errors.New("Obscured image does not exist"))
				continue
			}

			// Move the obscured image to the obscured overlay folder
			obscuredStoragePath := types.FIREBASE_STORAGE_OBSCURED_FACES_OVERLAY_FOLDER + imageId + ".png"
			err = tools.MoveObjectInStorage(c, tempObscuredStoragePath, obscuredStoragePath, storage)
			if err != nil {
				failedIds = append(failedIds, imageId)
				tools.LogError(logger, c, err)
				continue
			}

			// Update obscured url
			url, err := tools.UpdateImageUrl(c, firestoreClient, storage, types.FIREBASE_STORAGE_BUCKET, obscuredStoragePath, imageId)
			if err != nil {
				failedIds = append(failedIds, imageId)
				tools.LogError(logger, c, err)
				continue
			}

			// Update the image document with the obscured overlay url and storage path
			err = tools.UpdateFirestoreDocument(c, firestoreClient, types.FIREBASE_IMAGES_COLLECTION, imageId, map[string]interface{}{
				types.FIREBASE_IMAGES_FIELDS_FACES_OBSCURED_OVERLAY_URL:          url,
				types.FIREBASE_IMAGES_FIELDS_FACES_OBSCURED_OVERLAY_STORAGE_PATH: obscuredStoragePath,
			})

			if err != nil {
				failedIds = append(failedIds, imageId)
				tools.LogError(logger, c, err)
				continue
			}

			obscuredOverlays = append(obscuredOverlays, types.ObscuredOverlay{
				Id:          imageId,
				Url:         url,
				StoragePath: obscuredStoragePath,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"failedIds":        failedIds,
			"obscuredOverlays": obscuredOverlays,
		})
	}
}

func CreateTempObscuredOverlayHandler(logger *logging.Logger, firestoreClient *firestore.Client, storage *storage.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the data
		form, err := c.MultipartForm()
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		imageId := form.Value["imageId"][0]
		imageWidth := form.Value["imageWidth"][0]
		imageHeight := form.Value["imageHeight"][0]
		facesIdsToObscure := form.Value["facesIdsToObscure"]

		// Cast the image width and height to int
		imageWidthInt, err := strconv.Atoi(imageWidth)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		imageHeightInt, err := strconv.Atoi(imageHeight)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// Get face vertices from the image
		faceVertices, err := tools.GetFacesVertices(imageId, facesIdsToObscure, c, firestoreClient)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// Create obscured overlay in the temp folder
		obscuredTempStoragePath := types.FIREBASE_STORAGE_TEMP_FOLDER + imageId + ".png"

		if len(faceVertices) == 0 {
			// Delete the obscured image if it exists
			exists, err := tools.CheckIfImageExistsInStorage(c, obscuredTempStoragePath, storage)
			if err != nil {
				tools.LogError(logger, c, err)
				return
			}

			if exists {
				err = tools.DeleteObjectFromStorage(c, obscuredTempStoragePath, storage)
				if err != nil {
					tools.LogError(logger, c, err)
					return
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"obscuredOverlay": types.ObscuredOverlay{
					Id:          imageId,
					Url:         "",
					StoragePath: "",
				},
			})
			return
		}

		obscuredUrl, err := tools.ObscureFacesInImage(c, firestoreClient, storage, imageId, imageWidthInt, imageHeightInt, faceVertices)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"obscuredOverlay": types.ObscuredOverlay{
				Id:          imageId,
				Url:         obscuredUrl,
				StoragePath: obscuredTempStoragePath,
			},
		})
	}
}

func GetFacesOverlayHandler(logger *logging.Logger, firestoreClient *firestore.Client, storage *storage.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Image ID
		imageId, imageIdProvided := c.GetQuery("imageId")
		if !imageIdProvided {
			tools.LogError(logger, c, errors.New("Image ID not provided"))
			return
		}

		// Check if the image exists in the storage
		overlayStoragePath := types.FIREBASE_STORAGE_FACES_OVERLAY_FOLDER + imageId + ".png"
		exists, err := tools.CheckIfImageExistsInStorage(c, overlayStoragePath, storage)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// If not exists, return empty overlay
		if !exists {
			c.JSON(http.StatusOK, gin.H{
				"overlayId":          imageId,
				"overlayUrl":         "",
				"overlayStoragePath": "",
				"width":              0,
				"height":             0,
			})
			return
		}

		// Get image file path from Firestore
		imageDoc, err := tools.GetFirestoreDocument(c, firestoreClient, types.FIREBASE_IMAGES_COLLECTION, imageId)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// Get image file path
		overlayUrl, ok := imageDoc[types.FIREBASE_IMAGES_FIELDS_FACES_OVERLAY_URL].(string)
		if !ok {
			tools.LogError(logger, c, errors.New("Overlay url not found"))
			return
		}

		// Get the image width and height
		width, ok := imageDoc[types.FIREBASE_IMAGES_FIELDS_WIDTH].(int64)
		if !ok {
			tools.LogError(logger, c, errors.New("Width not found"))
			return
		}

		height, ok := imageDoc[types.FIREBASE_IMAGES_FIELDS_HEIGHT].(int64)
		if !ok {
			tools.LogError(logger, c, errors.New("Height not found"))
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"overlayId":          imageId,
			"overlayUrl":         overlayUrl,
			"overlayStoragePath": overlayStoragePath,
			"width":              width,
			"height":             height,
		})
	}
}

func DeleteFacesHandler(logger *logging.Logger, firestoreClient *firestore.Client, storage *storage.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the faces
		var faces map[string]string
		facesJson := c.PostForm("facesToDelete")
		err := json.Unmarshal([]byte(facesJson), &faces)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// collect ids for return
		var deletedIds []string
		var failedIds []string

		// Iterate over the faces and delete each one
		for id, storagePath := range faces {
			// Delete Firestore document
			err := tools.DeleteFirestoreDocument(c, firestoreClient, types.FIREBASE_FACES_COLLECTION, id)
			if err != nil {
				tools.LogError(logger, c, err)
				failedIds = append(failedIds, id)
				continue
			}

			// Delete face from storage
			err = tools.DeleteObjectFromStorage(c, storagePath, storage)
			if err != nil {
				tools.LogError(logger, c, err)
				failedIds = append(failedIds, id)
				continue
			}

			deletedIds = append(deletedIds, id)
		}

		c.JSON(http.StatusOK, gin.H{
			"deletedIds": deletedIds,
			"failedIds":  failedIds,
		})
	}
}

func DeleteObscuredFacesOverlayHandler(logger *logging.Logger, firestoreClient *firestore.Client, storage *storage.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the hash tags from the multipart form
		form, err := c.MultipartForm()
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		imagesIds := form.Value["imagesIds"]

		var deletedIds []string
		var failedIds []string

		for _, imageId := range imagesIds {
			// Get the obscured faces overlay storage path
			obscuredOverlayStoragePath := types.FIREBASE_STORAGE_OBSCURED_FACES_OVERLAY_FOLDER + imageId + ".png"

			// TODO: obscured overlays are not created for each image like, overlays. So in order to avoid errors it would be necessary to provide explicitly the obscured ids to delete. But for now just check if it exists, if not log warning
			// Check if the obscured overlay exists
			exists, err := tools.CheckIfImageExistsInStorage(c, obscuredOverlayStoragePath, storage)
			if err != nil {
				tools.LogError(logger, c, err)
				failedIds = append(failedIds, imageId)
				continue
			}

			if !exists {
				logger.Log(logging.Entry{
					Severity: logging.Warning,
					Payload:  "Obscured overlay does not exist for image id: " + imageId,
					Labels:   map[string]string{"status": "warning"},
				})
				continue
			}

			// Delete the overlay from storage
			if exists {
				err = tools.DeleteObjectFromStorage(c, obscuredOverlayStoragePath, storage)
				if err != nil {
					tools.LogError(logger, c, err)
					failedIds = append(failedIds, imageId)

					continue
				}

				deletedIds = append(deletedIds, imageId)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"deletedIds": deletedIds,
			"failedIds":  failedIds,
		})
	}
}

func DeleteFacesOverlayHandler(logger *logging.Logger, firestoreClient *firestore.Client, storage *storage.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the hash tags from the multipart form
		form, err := c.MultipartForm()
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		imagesIds := form.Value["imagesIds"]

		var deletedIds []string
		var failedIds []string

		for _, imageId := range imagesIds {
			// Get the faces overlay storage path
			overlayStoragePath := types.FIREBASE_STORAGE_FACES_OVERLAY_FOLDER + imageId + ".png"

			// Delete the overlay from storage
			err := tools.DeleteObjectFromStorage(c, overlayStoragePath, storage)
			if err != nil {
				tools.LogError(logger, c, err)
				failedIds = append(failedIds, imageId)
				continue
			}

			deletedIds = append(deletedIds, imageId)
		}

		c.JSON(http.StatusOK, gin.H{
			"deletedIds": deletedIds,
			"failedIds":  failedIds,
		})
	}
}
