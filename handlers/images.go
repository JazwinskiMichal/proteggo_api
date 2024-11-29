package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"proteggo_api/middlewares"
	"proteggo_api/tasks"
	"proteggo_api/tools"
	"proteggo_api/types"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"
	"cloud.google.com/go/storage"
	"firebase.google.com/go/messaging"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
)

func UploadImagesHandler(logger *logging.Logger, firestoreClient *firestore.Client, storage *storage.Client, messageClient *messaging.Client, tasksClient *cloudtasks.Client) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Use the middleware
		middlewares.ImageValidationMiddleware(logger)(c)
		if c.IsAborted() {
			// If the middleware aborted the request, stop the handler
			return
		}

		// Get fileInfos from the context
		fileInfosInterface, exists := c.Get("fileInfos")
		if !exists {
			tools.LogError(logger, c, errors.New("Error getting fileInfos from context"))
			return
		}

		fileInfos, ok := fileInfosInterface.([]map[string]interface{})
		if !ok {
			tools.LogError(logger, c, errors.New("Error casting fileInfos to []map[string]interface{}"))
			return
		}

		// collect ids for return
		var uploadedIds []string
		var failedIds []string

		for _, fileInfo := range fileInfos {

			// Get the image data from the context
			decodedFileInfo, err := tools.DecodeImageInfo(fileInfo)
			if err != nil {
				tools.LogError(logger, c, err)
				return
			}

			// Upload the image temporarly to Firebase Storage, so the imge url can be taken to the task handler for processing
			upload, err := tools.UploadImageToStorage(c, decodedFileInfo.File, logger, storage, decodedFileInfo.Id, decodedFileInfo.ContentType, decodedFileInfo.Extension)
			if err != nil {
				logger.Log(logging.Entry{
					Severity: logging.Error,
					Payload:  "Error uploading image to storage",
					Labels:   map[string]string{"error": err.Error()},
				})
				failedIds = append(failedIds, decodedFileInfo.Id)
			} else {

				// Create a task to process the image
				_, err := tasks.CreateTask(c, tasksClient, logger, upload)
				if err != nil {
					logger.Log(logging.Entry{
						Severity: logging.Error,
						Payload:  "Error creating task",
						Labels:   map[string]string{"error": err.Error()},
					})
					failedIds = append(failedIds, decodedFileInfo.Id)
				} else {
					uploadedIds = append(uploadedIds, decodedFileInfo.Id)
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"uploadedIds": uploadedIds,
			"failedIds":   failedIds,
		})
	}
}

func DeleteTempImagesHandler(logger *logging.Logger, firestoreClient *firestore.Client, storage *storage.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Delete all images in the storage _temp folder
		err := tools.DeleteObjectsFromTempFolderStorage(c, storage)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Temp images deleted successfully",
			"error":   false,
		})
	}
}

func DeleteUnusedImagesHandler(logger *logging.Logger, firestoreClient *firestore.Client, storage *storage.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := firestoreClient.Collection(types.FIREBASE_IMAGES_COLLECTION).OrderBy(types.FIREBASE_IMAGES_FIELDS_CREATED_AT, firestore.Desc)

		docs, err := query.Documents(c).GetAll()
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// Get all the images and check if they are used in any post
		for _, doc := range docs {
			var postId = doc.Data()[types.FIREBASE_IMAGES_FIELDS_POST_ID]
			if postId == nil {
				id, idOk := doc.Data()[types.FIREBASE_IMAGES_FIELDS_ID].(string)
				if !idOk {
					tools.LogError(logger, c, errors.New("Error casting id to string"))
					return
				}

				storagePath, storagePathOk := doc.Data()[types.FIREBASE_IMAGES_FIELDS_STORAGE_PATH].(string)
				if !storagePathOk {
					tools.LogError(logger, c, errors.New("Error casting storagePath to string"))
					return
				}

				facesIdsInterface, facesIdsOk := doc.Data()[types.FIREBASE_IMAGES_FIELDS_FACES_IDS]
				facesOverlayStoragePathInterface, facesOverlayStoragePathOk := doc.Data()[types.FIREBASE_IMAGES_FIELDS_FACES_OVERLAY_STORAGE_PATH]

				err := tools.DeleteFirestoreDocument(c, firestoreClient, types.FIREBASE_IMAGES_COLLECTION, id)
				if err != nil {
					tools.LogError(logger, c, err)
					return
				}

				err = tools.DeleteObjectFromStorage(c, storagePath, storage)
				if err != nil {
					tools.LogError(logger, c, err)
					return
				}

				// Check if the image has faces
				if facesIdsOk && facesIdsInterface != nil {

					facesIds, ok := facesIdsInterface.([]interface{})
					if !ok {
						tools.LogError(logger, c, errors.New("Error casting facesIds to []interface{}"))
						return
					}

					if len(facesIds) > 0 {
						for _, faceId := range facesIds {
							// Find the face document
							faceDoc, err := firestoreClient.Collection(types.FIREBASE_FACES_COLLECTION).Doc(faceId.(string)).Get(c)

							var faceId = faceDoc.Data()[types.FIREBASE_FACES_FIELDS_ID]
							var faceStoragePath = faceDoc.Data()[types.FIREBASE_FACES_FIELDS_STORAGE_PATH]

							err = tools.DeleteFirestoreDocument(c, firestoreClient, types.FIREBASE_FACES_COLLECTION, faceId.(string))
							if err != nil {
								tools.LogError(logger, c, err)
								return
							}

							err = tools.DeleteObjectFromStorage(c, faceStoragePath.(string), storage)
							if err != nil {
								tools.LogError(logger, c, err)
								return
							}
						}
					}
				}

				// Check if image has faces overlay
				if facesOverlayStoragePathOk && facesOverlayStoragePathInterface != nil {
					facesOverlayStoragePath, ok := facesOverlayStoragePathInterface.(string)
					if !ok {
						tools.LogError(logger, c, errors.New("Error casting facesOverlayStoragePath to string"))
						return
					}

					if facesOverlayStoragePath != "" {
						err = tools.DeleteObjectFromStorage(c, facesOverlayStoragePath, storage)
						if err != nil {
							tools.LogError(logger, c, err)
							return
						}
					}
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Unused images deleted successfully",
			"error":   false,
		})
	}
}

func DeleteImagesHandler(logger *logging.Logger, firestoreClient *firestore.Client, storage *storage.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the images
		var images map[string]string
		imagesJson := c.PostForm("imagesToDelete")
		err := json.Unmarshal([]byte(imagesJson), &images)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// collect ids for return
		var deletedIds []string
		var failedIds []string

		// Iterate over the images and delete each one
		for id, storagePath := range images {
			// Delete Firestore document
			err := tools.DeleteFirestoreDocument(c, firestoreClient, types.FIREBASE_IMAGES_COLLECTION, id)
			if err != nil {
				tools.LogError(logger, c, err)
				failedIds = append(failedIds, id)
				continue
			}

			// Delete image from storage
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

func GetImagesHandler(logger *logging.Logger, client *firestore.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the page size and page token from the query parameters
		pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
		pageToken := c.Query("pageToken")

		// Create a query to get the documents in the "images" collection
		query := client.Collection(types.FIREBASE_IMAGES_COLLECTION).OrderBy("createdAt", firestore.Asc).Limit(pageSize)

		// If a page token is provided, start after the document specified by the page token
		if pageToken != "" {
			doc, err := client.Collection(types.FIREBASE_IMAGES_COLLECTION).Doc(pageToken).Get(c)
			if err != nil {
				tools.LogError(logger, c, err)
				return
			}
			query = query.StartAfter(doc)
		}

		// Get an iterator over the documents
		iter := query.Documents(c)

		var paths []string
		var nextPageToken string

		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				tools.LogError(logger, c, err)
				return
			}

			// Get the URL from the document
			url, ok := doc.Data()["url"].(string)
			if !ok {
				tools.LogError(logger, c, errors.New("Error getting URL from document"))
				return
			}

			// Add the URL to the paths
			paths = append(paths, url)

			// Set the next page token to the name of the current document
			nextPageToken = doc.Ref.ID
		}

		c.JSON(http.StatusOK, gin.H{
			"paths":         paths,
			"nextPageToken": nextPageToken,
		})
	}
}
