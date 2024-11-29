package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"proteggo_api/tools"
	"proteggo_api/types"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"
	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
)

func SubmitPostHandler(logger *logging.Logger, db *firestore.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the post from the multipart form
		form, err := c.MultipartForm()
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		id := form.Value[types.FIREBASE_POSTS_FIELDS_ID]
		hashTagsValues := form.Value[types.FIREBASE_POSTS_FIELDS_HASH_TAGS_VALUES]
		hashTagsIds := form.Value[types.FIREBASE_POSTS_FIELDS_HASH_TAGS_IDS]
		body := form.Value[types.FIREBASE_POSTS_FIELDS_BODY]
		imagesIds := form.Value[types.FIREBASE_POSTS_FIELDS_IMAGES_IDS]
		imagesUrls := form.Value[types.FIREBASE_POSTS_FIELDS_IMAGES_URLS]
		imagesStoragePaths := form.Value[types.FIREBASE_POSTS_FIELDS_IMAGES_STORAGE_PATHS]
		// Unmarshal the facesIds from the form
		facesIdsJson := form.Value[types.FIREBASE_POSTS_FIELDS_FACES_IDS][0]
		var facesIds map[string][]string
		err = json.Unmarshal([]byte(facesIdsJson), &facesIds)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}
		// Unmarshal the facesUrls from the form
		facesUrlsJson := form.Value[types.FIREBASE_POSTS_FIELDS_FACES_URLS][0]
		var facesUrls map[string][]string
		err = json.Unmarshal([]byte(facesUrlsJson), &facesUrls)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}
		// Unmarshal the facesStoragePaths from the form
		facesStoragePathsJson := form.Value[types.FIREBASE_POSTS_FIELDS_FACES_STORAGE_PATHS][0]
		var facesStoragePaths map[string][]string
		err = json.Unmarshal([]byte(facesStoragePathsJson), &facesStoragePaths)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}
		overlaysIds := form.Value[types.FIREBASE_POSTS_FIELDS_OVERLAYS_IDS]
		overlaysUrls := form.Value[types.FIREBASE_POSTS_FIELDS_OVERLAYS_URLS]
		overlaysStoragePaths := form.Value[types.FIREBASE_POSTS_FIELDS_OVERLAYS_STORAGE_PATHS]
		obscuredOverlaysIds := form.Value[types.FIREBASE_POSTS_FIELDS_OBSCURED_OVERLAYS_IDS]
		obscuredOverlaysUrls := form.Value[types.FIREBASE_POSTS_FIELDS_OBSCURED_OVERLAYS_URLS]
		obscuredOverlaysStoragePaths := form.Value[types.FIREBASE_POSTS_FIELDS_OBSCURED_OVERLAYS_STORAGE_PATHS]

		// Add the post to the Firestore database
		err = tools.SetFirestoreDocument(c, db, types.FIREBASE_POSTS_COLLECTION, id[0], map[string]interface{}{
			types.FIREBASE_POSTS_FIELDS_ID:                              id[0],
			types.FIREBASE_POSTS_FIELDS_HASH_TAGS_VALUES:                hashTagsValues,
			types.FIREBASE_POSTS_FIELDS_HASH_TAGS_IDS:                   hashTagsIds,
			types.FIREBASE_POSTS_FIELDS_BODY:                            body[0],
			types.FIREBASE_POSTS_FIELDS_IMAGES_IDS:                      imagesIds,
			types.FIREBASE_POSTS_FIELDS_IMAGES_URLS:                     imagesUrls,
			types.FIREBASE_POSTS_FIELDS_IMAGES_STORAGE_PATHS:            imagesStoragePaths,
			types.FIREBASE_POSTS_FIELDS_CREATED_AT:                      firestore.ServerTimestamp,
			types.FIREBASE_POSTS_FIELDS_FACES_IDS:                       facesIds,
			types.FIREBASE_POSTS_FIELDS_FACES_URLS:                      facesUrls,
			types.FIREBASE_POSTS_FIELDS_FACES_STORAGE_PATHS:             facesStoragePaths,
			types.FIREBASE_POSTS_FIELDS_OVERLAYS_IDS:                    overlaysIds,
			types.FIREBASE_POSTS_FIELDS_OVERLAYS_URLS:                   overlaysUrls,
			types.FIREBASE_POSTS_FIELDS_OVERLAYS_STORAGE_PATHS:          overlaysStoragePaths,
			types.FIREBASE_POSTS_FIELDS_OBSCURED_OVERLAYS_IDS:           obscuredOverlaysIds,
			types.FIREBASE_POSTS_FIELDS_OBSCURED_OVERLAYS_URLS:          obscuredOverlaysUrls,
			types.FIREBASE_POSTS_FIELDS_OBSCURED_OVERLAYS_STORAGE_PATHS: obscuredOverlaysStoragePaths,
			// TODO: are overlays here missing?
		})

		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// If an image is used in the post, update the image document in Firestore
		for _, imageId := range imagesIds {
			if err != nil {
				tools.LogError(logger, c, err)
				return
			}

			err = tools.UpdateFirestoreDocument(c, db, types.FIREBASE_IMAGES_COLLECTION, imageId, map[string]interface{}{
				types.FIREBASE_IMAGES_FIELDS_POST_ID: id[0],
				// TODO: here should also save overlays and obscured overlays, because otherwise those are not being saved to the images table
			})

			// If an image has faces, update the face documents in Firestore
			for _, faceIds := range facesIds[imageId] {
				if err != nil {
					tools.LogError(logger, c, err)
					return
				}

				err = tools.UpdateFirestoreDocument(c, db, types.FIREBASE_FACES_COLLECTION, faceIds, map[string]interface{}{
					types.FIREBASE_FACES_FIELDS_POST_ID: id[0],
				})

				if err != nil {
					tools.LogError(logger, c, err)
					return
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	}
}

func GetPostsHandler(logger *logging.Logger, db *firestore.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var posts []types.Post

		// Get the page number and page size from the URL
		pageNumber, err := strconv.Atoi(c.DefaultQuery("pageNumber", "1"))
		pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
		startDateStr, startDateProvided := c.GetQuery("startDate")
		endDateStr, endDateProvided := c.GetQuery("endDate")

		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// Calculate the number of documents to skip
		skip := (pageNumber - 1) * pageSize

		// Start a query for the posts collection
		query := db.Collection(types.FIREBASE_POSTS_COLLECTION).OrderBy(types.FIREBASE_POSTS_FIELDS_CREATED_AT, firestore.Desc)

		// If a start date is provided, add a filter for it
		if startDateProvided {
			startDate, err := time.Parse(time.RFC3339, startDateStr)

			if err != nil {
				tools.LogError(logger, c, err)
				return
			}

			query = query.Where(types.FIREBASE_POSTS_FIELDS_CREATED_AT, ">=", startDate)
		}

		// If an end date is provided, add a filter for it
		if endDateProvided {
			endDate, err := time.Parse(time.RFC3339, endDateStr)

			if err != nil {
				tools.LogError(logger, c, err)
				return
			}

			query = query.Where(types.FIREBASE_POSTS_FIELDS_CREATED_AT, "<=", endDate)
		}

		// If this is not the first page, start after the last document of the previous page
		if pageNumber > 1 {
			docs, err := query.Documents(c).GetAll()

			if err != nil {
				tools.LogError(logger, c, err)
				return
			}

			lastDocSnapshot := docs[skip-1]
			query = query.StartAfter(lastDocSnapshot)
		}

		// Limit the query to the page size
		query = query.Limit(pageSize)

		// Get all posts that match the query
		iter := query.Documents(c)
		defer iter.Stop()

		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}

			if err != nil {
				tools.LogError(logger, c, err)
				return
			}

			posts = append(posts, types.Post{
				Id:                           doc.Data()[types.FIREBASE_POSTS_FIELDS_ID].(string),
				Body:                         doc.Data()[types.FIREBASE_POSTS_FIELDS_BODY].(string),
				CreatedAt:                    doc.Data()[types.FIREBASE_POSTS_FIELDS_CREATED_AT].(time.Time).Format(time.RFC3339),
				HashTagsValues:               convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_HASH_TAGS_VALUES]),
				HashTagsIds:                  convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_HASH_TAGS_IDS]),
				ImagesIds:                    convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_IMAGES_IDS]),
				ImagesUrls:                   convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_IMAGES_URLS]),
				ImagesStoragePaths:           convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_IMAGES_STORAGE_PATHS]),
				FacesIds:                     convertInterfaceToMapStringArray(doc.Data()[types.FIREBASE_POSTS_FIELDS_FACES_IDS]),
				FacesUrls:                    convertInterfaceToMapStringArray(doc.Data()[types.FIREBASE_POSTS_FIELDS_FACES_URLS]),
				FacesStoragePaths:            convertInterfaceToMapStringArray(doc.Data()[types.FIREBASE_POSTS_FIELDS_FACES_STORAGE_PATHS]),
				OverlaysIds:                  convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_OVERLAYS_IDS]),
				OverlaysUrls:                 convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_OVERLAYS_URLS]),
				OverlaysStoragePaths:         convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_OVERLAYS_STORAGE_PATHS]),
				ObscuredOverlaysIds:          convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_OBSCURED_OVERLAYS_IDS]),
				ObscuredOverlaysUrls:         convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_OBSCURED_OVERLAYS_URLS]),
				ObscuredOverlaysStoragePaths: convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_OBSCURED_OVERLAYS_STORAGE_PATHS]),
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"posts": posts,
		})
	}
}

func GetPostsByHashTagsHandler(logger *logging.Logger, db *firestore.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var posts []types.Post

		// Get the hashTags parameter from the URL and split it into individual hashTags
		hashTags := strings.Split(c.Param("hashTags"), ",")

		// Start a query for the posts collection
		query := db.Collection(types.FIREBASE_POSTS_COLLECTION).Query

		// Add a where clause for the hashTags
		query = query.Where(types.FIREBASE_POSTS_FIELDS_HASH_TAGS_VALUES, "array-contains-any", hashTags)

		// Get all posts with the hash tags
		iter := query.Documents(c)
		defer iter.Stop()

		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}

			if err != nil {
				tools.LogError(logger, c, err)
				return
			}

			posts = append(posts, types.Post{
				Id:                           doc.Data()[types.FIREBASE_POSTS_FIELDS_ID].(string),
				Body:                         doc.Data()[types.FIREBASE_POSTS_FIELDS_BODY].(string),
				CreatedAt:                    doc.Data()[types.FIREBASE_POSTS_FIELDS_CREATED_AT].(time.Time).Format(time.RFC3339),
				HashTagsValues:               convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_HASH_TAGS_VALUES]),
				HashTagsIds:                  convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_HASH_TAGS_IDS]),
				ImagesIds:                    convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_IMAGES_IDS]),
				ImagesUrls:                   convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_IMAGES_URLS]),
				ImagesStoragePaths:           convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_IMAGES_STORAGE_PATHS]),
				FacesIds:                     convertInterfaceToMapStringArray(doc.Data()[types.FIREBASE_POSTS_FIELDS_FACES_IDS]),
				FacesUrls:                    convertInterfaceToMapStringArray(doc.Data()[types.FIREBASE_POSTS_FIELDS_FACES_URLS]),
				FacesStoragePaths:            convertInterfaceToMapStringArray(doc.Data()[types.FIREBASE_POSTS_FIELDS_FACES_STORAGE_PATHS]),
				OverlaysIds:                  convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_OVERLAYS_IDS]),
				OverlaysUrls:                 convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_OVERLAYS_URLS]),
				OverlaysStoragePaths:         convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_OVERLAYS_STORAGE_PATHS]),
				ObscuredOverlaysIds:          convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_OBSCURED_OVERLAYS_IDS]),
				ObscuredOverlaysUrls:         convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_OBSCURED_OVERLAYS_URLS]),
				ObscuredOverlaysStoragePaths: convertInterfaceToArrayString(doc.Data()[types.FIREBASE_POSTS_FIELDS_OBSCURED_OVERLAYS_STORAGE_PATHS]),
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"posts": posts,
		})
	}
}

// TODO: Test delete with corelation to delete overlays and obscured overlays
func DeletePostHandler(logger *logging.Logger, firestoreClient *firestore.Client, storage *storage.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the post id
		id, idProvided := c.GetQuery(types.FIREBASE_POSTS_FIELDS_ID)
		if !idProvided {
			tools.LogError(logger, c, errors.New("id query parameter is required"))
			return
		}

		// Find the post in Firestore
		post, err := tools.GetFirestoreDocument(c, firestoreClient, types.FIREBASE_POSTS_COLLECTION, id)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// Get the hashTagsIds, imagesIds, imagesStoragePaths, faces and obscuredOverlays storage path from the post
		hashsTagsIds := convertInterfaceToArrayString(post[types.FIREBASE_POSTS_FIELDS_HASH_TAGS_IDS])
		imagesIds := convertInterfaceToArrayString(post[types.FIREBASE_POSTS_FIELDS_IMAGES_IDS])
		imagesStoragePaths := convertInterfaceToArrayString(post[types.FIREBASE_POSTS_FIELDS_IMAGES_STORAGE_PATHS])
		facesIds := convertInterfaceToMapStringArray(post[types.FIREBASE_POSTS_FIELDS_FACES_IDS])
		facesStoragePaths := convertInterfaceToMapStringArray(post[types.FIREBASE_POSTS_FIELDS_FACES_STORAGE_PATHS])
		overlaysStoragePaths := convertInterfaceToArrayString(post[types.FIREBASE_POSTS_FIELDS_OVERLAYS_STORAGE_PATHS])
		obscuredOverlaysStoragePaths := convertInterfaceToArrayString(post[types.FIREBASE_POSTS_FIELDS_OBSCURED_OVERLAYS_STORAGE_PATHS])

		// Find the hash tags in Firestore and check its score
		for _, hashTagId := range hashsTagsIds {
			hashTag, err := tools.GetFirestoreDocument(c, firestoreClient, types.FIREBASE_POSTS_HASHTAGS_COLLECTION, hashTagId)
			if err != nil {
				tools.LogError(logger, c, err)
				return
			}

			score := int(hashTag[types.FIREBASE_POSTS_HASHTAGS_FIELDS_SCORE].(int64))

			// If the score is 1, delete the hash tag from Firestore
			if score <= 1 {
				err := tools.DeleteFirestoreDocument(c, firestoreClient, types.FIREBASE_POSTS_HASHTAGS_COLLECTION, hashTagId)
				if err != nil {
					tools.LogError(logger, c, err)
					return
				}
			} else {
				// If the score is greater than 1, decrement the score by 1
				err := tools.SetFirestoreDocument(c, firestoreClient, types.FIREBASE_POSTS_HASHTAGS_COLLECTION, hashTagId, map[string]interface{}{
					types.FIREBASE_POSTS_HASHTAGS_FIELDS_ID:    hashTagId,
					types.FIREBASE_POSTS_HASHTAGS_FIELDS_VALUE: hashTag[types.FIREBASE_POSTS_HASHTAGS_FIELDS_VALUE],
					types.FIREBASE_POSTS_HASHTAGS_FIELDS_SCORE: score - 1,
				})
				if err != nil {
					tools.LogError(logger, c, err)
					return
				}
			}
		}

		// Delete each image from the images collection
		for _, imageId := range imagesIds {
			err := tools.DeleteFirestoreDocument(c, firestoreClient, types.FIREBASE_IMAGES_COLLECTION, imageId)
			if err != nil {
				tools.LogError(logger, c, err)
				return
			}
		}

		// Delete each image from storage
		for _, path := range imagesStoragePaths {
			err := tools.DeleteObjectFromStorage(c, path, storage)
			if err != nil {
				tools.LogError(logger, c, err)
				return
			}
		}

		// Delete each face from the faces collection
		for _, faceIds := range facesIds {
			for _, faceId := range faceIds {
				err := tools.DeleteFirestoreDocument(c, firestoreClient, types.FIREBASE_FACES_COLLECTION, faceId)
				if err != nil {
					tools.LogError(logger, c, err)
					return
				}
			}
		}

		// Delete each face from storage
		for _, paths := range facesStoragePaths {
			for _, path := range paths {
				err := tools.DeleteObjectFromStorage(c, path, storage)
				if err != nil {
					tools.LogError(logger, c, err)
					return
				}
			}
		}

		// Delete each obscured overlay from storage
		for _, path := range obscuredOverlaysStoragePaths {
			err := tools.DeleteObjectFromStorage(c, path, storage)
			if err != nil {
				tools.LogError(logger, c, err)
				return
			}
		}

		// Delete each overlay from storage
		for _, path := range overlaysStoragePaths {
			err := tools.DeleteObjectFromStorage(c, path, storage)
			if err != nil {
				tools.LogError(logger, c, err)
				return
			}
		}

		// Delete the post from Firestore
		err = tools.DeleteFirestoreDocument(c, firestoreClient, types.FIREBASE_POSTS_COLLECTION, id)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	}
}

func convertInterfaceToArrayString(data interface{}) []string {
	if data == nil {
		return []string{}
	}

	dataSlice, ok := data.([]interface{})
	if !ok {
		// data is not a slice of interfaces, return an empty slice
		return []string{}
	}

	result := make([]string, len(dataSlice))
	for i, v := range dataSlice {
		str, ok := v.(string)
		if ok {
			result[i] = str
		}
	}

	return result
}

func convertInterfaceToMapStringArray(value interface{}) map[string][]string {
	result := make(map[string][]string)
	for key, val := range value.(map[string]interface{}) {
		result[key] = convertInterfaceToArrayString(val)
	}
	return result
}
