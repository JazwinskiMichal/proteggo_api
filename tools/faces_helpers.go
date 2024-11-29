package tools

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log"
	"proteggo_api/types"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	vision "cloud.google.com/go/vision/v2/apiv1"
	"cloud.google.com/go/vision/v2/apiv1/visionpb"
	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
)

func CreateFaceImage(img image.Image, vertices []*visionpb.Vertex) *image.NRGBA {
	// Copy the image
	imgCopy := image.NewRGBA(img.Bounds())
	draw.Draw(imgCopy, imgCopy.Bounds(), img, image.Point{}, draw.Src)

	// Crop the image copy to the face
	topLeft := vertices[0]
	bottomRight := vertices[2]
	return imaging.Crop(imgCopy, image.Rect(int(topLeft.GetX()), int(topLeft.GetY()), int(bottomRight.GetX()), int(bottomRight.GetY())))
}

func DetectFaceEmotions(face *visionpb.FaceAnnotation) string {
	// Get the emotion with the highest likelihood
	emotions := map[string]int{
		"anger":    int(face.AngerLikelihood),
		"joy":      int(face.JoyLikelihood),
		"surprise": int(face.SurpriseLikelihood),
		"sorrow":   int(face.SorrowLikelihood),
	}

	maxLikelihood := 0
	maxEmotion := ""

	for emotion, likelihood := range emotions {
		if likelihood > maxLikelihood {
			maxLikelihood = likelihood
			maxEmotion = emotion
		}
	}

	return maxEmotion
}

func DetectFacesInImage(ctx *gin.Context, firestoreClient *firestore.Client, storage *storage.Client, img image.Image, maxResults int32) ([]types.Face, error) {
	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// Encode the image.Image (img) as a JPEG into a bytes.Buffer
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		return nil, err
	}

	// Create the request.
	req := &visionpb.BatchAnnotateImagesRequest{
		Requests: []*visionpb.AnnotateImageRequest{
			{
				Image: &visionpb.Image{
					Content: buf.Bytes(),
				},
				Features: []*visionpb.Feature{
					{
						Type:       visionpb.Feature_FACE_DETECTION,
						MaxResults: maxResults,
					},
				},
			},
		},
	}

	// Execute the request.
	resp, err := client.BatchAnnotateImages(ctx, req)
	if err != nil {
		return nil, err
	}

	// Handle the response.
	faces := []types.Face{}

	for _, res := range resp.Responses {
		if err := res.GetError(); err != nil {
			log.Printf("Response error: %v", err)
			continue
		}

		if len(res.FaceAnnotations) == 0 {
			continue
		}

		// Print detected face annotations.
		for _, face := range res.FaceAnnotations {
			// Create image with only the face
			vertices := face.GetBoundingPoly().GetVertices()
			faceImg := CreateFaceImage(img, vertices)

			// Generate the URL for the face image
			var buf bytes.Buffer
			err := jpeg.Encode(&buf, faceImg, nil)
			if err != nil {
				return nil, err
			}
			faceImgBytes := buf.Bytes()

			// Generate random name for the face image
			randomFaceName, err := GenerateRandomName()
			if err != nil {
				return nil, err
			}

			faceStoragePath := types.FIREBASE_STORAGE_FACES_FOLDER + randomFaceName + ".jpg"
			url, err := GenerateImageUrl(ctx, firestoreClient, storage, faceImgBytes, types.FIREBASE_STORAGE_BUCKET, faceStoragePath)

			if err != nil {
				return nil, err
			}

			// Detect face emotions
			emotion := DetectFaceEmotions(face)

			// Get the bounding box coordinates
			verticesData := make([]map[string]int, len(vertices))
			for i, vertex := range vertices {
				verticesData[i] = map[string]int{
					"x": int(vertex.GetX()),
					"y": int(vertex.GetY()),
				}
			}

			// Get the landmarks
			landmarks := face.GetLandmarks()
			landmarkData := make([]map[string]interface{}, len(landmarks))
			for i, landmark := range landmarks {
				landmarkData[i] = map[string]interface{}{
					"type": landmark.GetType().String(),
					"position": map[string]float32{
						"x": landmark.GetPosition().GetX(),
						"y": landmark.GetPosition().GetY(),
						"z": landmark.GetPosition().GetZ(),
					},
				}
			}

			// Get the angles
			rollAngle := face.GetRollAngle()
			panAngle := face.GetPanAngle()
			tiltAngle := face.GetTiltAngle()

			// Save these coordinates along with the face data
			faceName, err := GenerateRandomName()
			if err != nil {
				return nil, err
			}

			faces = append(faces, types.Face{
				Id:          faceName,
				Url:         url,
				StoragePath: faceStoragePath,
				Emotion:     emotion,
				Vertices:    verticesData,
				Landmarks:   landmarkData,
				RollAngle:   rollAngle,
				PanAngle:    panAngle,
				TiltAngle:   tiltAngle,
			})
		}
	}

	return faces, nil
}

func DrawBordersAroundFaces(ctx *gin.Context, firestoreClient *firestore.Client, storage *storage.Client, imageId string, imageWidth int, imageHeight int, facesVertices []types.FaceVertices) (string, error) {
	// Create a new image with the same dimensions as the original, but with a transparent background
	imgBounds := image.Rect(0, 0, imageWidth, imageHeight)
	imgCopy := image.NewNRGBA(imgBounds)
	draw.Draw(imgCopy, imgCopy.Bounds(), image.Transparent, image.Point{}, draw.Src)

	// Create a red rectangle for each face
	for _, face := range facesVertices {
		// Get the vertices of the face
		vertices := face.Vertices

		// Define the thickness of the border
		borderThickness := 8

		// Create rectangles for each side of the border
		topBorder := image.Rect(vertices[0]["x"], vertices[0]["y"], vertices[2]["x"], vertices[0]["y"]+borderThickness)
		bottomBorder := image.Rect(vertices[0]["x"], vertices[2]["y"]-borderThickness, vertices[2]["x"], vertices[2]["y"])
		leftBorder := image.Rect(vertices[0]["x"], vertices[0]["y"], vertices[0]["x"]+borderThickness, vertices[2]["y"])
		rightBorder := image.Rect(vertices[2]["x"]-borderThickness, vertices[0]["y"], vertices[2]["x"], vertices[2]["y"])

		// Draw the borders onto the copy of the image
		draw.Draw(imgCopy, topBorder, &image.Uniform{color.RGBA{255, 0, 0, 255}}, image.Point{}, draw.Over)
		draw.Draw(imgCopy, bottomBorder, &image.Uniform{color.RGBA{255, 0, 0, 255}}, image.Point{}, draw.Over)
		draw.Draw(imgCopy, leftBorder, &image.Uniform{color.RGBA{255, 0, 0, 255}}, image.Point{}, draw.Over)
		draw.Draw(imgCopy, rightBorder, &image.Uniform{color.RGBA{255, 0, 0, 255}}, image.Point{}, draw.Over)
	}

	// Generate the URL for the face image
	var buf bytes.Buffer
	err := png.Encode(&buf, imgCopy)
	if err != nil {
		return "", err
	}
	overlayImgBytes := buf.Bytes()

	overlayStoragePath := types.FIREBASE_STORAGE_FACES_OVERLAY_FOLDER + imageId + ".png"
	overlayUrl, err := GenerateImageUrl(ctx, firestoreClient, storage, overlayImgBytes, types.FIREBASE_STORAGE_BUCKET, overlayStoragePath)
	if err != nil {
		return "", err
	}

	return overlayUrl, nil
}

func ObscureFacesInImage(ctx *gin.Context, firestoreClient *firestore.Client, storage *storage.Client, imageId string, imageWidth int, imageHeight int, facesToObscure []types.FaceVertices) (string, error) {
	// Create a new image with the same dimensions as the original, but with a transparent background
	imgBounds := image.Rect(0, 0, imageWidth, imageHeight)
	imgCopy := image.NewNRGBA(imgBounds)
	draw.Draw(imgCopy, imgCopy.Bounds(), image.Transparent, image.Point{}, draw.Src)

	// Create a black rectangle for each face
	for _, face := range facesToObscure {
		// Get the vertices of the face
		vertices := face.Vertices

		// Create a black rectangle for the face
		black := color.RGBA{0, 0, 0, 255}
		draw.Draw(imgCopy, image.Rect(vertices[0]["x"], vertices[0]["y"], vertices[2]["x"], vertices[2]["y"]), &image.Uniform{black}, image.Point{}, draw.Over)
	}

	// Generate the URL for the face image
	var buf bytes.Buffer
	err := png.Encode(&buf, imgCopy)
	if err != nil {
		return "", err
	}
	overlayImgBytes := buf.Bytes()

	overlayStoragePath := types.FIREBASE_STORAGE_TEMP_FOLDER + imageId + ".png"
	overlayUrl, err := GenerateImageUrl(ctx, firestoreClient, storage, overlayImgBytes, types.FIREBASE_STORAGE_BUCKET, overlayStoragePath)
	if err != nil {
		return "", err
	}

	return overlayUrl, nil
}

func GetFacesVertices(imageId string, facesIds []string, c *gin.Context, firestoreClient *firestore.Client) ([]types.FaceVertices, error) {
	if len(facesIds) == 0 {
		return nil, nil
	}

	// Get the faces from Firestore
	query := firestoreClient.Collection(types.FIREBASE_FACES_COLLECTION).
		Where(types.FIREBASE_FACES_FIELDS_IMAGE_ID, "==", imageId).
		Where(types.FIREBASE_FACES_FIELDS_ID, "in", facesIds)

	iter := query.Documents(c)

	var faceVertices []types.FaceVertices

	// Iterate over the faces and get the vertices
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		id, ok := doc.Data()[types.FIREBASE_FACES_FIELDS_ID].(string)
		if !ok {
			return nil, errors.New("error casting face id to string")
		}

		imageId, ok := doc.Data()[types.FIREBASE_FACES_FIELDS_IMAGE_ID].(string)
		if !ok {
			return nil, errors.New("error casting face image id to string")
		}

		verticesData := doc.Data()[types.FIREBASE_FACES_FIELDS_VERTICES]
		vertices, ok := verticesData.([]interface{})
		if !ok {
			return nil, errors.New("error casting vertices to []interface{}")
		}

		var verticesMap []map[string]int

		for _, vertex := range vertices {
			vertexMap, ok := vertex.(map[string]interface{})
			if !ok {
				return nil, errors.New("error casting vertex to map[string]interface{}")
			}

			x, ok := vertexMap["x"].(int64) // Firestore uses int64 for numbers
			if !ok {
				return nil, errors.New("error casting x to int64")
			}

			y, ok := vertexMap["y"].(int64) // Firestore uses int64 for numbers
			if !ok {
				return nil, errors.New("error casting y to int64")
			}

			verticesMap = append(verticesMap, map[string]int{
				"x": int(x),
				"y": int(y),
			})
		}

		faceVertices = append(faceVertices, types.FaceVertices{
			Id:       id,
			ImageId:  imageId,
			Vertices: verticesMap,
		})
	}

	return faceVertices, nil
}
