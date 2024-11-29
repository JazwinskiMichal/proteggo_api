package types

type Face struct {
	Id          string                   `json:"id"`
	Url         string                   `json:"url"`
	StoragePath string                   `json:"storagePath"`
	Emotion     string                   `json:"emotion"`
	Vertices    []map[string]int         `json:"vertices"`
	Landmarks   []map[string]interface{} `json:"landmarks"`
	RollAngle   float32                  `json:"rollAngle"`
	PanAngle    float32                  `json:"panAngle"`
	TiltAngle   float32                  `json:"tiltAngle"`
	ImageId     string                   `json:"imageId"`
	CreatedAt   string                   `json:"createdAt"`
}
