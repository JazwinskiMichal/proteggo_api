package handlers

import (
	"errors"
	"net/http"
	"proteggo_api/tools"
	"proteggo_api/types"
	"strconv"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/logging"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
)

func DeleteHashTagsHandler(logger *logging.Logger, firestoreClient *firestore.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the hash tags from the multipart form
		form, err := c.MultipartForm()
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		hashTagsIds := form.Value["hashTagsIds"]

		for _, hashTagId := range hashTagsIds {
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

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	}
}

func SetHashTagsHandler(logger *logging.Logger, db *firestore.Client) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Get the hash tags from the multipart form
		form, err := c.MultipartForm()
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		hashTagsIds := form.Value["hashTagsIds"]
		hashTagsValues := form.Value["hashTagsValues"]

		// Add each hash tag to the hashTags document
		for i, hashTagId := range hashTagsIds {
			doc, err := tools.GetFirestoreDocument(c, db, types.FIREBASE_POSTS_HASHTAGS_COLLECTION, hashTagId)

			if err != nil {
				tools.LogError(logger, c, err)
				return
			}

			score := 1

			if doc != nil {
				score = int(doc[types.FIREBASE_POSTS_HASHTAGS_FIELDS_SCORE].(int64)) + 1
			}

			err = tools.SetFirestoreDocument(c, db, types.FIREBASE_POSTS_HASHTAGS_COLLECTION, hashTagId, map[string]interface{}{
				types.FIREBASE_POSTS_HASHTAGS_FIELDS_ID:    hashTagId,
				types.FIREBASE_POSTS_HASHTAGS_FIELDS_VALUE: hashTagsValues[i],
				types.FIREBASE_POSTS_HASHTAGS_FIELDS_SCORE: score,
			})

			if err != nil {
				tools.LogError(logger, c, err)
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	}
}

func GetHashTagsHandler(logger *logging.Logger, db *firestore.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var hashTags []types.HashTag

		query := db.Collection(types.FIREBASE_POSTS_HASHTAGS_COLLECTION)

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

			hashTags = append(hashTags, types.HashTag{
				Id:    doc.Data()[types.FIREBASE_POSTS_HASHTAGS_FIELDS_ID].(string),
				Value: doc.Data()[types.FIREBASE_POSTS_HASHTAGS_FIELDS_VALUE].(string),
				Score: int(doc.Data()[types.FIREBASE_POSTS_HASHTAGS_FIELDS_SCORE].(int64)),
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"hashTags": hashTags,
		})
	}
}

func GetTopScoredHashTagsHandler(logger *logging.Logger, db *firestore.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var hashTags []types.HashTag

		limitStr, limitProvided := c.GetQuery("limit")
		if !limitProvided {
			tools.LogError(logger, c, errors.New("limit query parameter is required"))
			return
		}

		// Get the elements limit
		elementsLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			tools.LogError(logger, c, err)
			return
		}

		// Get the elementLimit of hashtags with the highest score
		query := db.Collection(types.FIREBASE_POSTS_HASHTAGS_COLLECTION).OrderBy(types.FIREBASE_POSTS_HASHTAGS_FIELDS_SCORE, firestore.Desc).Limit(elementsLimit)

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

			hashTags = append(hashTags, types.HashTag{
				Id:    doc.Data()[types.FIREBASE_POSTS_HASHTAGS_FIELDS_ID].(string),
				Value: doc.Data()[types.FIREBASE_POSTS_HASHTAGS_FIELDS_VALUE].(string),
				Score: int(doc.Data()[types.FIREBASE_POSTS_HASHTAGS_FIELDS_SCORE].(int64)),
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"hashTags": hashTags,
		})
	}
}
