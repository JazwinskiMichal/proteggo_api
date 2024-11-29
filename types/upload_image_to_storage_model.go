package types

type UploadImageToStorageModel struct {
	Id          string `json:"id"`
	FilePath    string `json:"filePath"`
	Orientation int    `json:"imageOrientation"`
}
