package types

type FaceVertices struct {
	Id       string           `json:"id"`
	ImageId  string           `json:"imageId"`
	Vertices []map[string]int `json:"vertices"`
}
