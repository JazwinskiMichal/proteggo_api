package types

type Image struct {
	Id          string   `json:"id"`
	Url         string   `json:"url"`
	StoragePath string   `json:"storagePath"`
	CreatedAt   string   `json:"createdAt"`
	PostsIds    []string `json:"postsIds"`
	FacesIds    []string `json:"facesIds"`
}
