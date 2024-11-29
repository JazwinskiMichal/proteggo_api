package main

import (
	"log"
	"os"

	"proteggo_api/firebase"
	"proteggo_api/handlers"
	"proteggo_api/middlewares"
	"proteggo_api/tasks"
	"proteggo_api/types"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize Firebase app
	firebaseApp, err := firebase.InitFirebaseApp()
	if err != nil {
		log.Fatalf("Failed to initialize Firebase: %v\n", err)
	}

	// Check each component of the Firebase app
	if firebaseApp.Admin == nil {
		log.Fatalf("Failed to initialize Firebase Admin app\n")
	}
	if firebaseApp.DB == nil {
		log.Fatalf("Failed to initialize Firestore client\n")
	}
	if firebaseApp.Storage == nil {
		log.Fatalf("Failed to initialize Google Cloud Storage client\n")
	}
	if firebaseApp.Auth == nil {
		log.Fatalf("Failed to initialize Firebase Auth client\n")
	}
	if firebaseApp.Logger == nil {
		log.Fatalf("Failed to initialize Firebase Logger\n")
	}
	if firebaseApp.MessageClient == nil {
		log.Fatalf("Failed to initialize Firebase Messaging client\n")
	}
	if firebaseApp.TaskClient == nil {
		log.Fatalf("Failed to initialize Firebase Task client\n")
	}

	r := gin.Default()

	// Disable TrustedProxies feature
	err = r.SetTrustedProxies(nil)
	if err != nil {
		log.Fatalf("Failed to set trusted proxies: %v\n", err)
	}

	// Define the routes for the tasks handler
	taskGroup := r.Group(types.CLOUD_TASKS_HANDLER_PATH)
	taskGroup.POST("", tasks.ImageProcessingTaskHandler(firebaseApp.Logger, firebaseApp.MessageClient, firebaseApp.Storage, firebaseApp.DB))

	// Define the routes for the application
	hashTagsGroup := r.Group("/api/hashTags")
	hashTagsGroup.Use(middlewares.AuthMiddleware(firebaseApp.Logger, firebaseApp.Auth))
	hashTagsGroup.GET("", handlers.GetHashTagsHandler(firebaseApp.Logger, firebaseApp.DB))
	hashTagsGroup.GET("/topScored", handlers.GetTopScoredHashTagsHandler(firebaseApp.Logger, firebaseApp.DB))
	hashTagsGroup.Use(middlewares.AdminAuthMiddleware(firebaseApp.Logger))
	hashTagsGroup.POST("", handlers.SetHashTagsHandler(firebaseApp.Logger, firebaseApp.DB))
	hashTagsGroup.DELETE("", handlers.DeleteHashTagsHandler(firebaseApp.Logger, firebaseApp.DB))

	postsGroup := r.Group("/api/posts")
	postsGroup.Use(middlewares.AuthMiddleware(firebaseApp.Logger, firebaseApp.Auth))
	postsGroup.GET("", handlers.GetPostsHandler(firebaseApp.Logger, firebaseApp.DB))
	postsGroup.GET("/byHashTags/:hashTags", handlers.GetPostsByHashTagsHandler(firebaseApp.Logger, firebaseApp.DB))
	postsGroup.Use(middlewares.AdminAuthMiddleware(firebaseApp.Logger))
	postsGroup.POST("", handlers.SubmitPostHandler(firebaseApp.Logger, firebaseApp.DB))
	postsGroup.DELETE("", handlers.DeletePostHandler(firebaseApp.Logger, firebaseApp.DB, firebaseApp.Storage))

	imagesGroup := r.Group("/api/images")
	imagesGroup.DELETE("/deleteTemp", handlers.DeleteTempImagesHandler(firebaseApp.Logger, firebaseApp.DB, firebaseApp.Storage))
	imagesGroup.DELETE("/deleteUnused", handlers.DeleteUnusedImagesHandler(firebaseApp.Logger, firebaseApp.DB, firebaseApp.Storage))
	imagesGroup.Use(middlewares.AuthMiddleware(firebaseApp.Logger, firebaseApp.Auth))
	imagesGroup.GET("", handlers.GetImagesHandler(firebaseApp.Logger, firebaseApp.DB))
	imagesGroup.Use(middlewares.AdminAuthMiddleware(firebaseApp.Logger))
	imagesGroup.POST("", handlers.UploadImagesHandler(firebaseApp.Logger, firebaseApp.DB, firebaseApp.Storage, firebaseApp.MessageClient, firebaseApp.TaskClient))
	imagesGroup.DELETE("", handlers.DeleteImagesHandler(firebaseApp.Logger, firebaseApp.DB, firebaseApp.Storage))

	facesGroup := r.Group("/api/faces")
	facesGroup.Use(middlewares.AuthMiddleware(firebaseApp.Logger, firebaseApp.Auth))
	facesGroup.Use(middlewares.AdminAuthMiddleware(firebaseApp.Logger))
	facesGroup.GET("/overlay", handlers.GetFacesOverlayHandler(firebaseApp.Logger, firebaseApp.DB, firebaseApp.Storage))
	facesGroup.POST("/overlay/obscured", handlers.SetObscuredOverlayHandler(firebaseApp.Logger, firebaseApp.DB, firebaseApp.Storage))
	facesGroup.POST("/overlay/obscured/temp", handlers.CreateTempObscuredOverlayHandler(firebaseApp.Logger, firebaseApp.DB, firebaseApp.Storage))
	facesGroup.DELETE("", handlers.DeleteFacesHandler(firebaseApp.Logger, firebaseApp.DB, firebaseApp.Storage))
	facesGroup.DELETE("/overlay", handlers.DeleteFacesOverlayHandler(firebaseApp.Logger, firebaseApp.DB, firebaseApp.Storage))
	facesGroup.DELETE("/overlay/obscured", handlers.DeleteObscuredFacesOverlayHandler(firebaseApp.Logger, firebaseApp.DB, firebaseApp.Storage))

	messagingGroup := r.Group("/api/messaging")
	messagingGroup.Use(middlewares.AuthMiddleware(firebaseApp.Logger, firebaseApp.Auth))
	messagingGroup.POST("", handlers.SetMessagingRegistrationToken(firebaseApp.Logger, firebaseApp.DB))

	// Determine the port to listen on from the PORT environment variable
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port if not specified
	}

	// Start the server on the App Engine-specified port
	r.Run("0.0.0.0:" + port)
}
