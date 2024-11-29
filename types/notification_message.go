package types

type NotificationMessage struct {
	ImageId           string   `json:"imageId"`
	ImageUrl          string   `json:"imageUrl"`
	ImageStoragePath  string   `json:"imageStoragePath"`
	FacesIds          []string `json:"facesIds"`
	FacesUrls         []string `json:"facesUrls"`
	FacesStoragePaths []string `json:"facesStoragePaths"`
}
