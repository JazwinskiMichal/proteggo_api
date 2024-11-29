package types

type Post struct {
	Id                           string              `json:"id"`
	Body                         string              `json:"body"`
	CreatedAt                    string              `json:"createdAt"`
	HashTagsValues               []string            `json:"hashTagsValues"`
	HashTagsIds                  []string            `json:"hashTagsIds"`
	ImagesIds                    []string            `json:"imagesIds"`
	ImagesUrls                   []string            `json:"imagesUrls"`
	ImagesStoragePaths           []string            `json:"imagesStoragePaths"`
	FacesIds                     map[string][]string `json:"facesIds"`
	FacesUrls                    map[string][]string `json:"facesUrls"`
	FacesStoragePaths            map[string][]string `json:"facesStoragePaths"`
	OverlaysIds                  []string            `json:"overlaysIds"`
	OverlaysUrls                 []string            `json:"overlaysUrls"`
	OverlaysStoragePaths         []string            `json:"overlaysStoragePaths"`
	ObscuredOverlaysIds          []string            `json:"obscuredOverlaysIds"`
	ObscuredOverlaysUrls         []string            `json:"obscuredOverlaysUrls"`
	ObscuredOverlaysStoragePaths []string            `json:"obscuredOverlaysStoragePaths"`
}
