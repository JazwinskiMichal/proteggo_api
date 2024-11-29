package tools

/*
#cgo CFLAGS: -I/usr/include/webp
#cgo LDFLAGS: -lwebp
#include <webp/encode.h>
#include <stdlib.h> // Include the standard library header for free
*/
import "C"
import (
	"errors"
	"image"
	"image/draw"
	_ "image/png" // Import for side effects, to support PNG decoding.
	"unsafe"

	"cloud.google.com/go/logging"
)

// encodeWebP encodes an image.Image to WebP format with the specified quality.
// It returns a byte slice containing the WebP image data or an error.
func EncodeWebP(logger *logging.Logger, img image.Image, quality float32) ([]byte, error) {
	if img == nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Image to encode cannot be null",
			Labels:   map[string]string{"status": "error"},
		})
		return nil, errors.New("image cannot be nil")
	}

	// Convert image.Image to *image.RGBA if it's not already.
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	if width <= 0 || height <= 0 {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Invalid image dimensions",
			Labels:   map[string]string{"status": "error"},
		})
		return nil, errors.New("invalid image dimensions")
	}

	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	if rgba == nil || rgba.Bounds().Dx() != width || rgba.Bounds().Dy() != height {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Failed to create new RGBA image",
			Labels:   map[string]string{"status": "error"},
		})
		return nil, errors.New("failed to create new RGBA image")
	}

	draw.Draw(rgba, rgba.Bounds(), img, bounds.Min, draw.Src)

	// Call WebPEncodeRGBA to encode the image.
	var output *C.uint8_t
	dataSize := C.WebPEncodeRGBA((*C.uint8_t)(&rgba.Pix[0]), C.int(width), C.int(height), C.int(rgba.Stride), C.float(quality), &output)
	if dataSize == 0 {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Failed to encode image to WebP",
			Labels:   map[string]string{"status": "error"},
		})
		return nil, errors.New("failed to encode image to WebP")
	}
	defer C.free(unsafe.Pointer(output))

	// Convert C data to Go byte slice.
	webpData := C.GoBytes(unsafe.Pointer(output), C.int(dataSize))

	if webpData == nil || len(webpData) != int(dataSize) {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Failed to convert C data to Go byte slice",
			Labels:   map[string]string{"status": "error"},
		})
		return nil, errors.New("failed to convert C data to Go byte slice")
	}

	return webpData, nil
}
