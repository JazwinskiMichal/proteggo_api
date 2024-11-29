package types

import "mime/multipart"

type DecodedImageInfo struct {
	File        multipart.File
	Id          string
	Extension   string
	ContentType string
}
