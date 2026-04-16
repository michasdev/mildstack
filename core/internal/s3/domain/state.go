package domain

const StateKey = "services/s3"

type State struct {
	Service string
	Buckets []Bucket
	Objects []Object
}

type Bucket struct {
	Name   string
	Region string
}

type Object struct {
	Bucket      string
	Key         string
	Size        int64
	ContentType string
}

func NewState() State {
	return State{
		Service: "s3",
		Buckets: []Bucket{
			{Name: "mildstack-assets", Region: "us-east-1"},
		},
		Objects: []Object{
			{
				Bucket:      "mildstack-assets",
				Key:         "bootstrap.txt",
				Size:        18,
				ContentType: "text/plain",
			},
		},
	}
}

func (s State) Snapshot() map[string]any {
	buckets := make([]any, 0, len(s.Buckets))
	for _, bucket := range s.Buckets {
		buckets = append(buckets, map[string]any{
			"name":   bucket.Name,
			"region": bucket.Region,
		})
	}

	objects := make([]any, 0, len(s.Objects))
	for _, object := range s.Objects {
		objects = append(objects, map[string]any{
			"bucket":       object.Bucket,
			"key":          object.Key,
			"size":         object.Size,
			"content_type": object.ContentType,
		})
	}

	return map[string]any{
		"service": s.Service,
		"buckets": buckets,
		"objects": objects,
	}
}
