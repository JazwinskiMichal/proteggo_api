package tools

import (
	"image"
	"mime/multipart"

	"cloud.google.com/go/logging"
	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
)

func TryFindExifOrientation(logger *logging.Logger, file multipart.File) (int, error) {
	foundExif := false

	// Decode the EXIF data from the reader
	x, err := exif.Decode(file)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Warning,
			Payload:  "Warning decoding EXIF data, applying default image orientation.",
			Labels:   map[string]string{"error": err.Error()},
		})

		foundExif = false
	} else {
		foundExif = true
	}

	if _, err := file.Seek(0, 0); err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error resetting file pointer",
			Labels:   map[string]string{"error": err.Error()},
		})
		return 1, err
	}

	if foundExif {
		// Find the Orientation tag
		orientTag, err := x.Get(exif.Orientation)
		if err != nil {
			logger.Log(logging.Entry{
				Severity: logging.Warning,
				Payload:  "Warning reading orientation tag, applying default image orientation.",
				Labels:   map[string]string{"error": err.Error()},
			})
			return 1, nil
		}

		imageOrientation, err := orientTag.Int(0)
		if err != nil {
			logger.Log(logging.Entry{
				Severity: logging.Warning,
				Payload:  "Warning reading orientation tag, applying default image orientation.",
				Labels:   map[string]string{"error": err.Error()},
			})
			return 1, nil
		}

		return imageOrientation, nil
	} else {
		return 1, nil
	}
}

func CorrectImageOrientation(logger *logging.Logger, img image.Image, orientation int) (image.Image, error) {
	switch orientation {
	case 2:
		return imaging.FlipH(img), nil
	case 3:
		return imaging.Rotate180(img), nil
	case 4:
		return imaging.FlipV(img), nil
	case 5:
		return imaging.Transpose(img), nil
	case 6:
		return imaging.Rotate270(img), nil
	case 7:
		return imaging.Transverse(img), nil
	case 8:
		return imaging.Rotate90(img), nil
	default:
		return img, nil
	}
}
