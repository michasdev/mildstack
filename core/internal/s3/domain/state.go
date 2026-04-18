package domain

import (
	"crypto/md5"
	"encoding/hex"
	"sort"
	"time"
)

const StateKey = "services/s3"

type State struct {
	Service             string
	Buckets             []Bucket
	Objects             []Object
	BucketVersioning    []BucketVersioning
	VersionHistory      VersionHistory
	BucketPolicies      map[string][]byte
	BucketEncryption    map[string][]byte
	BucketLifecycle     map[string][]byte
	BucketCORS          map[string][]byte
	BucketACL           map[string][]byte
	BucketTagging       map[string][]byte
	BucketOwnership     map[string][]byte
	BucketPublicAccess  map[string][]byte
	BucketNotifications map[string][]byte
	BucketLogging       map[string][]byte
	BucketReplication   map[string]BucketReplicationConfig
	BucketObjectLock    map[string]ObjectLockConfiguration
	ObjectACLs          map[string]map[string][]byte
	ObjectTaggings      map[string]map[string][]byte
	ObjectRetention     map[string]map[string]ObjectRetention
	ObjectLegalHold     map[string]map[string]ObjectLegalHold
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

type BucketReplicationConfig struct {
	Role  string                  `json:"role"`
	Rules []BucketReplicationRule `json:"rules,omitempty"`
}

type ObjectLockConfiguration struct {
	Enabled          bool                 `json:"enabled"`
	DefaultRetention *ObjectLockRetention `json:"default_retention,omitempty"`
}

type ObjectLockRetention struct {
	Mode  string `json:"mode"`
	Days  int    `json:"days,omitempty"`
	Years int    `json:"years,omitempty"`
}

type ObjectRetention struct {
	Mode            string    `json:"mode"`
	RetainUntilDate time.Time `json:"retain_until_date"`
}

type ObjectLegalHold struct {
	Status string `json:"status"`
}

type BucketReplicationRule struct {
	ID          string                       `json:"id,omitempty"`
	Status      string                       `json:"status,omitempty"`
	Prefix      string                       `json:"prefix,omitempty"`
	Destination BucketReplicationDestination `json:"destination,omitempty"`
}

type BucketReplicationDestination struct {
	Bucket       string `json:"bucket,omitempty"`
	StorageClass string `json:"storage_class,omitempty"`
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

type ListObjectsV1Request struct {
	Bucket    string
	Prefix    string
	Delimiter string
	Marker    string
	MaxKeys   int
}

type ListObjectsV1Result struct {
	Bucket         string
	Prefix         string
	Marker         string
	Delimiter      string
	MaxKeys        int
	IsTruncated    bool
	NextMarker     string
	Objects        []Object
	CommonPrefixes []string
}

type ListObjectsV2Request struct {
	Bucket            string
	Prefix            string
	Delimiter         string
	ContinuationToken string
	StartAfter        string
	MaxKeys           int
}

type ListObjectsV2Result struct {
	Bucket                string
	Prefix                string
	Delimiter             string
	ContinuationToken     string
	StartAfter            string
	MaxKeys               int
	KeyCount              int
	IsTruncated           bool
	NextContinuationToken string
	Objects               []Object
	CommonPrefixes        []string
}

type DeleteObjectsRequest struct {
	Bucket string
	Keys   []string
	Quiet  bool
}

type DeletedObject struct {
	Key string
}

type DeleteObjectsError struct {
	Key     string
	Code    string
	Message string
}

type DeleteObjectsResult struct {
	Deleted []DeletedObject
	Errors  []DeleteObjectsError
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
			s.DeleteObjectRetention(bucket, key)
			s.DeleteObjectLegalHold(bucket, key)
			s.DeleteObjectACL(bucket, key)
			s.DeleteObjectTagging(bucket, key)
			return true
		}
	}
	s.DeleteObjectRetention(bucket, key)
	s.DeleteObjectLegalHold(bucket, key)
	s.DeleteObjectACL(bucket, key)
	s.DeleteObjectTagging(bucket, key)
	return false
}

func (s *State) DeleteBucket(name string) bool {
	for _, object := range s.Objects {
		if object.Bucket == name {
			return false
		}
	}
	if hasVersionHistoryForBucket(s.VersionHistory, name) {
		return false
	}

	for i := range s.Buckets {
		if s.Buckets[i].Name == name {
			s.Buckets = append(s.Buckets[:i], s.Buckets[i+1:]...)
			s.removeBucketVersioning(name)
			s.removeBucketGovernance(name)
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

	versioning := make([]any, 0, len(s.BucketVersioning))
	for _, setting := range s.BucketVersioning {
		versioning = append(versioning, map[string]any{
			"bucket": setting.Bucket,
			"status": setting.Status,
		})
	}

	versionHistory := make([]any, 0, len(s.VersionHistory))
	for _, record := range s.VersionHistory.sorted() {
		versionHistory = append(versionHistory, map[string]any{
			"bucket":           record.Bucket,
			"key":              record.Key,
			"version_id":       record.VersionID,
			"sequence":         record.Sequence,
			"is_delete_marker": record.IsDeleteMarker,
			"size":             record.Size,
			"content_type":     record.ContentType,
			"etag":             record.ETag,
			"last_modified":    record.LastModified,
		})
	}

	policies := bucketBodySnapshot(s.BucketPolicies)
	encryption := bucketBodySnapshot(s.BucketEncryption)
	lifecycle := bucketBodySnapshot(s.BucketLifecycle)
	cors := bucketBodySnapshot(s.BucketCORS)
	acl := bucketBodySnapshot(s.BucketACL)
	tagging := bucketBodySnapshot(s.BucketTagging)
	ownership := bucketBodySnapshot(s.BucketOwnership)
	publicAccess := bucketBodySnapshot(s.BucketPublicAccess)
	notifications := bucketBodySnapshot(s.BucketNotifications)
	logging := bucketBodySnapshot(s.BucketLogging)
	replication := bucketReplicationSnapshot(s.BucketReplication)
	objectLock := bucketObjectLockSnapshot(s.BucketObjectLock)
	objectACL := nestedBucketBodySnapshot(s.ObjectACLs)
	objectTagging := nestedBucketBodySnapshot(s.ObjectTaggings)
	objectRetention := objectRetentionSnapshot(s.ObjectRetention)
	objectLegalHold := objectLegalHoldSnapshot(s.ObjectLegalHold)

	return map[string]any{
		"service":              s.Service,
		"buckets":              buckets,
		"objects":              objects,
		"bucket_versioning":    versioning,
		"version_history":      versionHistory,
		"bucket_policies":      policies,
		"bucket_encryption":    encryption,
		"bucket_lifecycle":     lifecycle,
		"bucket_cors":          cors,
		"bucket_acl":           acl,
		"bucket_tagging":       tagging,
		"bucket_ownership":     ownership,
		"bucket_public_access": publicAccess,
		"bucket_notifications": notifications,
		"bucket_logging":       logging,
		"bucket_replication":   replication,
		"bucket_object_lock":   objectLock,
		"object_acl":           objectACL,
		"object_tagging":       objectTagging,
		"object_retention":     objectRetention,
		"object_legal_hold":    objectLegalHold,
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

func (s State) BucketPolicy(bucket string) ([]byte, bool) {
	return bucketBodyValue(s.BucketPolicies, bucket)
}

func (s State) BucketEncryptionConfig(bucket string) ([]byte, bool) {
	return bucketBodyValue(s.BucketEncryption, bucket)
}

func (s State) BucketLifecycleConfig(bucket string) ([]byte, bool) {
	return bucketBodyValue(s.BucketLifecycle, bucket)
}

func (s State) BucketCORSConfig(bucket string) ([]byte, bool) {
	return bucketBodyValue(s.BucketCORS, bucket)
}

func (s State) BucketACLConfig(bucket string) ([]byte, bool) {
	return bucketBodyValue(s.BucketACL, bucket)
}

func (s State) BucketTaggingConfig(bucket string) ([]byte, bool) {
	return bucketBodyValue(s.BucketTagging, bucket)
}

func (s State) BucketOwnershipControls(bucket string) ([]byte, bool) {
	return bucketBodyValue(s.BucketOwnership, bucket)
}

func (s State) BucketPublicAccessBlock(bucket string) ([]byte, bool) {
	return bucketBodyValue(s.BucketPublicAccess, bucket)
}

func (s State) ObjectACL(bucket, key string) ([]byte, bool) {
	return nestedBucketBodyValue(s.ObjectACLs, bucket, key)
}

func (s State) ObjectTagging(bucket, key string) ([]byte, bool) {
	return nestedBucketBodyValue(s.ObjectTaggings, bucket, key)
}

func (s State) BucketObjectLockConfig(bucket string) (ObjectLockConfiguration, bool) {
	config, ok := s.BucketObjectLock[bucket]
	if !ok {
		return ObjectLockConfiguration{}, false
	}
	return cloneObjectLockConfiguration(config), true
}

func (s State) ObjectRetentionConfig(bucket, key string) (ObjectRetention, bool) {
	objects, ok := s.ObjectRetention[bucket]
	if !ok {
		return ObjectRetention{}, false
	}
	retention, ok := objects[key]
	if !ok {
		return ObjectRetention{}, false
	}
	return retention, true
}

func (s State) ObjectLegalHoldConfig(bucket, key string) (ObjectLegalHold, bool) {
	objects, ok := s.ObjectLegalHold[bucket]
	if !ok {
		return ObjectLegalHold{}, false
	}
	hold, ok := objects[key]
	if !ok {
		return ObjectLegalHold{}, false
	}
	return hold, true
}

func (s *State) SetBucketPolicy(bucket string, body []byte) []byte {
	s.BucketPolicies = upsertBucketBodyMap(s.BucketPolicies, bucket, body)
	return cloneBytes(body)
}

func (s *State) SetBucketEncryptionConfig(bucket string, body []byte) []byte {
	s.BucketEncryption = upsertBucketBodyMap(s.BucketEncryption, bucket, body)
	return cloneBytes(body)
}

func (s *State) SetBucketLifecycleConfig(bucket string, body []byte) []byte {
	s.BucketLifecycle = upsertBucketBodyMap(s.BucketLifecycle, bucket, body)
	return cloneBytes(body)
}

func (s *State) SetBucketCORSConfig(bucket string, body []byte) []byte {
	s.BucketCORS = upsertBucketBodyMap(s.BucketCORS, bucket, body)
	return cloneBytes(body)
}

func (s *State) SetBucketACLConfig(bucket string, body []byte) []byte {
	s.BucketACL = upsertBucketBodyMap(s.BucketACL, bucket, body)
	return cloneBytes(body)
}

func (s *State) SetBucketTaggingConfig(bucket string, body []byte) []byte {
	s.BucketTagging = upsertBucketBodyMap(s.BucketTagging, bucket, body)
	return cloneBytes(body)
}

func (s *State) SetBucketOwnershipControls(bucket string, body []byte) []byte {
	s.BucketOwnership = upsertBucketBodyMap(s.BucketOwnership, bucket, body)
	return cloneBytes(body)
}

func (s *State) SetBucketPublicAccessBlock(bucket string, body []byte) []byte {
	s.BucketPublicAccess = upsertBucketBodyMap(s.BucketPublicAccess, bucket, body)
	return cloneBytes(body)
}

func (s *State) SetObjectACL(bucket, key string, body []byte) []byte {
	s.ObjectACLs = upsertNestedBucketBodyMap(s.ObjectACLs, bucket, key, body)
	return cloneBytes(body)
}

func (s *State) SetObjectTagging(bucket, key string, body []byte) []byte {
	s.ObjectTaggings = upsertNestedBucketBodyMap(s.ObjectTaggings, bucket, key, body)
	return cloneBytes(body)
}

func (s *State) SetBucketObjectLockConfig(bucket string, config ObjectLockConfiguration) ObjectLockConfiguration {
	config = cloneObjectLockConfiguration(config)
	if !config.Enabled {
		config.Enabled = true
	}
	if s.BucketObjectLock == nil {
		s.BucketObjectLock = make(map[string]ObjectLockConfiguration)
	}
	s.BucketObjectLock[bucket] = config
	return cloneObjectLockConfiguration(config)
}

func (s State) BucketNotification(bucket string) ([]byte, bool) {
	return bucketBodyValue(s.BucketNotifications, bucket)
}

func (s State) BucketLoggingConfig(bucket string) ([]byte, bool) {
	return bucketBodyValue(s.BucketLogging, bucket)
}

func (s State) BucketReplicationConfig(bucket string) (BucketReplicationConfig, bool) {
	config, ok := s.BucketReplication[bucket]
	if !ok {
		return BucketReplicationConfig{}, false
	}
	return cloneBucketReplicationConfig(config), true
}

func (s *State) SetBucketNotification(bucket string, body []byte) []byte {
	s.BucketNotifications = upsertBucketBodyMap(s.BucketNotifications, bucket, body)
	return cloneBytes(body)
}

func (s *State) SetBucketLoggingConfig(bucket string, body []byte) []byte {
	s.BucketLogging = upsertBucketBodyMap(s.BucketLogging, bucket, body)
	return cloneBytes(body)
}

func (s *State) SetBucketReplicationConfig(bucket string, config BucketReplicationConfig) BucketReplicationConfig {
	if config.Rules == nil {
		config.Rules = nil
	}
	s.BucketReplication = upsertBucketReplicationMap(s.BucketReplication, bucket, config)
	return cloneBucketReplicationConfig(config)
}

func (s *State) SetObjectRetention(bucket, key string, retention ObjectRetention) ObjectRetention {
	if s.ObjectRetention == nil {
		s.ObjectRetention = make(map[string]map[string]ObjectRetention)
	}
	if s.ObjectRetention[bucket] == nil {
		s.ObjectRetention[bucket] = make(map[string]ObjectRetention)
	}
	s.ObjectRetention[bucket][key] = retention
	return retention
}

func (s *State) SetObjectLegalHold(bucket, key string, hold ObjectLegalHold) ObjectLegalHold {
	if s.ObjectLegalHold == nil {
		s.ObjectLegalHold = make(map[string]map[string]ObjectLegalHold)
	}
	if s.ObjectLegalHold[bucket] == nil {
		s.ObjectLegalHold[bucket] = make(map[string]ObjectLegalHold)
	}
	s.ObjectLegalHold[bucket][key] = hold
	return hold
}

func (s *State) DeleteBucketPolicy(bucket string) bool {
	return deleteBucketBodyMap(&s.BucketPolicies, bucket)
}

func (s *State) DeleteBucketEncryptionConfig(bucket string) bool {
	return deleteBucketBodyMap(&s.BucketEncryption, bucket)
}

func (s *State) DeleteBucketLifecycleConfig(bucket string) bool {
	return deleteBucketBodyMap(&s.BucketLifecycle, bucket)
}

func (s *State) DeleteBucketCORSConfig(bucket string) bool {
	return deleteBucketBodyMap(&s.BucketCORS, bucket)
}

func (s *State) DeleteBucketACLConfig(bucket string) bool {
	return deleteBucketBodyMap(&s.BucketACL, bucket)
}

func (s *State) DeleteBucketTaggingConfig(bucket string) bool {
	return deleteBucketBodyMap(&s.BucketTagging, bucket)
}

func (s *State) DeleteBucketOwnershipControls(bucket string) bool {
	return deleteBucketBodyMap(&s.BucketOwnership, bucket)
}

func (s *State) DeleteBucketPublicAccessBlock(bucket string) bool {
	return deleteBucketBodyMap(&s.BucketPublicAccess, bucket)
}

func (s *State) DeleteBucketNotification(bucket string) bool {
	return deleteBucketBodyMap(&s.BucketNotifications, bucket)
}

func (s *State) DeleteBucketLoggingConfig(bucket string) bool {
	return deleteBucketBodyMap(&s.BucketLogging, bucket)
}

func (s *State) DeleteBucketReplicationConfig(bucket string) bool {
	return deleteBucketReplicationMap(&s.BucketReplication, bucket)
}

func (s *State) DeleteBucketObjectLockConfig(bucket string) bool {
	return deleteBucketObjectLockMap(&s.BucketObjectLock, bucket)
}

func (s *State) DeleteObjectRetention(bucket, key string) bool {
	return deleteNestedObjectMap(&s.ObjectRetention, bucket, key)
}

func (s *State) DeleteObjectLegalHold(bucket, key string) bool {
	return deleteNestedObjectMapHold(&s.ObjectLegalHold, bucket, key)
}

func (s *State) DeleteObjectACL(bucket, key string) bool {
	return deleteNestedBucketBodyMap(&s.ObjectACLs, bucket, key)
}

func (s *State) DeleteObjectTagging(bucket, key string) bool {
	return deleteNestedBucketBodyMap(&s.ObjectTaggings, bucket, key)
}

func (s *State) removeBucketGovernance(bucket string) {
	s.DeleteBucketPolicy(bucket)
	s.DeleteBucketEncryptionConfig(bucket)
	s.DeleteBucketLifecycleConfig(bucket)
	s.DeleteBucketCORSConfig(bucket)
	s.DeleteBucketACLConfig(bucket)
	s.DeleteBucketTaggingConfig(bucket)
	s.DeleteBucketOwnershipControls(bucket)
	s.DeleteBucketPublicAccessBlock(bucket)
	s.DeleteBucketNotification(bucket)
	s.DeleteBucketLoggingConfig(bucket)
	s.DeleteBucketReplicationConfig(bucket)
	s.DeleteBucketObjectLockConfig(bucket)
	deleteBucketNestedBodyMap(&s.ObjectACLs, bucket)
	deleteBucketNestedBodyMap(&s.ObjectTaggings, bucket)
	deleteBucketNestedObjectMap(&s.ObjectRetention, bucket)
	deleteBucketNestedObjectMapHold(&s.ObjectLegalHold, bucket)
}

func bucketBodySnapshot(values map[string][]byte) []any {
	if len(values) == 0 {
		return nil
	}

	buckets := make([]string, 0, len(values))
	for bucket := range values {
		buckets = append(buckets, bucket)
	}
	sort.Strings(buckets)

	snapshot := make([]any, 0, len(buckets))
	for _, bucket := range buckets {
		snapshot = append(snapshot, map[string]any{
			"bucket": bucket,
			"body":   string(values[bucket]),
		})
	}
	return snapshot
}

func nestedBucketBodySnapshot(values map[string]map[string][]byte) []any {
	if len(values) == 0 {
		return nil
	}

	buckets := make([]string, 0, len(values))
	for bucket := range values {
		buckets = append(buckets, bucket)
	}
	sort.Strings(buckets)

	snapshot := make([]any, 0)
	for _, bucket := range buckets {
		keys := make([]string, 0, len(values[bucket]))
		for key := range values[bucket] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			snapshot = append(snapshot, map[string]any{
				"bucket": bucket,
				"key":    key,
				"body":   string(values[bucket][key]),
			})
		}
	}
	return snapshot
}

func bucketReplicationSnapshot(values map[string]BucketReplicationConfig) []any {
	if len(values) == 0 {
		return nil
	}

	buckets := make([]string, 0, len(values))
	for bucket := range values {
		buckets = append(buckets, bucket)
	}
	sort.Strings(buckets)

	snapshot := make([]any, 0, len(buckets))
	for _, bucket := range buckets {
		config := values[bucket]
		snapshot = append(snapshot, map[string]any{
			"bucket": bucket,
			"role":   config.Role,
			"rules":  bucketReplicationRulesSnapshot(config.Rules),
		})
	}
	return snapshot
}

func bucketObjectLockSnapshot(values map[string]ObjectLockConfiguration) []any {
	if len(values) == 0 {
		return nil
	}

	buckets := make([]string, 0, len(values))
	for bucket := range values {
		buckets = append(buckets, bucket)
	}
	sort.Strings(buckets)

	snapshot := make([]any, 0, len(buckets))
	for _, bucket := range buckets {
		config := values[bucket]
		entry := map[string]any{
			"bucket":  bucket,
			"enabled": config.Enabled,
		}
		if config.DefaultRetention != nil {
			retention := map[string]any{
				"mode": config.DefaultRetention.Mode,
			}
			if config.DefaultRetention.Days > 0 {
				retention["days"] = config.DefaultRetention.Days
			}
			if config.DefaultRetention.Years > 0 {
				retention["years"] = config.DefaultRetention.Years
			}
			entry["default_retention"] = retention
		}
		snapshot = append(snapshot, entry)
	}
	return snapshot
}

func objectRetentionSnapshot(values map[string]map[string]ObjectRetention) []any {
	if len(values) == 0 {
		return nil
	}

	buckets := make([]string, 0, len(values))
	for bucket := range values {
		buckets = append(buckets, bucket)
	}
	sort.Strings(buckets)

	snapshot := make([]any, 0)
	for _, bucket := range buckets {
		keys := make([]string, 0, len(values[bucket]))
		for key := range values[bucket] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			retention := values[bucket][key]
			snapshot = append(snapshot, map[string]any{
				"bucket":            bucket,
				"key":               key,
				"mode":              retention.Mode,
				"retain_until_date": retention.RetainUntilDate,
			})
		}
	}
	return snapshot
}

func objectLegalHoldSnapshot(values map[string]map[string]ObjectLegalHold) []any {
	if len(values) == 0 {
		return nil
	}

	buckets := make([]string, 0, len(values))
	for bucket := range values {
		buckets = append(buckets, bucket)
	}
	sort.Strings(buckets)

	snapshot := make([]any, 0)
	for _, bucket := range buckets {
		keys := make([]string, 0, len(values[bucket]))
		for key := range values[bucket] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			hold := values[bucket][key]
			snapshot = append(snapshot, map[string]any{
				"bucket": bucket,
				"key":    key,
				"status": hold.Status,
			})
		}
	}
	return snapshot
}

func bucketReplicationRulesSnapshot(rules []BucketReplicationRule) []any {
	if len(rules) == 0 {
		return nil
	}

	snapshot := make([]any, 0, len(rules))
	for _, rule := range rules {
		entry := map[string]any{
			"id":     rule.ID,
			"status": rule.Status,
		}
		if rule.Prefix != "" {
			entry["prefix"] = rule.Prefix
		}
		dest := make(map[string]any)
		if rule.Destination.Bucket != "" {
			dest["bucket"] = rule.Destination.Bucket
		}
		if rule.Destination.StorageClass != "" {
			dest["storage_class"] = rule.Destination.StorageClass
		}
		if len(dest) > 0 {
			entry["destination"] = dest
		}
		snapshot = append(snapshot, entry)
	}
	return snapshot
}

func bucketBodyValue(values map[string][]byte, bucket string) ([]byte, bool) {
	body, ok := values[bucket]
	if !ok {
		return nil, false
	}
	return cloneBytes(body), true
}

func nestedBucketBodyValue(values map[string]map[string][]byte, bucket, key string) ([]byte, bool) {
	objects, ok := values[bucket]
	if !ok {
		return nil, false
	}
	body, ok := objects[key]
	if !ok {
		return nil, false
	}
	return cloneBytes(body), true
}

func upsertBucketBodyMap(values map[string][]byte, bucket string, body []byte) map[string][]byte {
	if values == nil {
		values = make(map[string][]byte)
	}
	values[bucket] = cloneBytes(body)
	return values
}

func upsertNestedBucketBodyMap(values map[string]map[string][]byte, bucket, key string, body []byte) map[string]map[string][]byte {
	if values == nil {
		values = make(map[string]map[string][]byte)
	}
	if values[bucket] == nil {
		values[bucket] = make(map[string][]byte)
	}
	values[bucket][key] = cloneBytes(body)
	return values
}

func deleteBucketBodyMap(values *map[string][]byte, bucket string) bool {
	if values == nil || *values == nil {
		return false
	}
	store := *values
	if _, ok := store[bucket]; !ok {
		return false
	}
	delete(store, bucket)
	if len(store) == 0 {
		*values = nil
	}
	return true
}

func deleteNestedBucketBodyMap(values *map[string]map[string][]byte, bucket, key string) bool {
	if values == nil || *values == nil {
		return false
	}
	store := *values
	objects, ok := store[bucket]
	if !ok {
		return false
	}
	if _, ok := objects[key]; !ok {
		return false
	}
	delete(objects, key)
	if len(objects) == 0 {
		delete(store, bucket)
	} else {
		store[bucket] = objects
	}
	if len(store) == 0 {
		*values = nil
	}
	return true
}

func deleteBucketNestedBodyMap(values *map[string]map[string][]byte, bucket string) bool {
	if values == nil || *values == nil {
		return false
	}
	store := *values
	if _, ok := store[bucket]; !ok {
		return false
	}
	delete(store, bucket)
	if len(store) == 0 {
		*values = nil
	}
	return true
}

func cloneBytes(values []byte) []byte {
	return append([]byte(nil), values...)
}

func cloneBucketReplicationConfig(config BucketReplicationConfig) BucketReplicationConfig {
	cloned := BucketReplicationConfig{
		Role:  config.Role,
		Rules: make([]BucketReplicationRule, len(config.Rules)),
	}
	for i := range config.Rules {
		cloned.Rules[i] = config.Rules[i]
	}
	return cloned
}

func cloneObjectLockConfiguration(config ObjectLockConfiguration) ObjectLockConfiguration {
	cloned := ObjectLockConfiguration{Enabled: config.Enabled}
	if config.DefaultRetention != nil {
		retention := *config.DefaultRetention
		cloned.DefaultRetention = &retention
	}
	return cloned
}

func cloneObjectRetentionMap(values map[string]map[string]ObjectRetention) map[string]map[string]ObjectRetention {
	if len(values) == 0 {
		return nil
	}

	cloned := make(map[string]map[string]ObjectRetention, len(values))
	for bucket, objects := range values {
		if len(objects) == 0 {
			continue
		}
		cloned[bucket] = make(map[string]ObjectRetention, len(objects))
		for key, retention := range objects {
			cloned[bucket][key] = retention
		}
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func cloneObjectLegalHoldMap(values map[string]map[string]ObjectLegalHold) map[string]map[string]ObjectLegalHold {
	if len(values) == 0 {
		return nil
	}

	cloned := make(map[string]map[string]ObjectLegalHold, len(values))
	for bucket, objects := range values {
		if len(objects) == 0 {
			continue
		}
		cloned[bucket] = make(map[string]ObjectLegalHold, len(objects))
		for key, hold := range objects {
			cloned[bucket][key] = hold
		}
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func upsertBucketReplicationMap(values map[string]BucketReplicationConfig, bucket string, config BucketReplicationConfig) map[string]BucketReplicationConfig {
	if values == nil {
		values = make(map[string]BucketReplicationConfig)
	}
	values[bucket] = cloneBucketReplicationConfig(config)
	return values
}

func deleteBucketReplicationMap(values *map[string]BucketReplicationConfig, bucket string) bool {
	if values == nil || *values == nil {
		return false
	}
	store := *values
	if _, ok := store[bucket]; !ok {
		return false
	}
	delete(store, bucket)
	if len(store) == 0 {
		*values = nil
	}
	return true
}

func deleteBucketObjectLockMap(values *map[string]ObjectLockConfiguration, bucket string) bool {
	if values == nil || *values == nil {
		return false
	}
	store := *values
	if _, ok := store[bucket]; !ok {
		return false
	}
	delete(store, bucket)
	if len(store) == 0 {
		*values = nil
	}
	return true
}

func deleteNestedObjectMap(values *map[string]map[string]ObjectRetention, bucket, key string) bool {
	if values == nil || *values == nil {
		return false
	}
	store := *values
	objects, ok := store[bucket]
	if !ok {
		return false
	}
	if _, ok := objects[key]; !ok {
		return false
	}
	delete(objects, key)
	if len(objects) == 0 {
		delete(store, bucket)
	} else {
		store[bucket] = objects
	}
	if len(store) == 0 {
		*values = nil
	}
	return true
}

func deleteNestedObjectMapHold(values *map[string]map[string]ObjectLegalHold, bucket, key string) bool {
	if values == nil || *values == nil {
		return false
	}
	store := *values
	objects, ok := store[bucket]
	if !ok {
		return false
	}
	if _, ok := objects[key]; !ok {
		return false
	}
	delete(objects, key)
	if len(objects) == 0 {
		delete(store, bucket)
	} else {
		store[bucket] = objects
	}
	if len(store) == 0 {
		*values = nil
	}
	return true
}

func deleteBucketNestedObjectMap(values *map[string]map[string]ObjectRetention, bucket string) bool {
	if values == nil || *values == nil {
		return false
	}
	store := *values
	if _, ok := store[bucket]; !ok {
		return false
	}
	delete(store, bucket)
	if len(store) == 0 {
		*values = nil
	}
	return true
}

func deleteBucketNestedObjectMapHold(values *map[string]map[string]ObjectLegalHold, bucket string) bool {
	if values == nil || *values == nil {
		return false
	}
	store := *values
	if _, ok := store[bucket]; !ok {
		return false
	}
	delete(store, bucket)
	if len(store) == 0 {
		*values = nil
	}
	return true
}
