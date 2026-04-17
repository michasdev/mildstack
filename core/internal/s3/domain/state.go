package domain

import (
	"crypto/md5"
	"encoding/hex"
	"sort"
	"time"
)

const StateKey = "services/s3"

type State struct {
	Service string
	Buckets []Bucket
	Objects []Object
}

type Bucket struct {
	Name      string    `json:"name"`
	Region    string    `json:"region"`
	CreatedAt time.Time `json:"created_at"`
}

type Object struct {
	Bucket           string            `json:"bucket"`
	Key              string            `json:"key"`
	Body             []byte            `json:"body,omitempty"`
	Size             int64             `json:"size"`
	ContentType      string            `json:"content_type"`
	ETag             string            `json:"etag,omitempty"`
	LastModified     time.Time         `json:"last_modified,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	PreservedHeaders map[string]string `json:"preserved_headers,omitempty"`
}

type ListObjectsOptions struct {
	Prefix     string
	Delimiter  string
	MaxKeys    int
	StartAfter string
}

type ObjectListPage struct {
	Objects        []Object
	CommonPrefixes []string
	IsTruncated    bool
	NextMarker     string
}

func NewState() State {
	return State{
		Service: "s3",
		Buckets: []Bucket{
			{
				Name:      "mildstack-assets",
				Region:    "us-east-1",
				CreatedAt: time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC),
			},
		},
		Objects: []Object{
			{
				Bucket:       "mildstack-assets",
				Key:          "bootstrap.txt",
				Body:         bootstrapObjectBody(),
				Size:         int64(len(bootstrapObjectBody())),
				ContentType:  "text/plain",
				ETag:         etagForBody(bootstrapObjectBody()),
				LastModified: time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC),
			},
		},
	}
}

func (s State) ListBuckets() []Bucket {
	buckets := make([]Bucket, len(s.Buckets))
	copy(buckets, s.Buckets)
	sort.SliceStable(buckets, func(i, j int) bool {
		return buckets[i].Name < buckets[j].Name
	})
	return buckets
}

func (s State) ListObjects(bucket string) []Object {
	objects := make([]Object, 0, len(s.Objects))
	for _, object := range s.Objects {
		if object.Bucket == bucket {
			objects = append(objects, cloneObject(object))
		}
	}

	sort.SliceStable(objects, func(i, j int) bool {
		return objects[i].Key < objects[j].Key
	})
	return objects
}

func (s State) ListObjectPage(bucket string, options ListObjectsOptions) ObjectListPage {
	objects := s.ListObjects(bucket)
	if options.MaxKeys <= 0 {
		options.MaxKeys = 1000
	}

	allKeys := make([]string, 0, len(objects))
	objectsByKey := make(map[string]Object, len(objects))
	for _, object := range objects {
		if options.Prefix != "" && len(object.Key) >= len(options.Prefix) && object.Key[:len(options.Prefix)] != options.Prefix {
			continue
		}
		if options.Prefix != "" && len(object.Key) < len(options.Prefix) {
			continue
		}
		if object.Key <= options.StartAfter {
			continue
		}
		allKeys = append(allKeys, object.Key)
		objectsByKey[object.Key] = object
	}

	page := ObjectListPage{
		Objects:        make([]Object, 0, min(len(allKeys), options.MaxKeys)),
		CommonPrefixes: make([]string, 0),
	}
	commonPrefixes := make(map[string]struct{})
	count := 0

	for i := 0; i < len(allKeys); {
		key := allKeys[i]

		if options.Delimiter != "" {
			suffix := key[len(options.Prefix):]
			if index := indexOfDelimiter(suffix, options.Delimiter); index >= 0 {
				prefix := options.Prefix + suffix[:index+len(options.Delimiter)]
				if _, seen := commonPrefixes[prefix]; !seen {
					if count >= options.MaxKeys {
						page.IsTruncated = true
						break
					}
					commonPrefixes[prefix] = struct{}{}
					page.CommonPrefixes = append(page.CommonPrefixes, prefix)
					count++
				}
				page.NextMarker = key
				i++
				for i < len(allKeys) && hasPrefix(allKeys[i], prefix) {
					page.NextMarker = allKeys[i]
					i++
				}
				continue
			}
		}

		if count >= options.MaxKeys {
			page.IsTruncated = true
			break
		}

		page.Objects = append(page.Objects, cloneObject(objectsByKey[key]))
		page.NextMarker = key
		count++
		i++
	}

	page.CommonPrefixes = append([]string(nil), page.CommonPrefixes...)
	return page
}

func (s State) Bucket(name string) (Bucket, bool) {
	for _, bucket := range s.Buckets {
		if bucket.Name == name {
			return bucket, true
		}
	}
	return Bucket{}, false
}

func (s State) Object(bucket, key string) (Object, bool) {
	for _, object := range s.Objects {
		if object.Bucket == bucket && object.Key == key {
			return cloneObject(object), true
		}
	}
	return Object{}, false
}

func (s State) HasBucket(name string) bool {
	_, ok := s.Bucket(name)
	return ok
}

func (s State) HasObject(bucket, key string) bool {
	_, ok := s.Object(bucket, key)
	return ok
}

func (s *State) UpsertBucket(bucket Bucket) Bucket {
	if bucket.CreatedAt.IsZero() {
		bucket.CreatedAt = time.Now().UTC()
	}

	for i := range s.Buckets {
		if s.Buckets[i].Name == bucket.Name {
			if bucket.Region != "" {
				s.Buckets[i].Region = bucket.Region
			}
			if s.Buckets[i].CreatedAt.IsZero() {
				s.Buckets[i].CreatedAt = bucket.CreatedAt
			}
			return s.Buckets[i]
		}
	}

	s.Buckets = append(s.Buckets, bucket)
	return bucket
}

func (s *State) UpsertObject(object Object) Object {
	object = cloneObject(object)
	if object.Size == 0 {
		object.Size = int64(len(object.Body))
	}
	if object.ETag == "" {
		object.ETag = etagForBody(object.Body)
	}
	if object.LastModified.IsZero() {
		object.LastModified = time.Now().UTC()
	}

	for i := range s.Objects {
		if s.Objects[i].Bucket == object.Bucket && s.Objects[i].Key == object.Key {
			s.Objects[i] = cloneObject(object)
			return cloneObject(s.Objects[i])
		}
	}

	s.Objects = append(s.Objects, cloneObject(object))
	return cloneObject(object)
}

func (s *State) DeleteObject(bucket, key string) bool {
	for i := range s.Objects {
		if s.Objects[i].Bucket == bucket && s.Objects[i].Key == key {
			s.Objects = append(s.Objects[:i], s.Objects[i+1:]...)
			return true
		}
	}
	return false
}

func (s *State) DeleteBucket(name string) bool {
	for _, object := range s.Objects {
		if object.Bucket == name {
			return false
		}
	}

	for i := range s.Buckets {
		if s.Buckets[i].Name == name {
			s.Buckets = append(s.Buckets[:i], s.Buckets[i+1:]...)
			return true
		}
	}

	return false
}

func (s State) Snapshot() map[string]any {
	buckets := make([]any, 0, len(s.Buckets))
	for _, bucket := range s.ListBuckets() {
		buckets = append(buckets, map[string]any{
			"name":       bucket.Name,
			"region":     bucket.Region,
			"created_at": bucket.CreatedAt,
		})
	}

	objects := make([]any, 0, len(s.Objects))
	for _, object := range s.sortedObjects() {
		objects = append(objects, map[string]any{
			"bucket":        object.Bucket,
			"key":           object.Key,
			"size":          object.Size,
			"content_type":  object.ContentType,
			"etag":          object.ETag,
			"last_modified": object.LastModified,
		})
	}

	return map[string]any{
		"service": s.Service,
		"buckets": buckets,
		"objects": objects,
	}
}

func (s State) sortedObjects() []Object {
	objects := make([]Object, len(s.Objects))
	for i := range s.Objects {
		objects[i] = cloneObject(s.Objects[i])
	}
	sort.SliceStable(objects, func(i, j int) bool {
		if objects[i].Bucket == objects[j].Bucket {
			return objects[i].Key < objects[j].Key
		}
		return objects[i].Bucket < objects[j].Bucket
	})
	return objects
}

func bootstrapObjectBody() []byte {
	return []byte("MildStack asset v1")
}

func cloneObject(object Object) Object {
	object.Body = append([]byte(nil), object.Body...)
	object.Metadata = cloneStringMap(object.Metadata)
	object.PreservedHeaders = cloneStringMap(object.PreservedHeaders)
	return object
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func etagForBody(body []byte) string {
	sum := md5.Sum(body)
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func hasPrefix(value, prefix string) bool {
	if len(prefix) > len(value) {
		return false
	}
	return value[:len(prefix)] == prefix
}

func indexOfDelimiter(value, delimiter string) int {
	if delimiter == "" || len(delimiter) > len(value) {
		return -1
	}
	for i := 0; i <= len(value)-len(delimiter); i++ {
		if value[i:i+len(delimiter)] == delimiter {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
