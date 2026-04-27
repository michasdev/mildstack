package http

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/michasdev/mildstack/core/internal/resources/awscontext"
	s3domain "github.com/michasdev/mildstack/core/internal/resources/s3/domain"
)

type S3NativeService interface {
	ListBuckets() []s3domain.Bucket
	ListObjects(bucket string) ([]s3domain.Object, error)
	CreateBucket(name, region string) (s3domain.Bucket, error)
	HeadBucket(name string) (s3domain.Bucket, error)
	DeleteBucket(name string) error
	GetObject(bucket, key string) (s3domain.Object, error)
	HeadObject(bucket, key string) (s3domain.Object, error)
	PutObject(bucket, key string, body io.Reader, contentType string) (s3domain.Object, error)
	DeleteObject(bucket, key string) error
}

type s3NativeMetadataWriter interface {
	PutObjectWithMetadata(bucket, key string, body io.Reader, contentType string, metadata, preservedHeaders map[string]string) (s3domain.Object, error)
}

type s3NativeListObjectsV2Service interface {
	ListObjectsV2(request s3domain.ListObjectsV2Request) (s3domain.ListObjectsV2Result, error)
}

type s3NativeListObjectsV1Service interface {
	ListObjectsV1(request s3domain.ListObjectsV1Request) (s3domain.ListObjectsV1Result, error)
}

type s3NativeDeleteObjectsService interface {
	DeleteObjects(request s3domain.DeleteObjectsRequest) (s3domain.DeleteObjectsResult, error)
}

type s3NativeCopyObjectService interface {
	CopyObject(bucket, key, sourceBucket, sourceKey string) (s3domain.Object, error)
}

type s3NativeMultipartService interface {
	CreateMultipartUpload(bucket, key, contentType string, metadata, preservedHeaders map[string]string) (s3domain.MultipartUpload, error)
	UploadPart(uploadID string, partNumber int, body []byte) (s3domain.MultipartPart, error)
	CompleteMultipartUpload(uploadID string) (s3domain.Object, error)
}

type s3NativeCopyObjectWithOptionsService interface {
	CopyObjectWithOptions(bucket, key, sourceBucket, sourceKey, metadataDirective string, metadata map[string]string) (s3domain.Object, error)
}

type s3NativeBucketPolicyService interface {
	GetBucketPolicy(bucket string) ([]byte, error)
	PutBucketPolicy(bucket string, body []byte) ([]byte, error)
	DeleteBucketPolicy(bucket string) error
}

type s3NativeBucketEncryptionService interface {
	GetBucketEncryption(bucket string) ([]byte, error)
	PutBucketEncryption(bucket string, body []byte) ([]byte, error)
	DeleteBucketEncryption(bucket string) error
}

type s3NativeBucketLifecycleService interface {
	GetBucketLifecycle(bucket string) ([]byte, error)
	PutBucketLifecycle(bucket string, body []byte) ([]byte, error)
	DeleteBucketLifecycle(bucket string) error
}

type s3NativeBucketCORSService interface {
	GetBucketCORS(bucket string) ([]byte, error)
	PutBucketCORS(bucket string, body []byte) ([]byte, error)
	DeleteBucketCORS(bucket string) error
}

type s3NativeBucketACLService interface {
	GetBucketACL(bucket string) ([]byte, error)
	PutBucketACL(bucket string, body []byte) ([]byte, error)
}

type s3NativeBucketTaggingService interface {
	GetBucketTagging(bucket string) ([]byte, error)
	PutBucketTagging(bucket string, body []byte) ([]byte, error)
	DeleteBucketTagging(bucket string) error
}

type s3NativeOwnershipControlsService interface {
	GetBucketOwnershipControls(bucket string) ([]byte, error)
	PutBucketOwnershipControls(bucket string, body []byte) ([]byte, error)
	DeleteBucketOwnershipControls(bucket string) error
}

type s3NativePublicAccessBlockService interface {
	GetPublicAccessBlock(bucket string) ([]byte, error)
	PutPublicAccessBlock(bucket string, body []byte) ([]byte, error)
	DeletePublicAccessBlock(bucket string) error
}

type s3NativeVersioningService interface {
	GetBucketVersioning(bucket string) (s3domain.BucketVersioning, error)
	PutBucketVersioning(bucket, status string) (s3domain.BucketVersioning, error)
}

const s3XMLNamespace = "http://s3.amazonaws.com/doc/2006-03-01/"
const s3ControlXMLNamespace = "http://awss3control.amazonaws.com/doc/2018-08-20/"

func RegisterS3NativeRoutes(engine *gin.Engine, service S3NativeService) {
	if engine == nil || service == nil {
		return
	}

	handler := newS3NativeHandler(service)
	engine.Use(func(c *gin.Context) {
		if handled := handler.dispatch(c); handled {
			c.Abort()
			return
		}
		c.Next()
	})
}

type s3NativeHandler struct {
	service S3NativeService
}

func newS3NativeHandler(service S3NativeService) s3NativeHandler {
	return s3NativeHandler{service: service}
}

func (h s3NativeHandler) dispatch(c *gin.Context) bool {
	path := strings.TrimSpace(c.Request.URL.Path)
	if path == "" || strings.HasPrefix(path, "/api/") {
		return false
	}
	if h.handleS3ControlTags(c, path) {
		return true
	}
	query := c.Request.URL.Query()

	trimmed := strings.Trim(path, "/")
	segments := []string{}
	if trimmed != "" {
		segments = strings.Split(trimmed, "/")
	}

	switch {
	case len(segments) == 0:
		if c.Request.Method == http.MethodGet {
			h.listBuckets(c)
			return true
		}
	case len(segments) == 1:
		bucket := segments[0]
		switch c.Request.Method {
		case http.MethodPut:
			if h.handleBucketSubresource(c, bucket, query) {
				return true
			}
			h.createBucket(c, bucket)
			return true
		case http.MethodPost:
			if hasS3QueryParam(query, "delete") {
				h.deleteObjects(c, bucket)
				return true
			}
		case http.MethodHead:
			h.headBucket(c, bucket)
			return true
		case http.MethodDelete:
			if h.handleBucketSubresource(c, bucket, query) {
				return true
			}
			h.deleteBucket(c, bucket)
			return true
		case http.MethodGet:
			if h.handleBucketSubresource(c, bucket, query) {
				return true
			}
			if strings.TrimSpace(query.Get("list-type")) == "2" {
				h.listObjectsV2(c, bucket)
				return true
			}
			h.listObjects(c, bucket)
			return true
		}
	default:
		bucket := segments[0]
		key := strings.Join(segments[1:], "/")
		switch c.Request.Method {
		case http.MethodPost:
			switch {
			case hasS3QueryParam(query, "uploads"):
				h.createMultipartUpload(c, bucket, key)
				return true
			case strings.TrimSpace(query.Get("uploadId")) != "":
				h.completeMultipartUpload(c, bucket, key)
				return true
			}
		case http.MethodGet:
			h.getObject(c, bucket, key)
			return true
		case http.MethodPut:
			switch {
			case strings.TrimSpace(query.Get("uploadId")) != "" && strings.TrimSpace(query.Get("partNumber")) != "":
				h.uploadPart(c, bucket, key)
				return true
			case strings.TrimSpace(c.GetHeader("x-amz-copy-source")) != "":
				h.copyObject(c, bucket, key)
				return true
			}
			h.putObject(c, bucket, key)
			return true
		case http.MethodHead:
			h.headObject(c, bucket, key)
			return true
		case http.MethodDelete:
			h.deleteObject(c, bucket, key)
			return true
		}
	}

	return false
}

func (h s3NativeHandler) listBuckets(c *gin.Context) {
	buckets := h.service.ListBuckets()
	aws := awscontext.Default()
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, listBucketsResult{
		XMLName: xml.Name{Local: "ListAllMyBucketsResult"},
		XMLNS:   s3XMLNamespace,
		Owner: bucketOwner{
			ID:          aws.AccountID,
			DisplayName: "mildstack",
		},
		Buckets: listBucketsContainer{
			Buckets: bucketEntriesFromDomain(buckets),
		},
	})
}

func (h s3NativeHandler) listObjects(c *gin.Context, bucketName string) {
	if service, ok := h.service.(s3NativeListObjectsV1Service); ok {
		maxKeys := 0
		if raw := strings.TrimSpace(c.Query("max-keys")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				writeS3Error(c, err)
				return
			}
			maxKeys = parsed
		}

		result, err := service.ListObjectsV1(s3domain.ListObjectsV1Request{
			Bucket:    bucketName,
			Prefix:    strings.TrimSpace(c.Query("prefix")),
			Delimiter: strings.TrimSpace(c.Query("delimiter")),
			Marker:    strings.TrimSpace(c.Query("marker")),
			MaxKeys:   maxKeys,
		})
		if err != nil {
			writeS3Error(c, err)
			return
		}

		c.Header("Content-Type", "application/xml")
		c.XML(http.StatusOK, listObjectsResult{
			XMLName:        xml.Name{Local: "ListBucketResult"},
			XMLNS:          s3XMLNamespace,
			Name:           result.Bucket,
			Prefix:         result.Prefix,
			Marker:         result.Marker,
			Delimiter:      result.Delimiter,
			MaxKeys:        result.MaxKeys,
			IsTruncated:    result.IsTruncated,
			NextMarker:     result.NextMarker,
			Contents:       listObjectEntriesFromDomain(result.Objects),
			CommonPrefixes: commonPrefixEntries(result.CommonPrefixes),
		})
		return
	}

	objects, err := h.service.ListObjects(bucketName)
	if err != nil {
		writeS3Error(c, err)
		return
	}

	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, listObjectsResult{
		XMLName:     xml.Name{Local: "ListBucketResult"},
		XMLNS:       s3XMLNamespace,
		Name:        bucketName,
		MaxKeys:     1000,
		IsTruncated: false,
		Contents:    listObjectEntriesFromDomain(objects),
	})
}

func (h s3NativeHandler) listObjectsV2(c *gin.Context, bucketName string) {
	service, ok := h.service.(s3NativeListObjectsV2Service)
	if !ok {
		writeS3Error(c, io.ErrUnexpectedEOF)
		return
	}

	maxKeys := 0
	if raw := strings.TrimSpace(c.Query("max-keys")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeS3Error(c, err)
			return
		}
		maxKeys = parsed
	}

	result, err := service.ListObjectsV2(s3domain.ListObjectsV2Request{
		Bucket:            bucketName,
		Prefix:            strings.TrimSpace(c.Query("prefix")),
		Delimiter:         strings.TrimSpace(c.Query("delimiter")),
		ContinuationToken: strings.TrimSpace(c.Query("continuation-token")),
		StartAfter:        strings.TrimSpace(c.Query("start-after")),
		MaxKeys:           maxKeys,
	})
	if err != nil {
		writeS3Error(c, err)
		return
	}

	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, listObjectsV2Result{
		XMLName:               xml.Name{Local: "ListBucketResult"},
		XMLNS:                 s3XMLNamespace,
		Name:                  result.Bucket,
		Prefix:                result.Prefix,
		Delimiter:             result.Delimiter,
		MaxKeys:               result.MaxKeys,
		KeyCount:              result.KeyCount,
		IsTruncated:           result.IsTruncated,
		ContinuationToken:     result.ContinuationToken,
		NextContinuationToken: result.NextContinuationToken,
		StartAfter:            result.StartAfter,
		Contents:              listObjectEntriesFromDomain(result.Objects),
		CommonPrefixes:        commonPrefixEntries(result.CommonPrefixes),
	})
}

func (h s3NativeHandler) createBucket(c *gin.Context, bucketName string) {
	region := strings.TrimSpace(c.GetHeader("x-amz-bucket-region"))
	if region == "" {
		region = awscontext.Default().Region
	}

	bucket, err := h.service.CreateBucket(bucketName, region)
	if err != nil {
		writeS3Error(c, err)
		return
	}

	c.Header("x-amz-bucket-region", bucket.Region)
	c.Status(http.StatusOK)
}

func (h s3NativeHandler) headBucket(c *gin.Context, bucketName string) {
	bucket, err := h.service.HeadBucket(bucketName)
	if err != nil {
		writeS3Error(c, err)
		return
	}

	c.Header("x-amz-bucket-region", bucket.Region)
	c.Status(http.StatusOK)
}

func (h s3NativeHandler) deleteBucket(c *gin.Context, bucketName string) {
	if err := h.service.DeleteBucket(bucketName); err != nil {
		writeS3Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h s3NativeHandler) getObject(c *gin.Context, bucketName, objectKey string) {
	object, err := h.service.GetObject(bucketName, objectKey)
	if err != nil {
		writeS3Error(c, err)
		return
	}

	rangeHeader := strings.TrimSpace(c.GetHeader("Range"))
	if rangeHeader != "" {
		sliced, contentRange, rangeErr := applyS3Range(rangeHeader, object.Body)
		if rangeErr != nil {
			writeS3Error(c, rangeErr)
			return
		}
		object.Body = sliced
		object.Size = int64(len(sliced))
		c.Header("Content-Range", contentRange)
		writeObjectResponse(c, object, true, http.StatusPartialContent)
		return
	}

	writeObjectResponse(c, object, true, http.StatusOK)
}

func (h s3NativeHandler) headObject(c *gin.Context, bucketName, objectKey string) {
	object, err := h.service.HeadObject(bucketName, objectKey)
	if err != nil {
		writeS3Error(c, err)
		return
	}

	writeObjectResponse(c, object, false, http.StatusOK)
}

func (h s3NativeHandler) putObject(c *gin.Context, bucketName, objectKey string) {
	contentType := strings.TrimSpace(c.GetHeader("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	metadata := metadataFromHeaders(c.Request.Header)
	preservedHeaders := preservedObjectHeaders(c.Request.Header)

	var (
		object s3domain.Object
		err    error
	)
	body := c.Request.Body
	if strings.EqualFold(strings.TrimSpace(c.GetHeader("x-amz-content-sha256")), "STREAMING-AWS4-HMAC-SHA256-PAYLOAD") {
		decoded, decodeErr := decodeAWSChunkedBody(c.Request.Body)
		if decodeErr != nil {
			writeS3Error(c, decodeErr)
			return
		}
		body = io.NopCloser(bytes.NewReader(decoded))
	}
	if writer, ok := h.service.(s3NativeMetadataWriter); ok {
		object, err = writer.PutObjectWithMetadata(bucketName, objectKey, body, contentType, metadata, preservedHeaders)
	} else {
		object, err = h.service.PutObject(bucketName, objectKey, body, contentType)
	}
	if err != nil {
		writeS3Error(c, err)
		return
	}

	c.Header("ETag", object.ETag)
	c.Status(http.StatusOK)
}

func (h s3NativeHandler) copyObject(c *gin.Context, bucketName, objectKey string) {
	sourceBucket, sourceKey, err := parseCopySource(c.GetHeader("x-amz-copy-source"))
	if err != nil {
		writeS3Error(c, err)
		return
	}

	var object s3domain.Object
	if service, ok := h.service.(s3NativeCopyObjectWithOptionsService); ok {
		object, err = service.CopyObjectWithOptions(
			bucketName,
			objectKey,
			sourceBucket,
			sourceKey,
			strings.TrimSpace(c.GetHeader("x-amz-metadata-directive")),
			metadataFromHeaders(c.Request.Header),
		)
	} else {
		service, ok := h.service.(s3NativeCopyObjectService)
		if !ok {
			writeS3Error(c, io.ErrUnexpectedEOF)
			return
		}
		object, err = service.CopyObject(bucketName, objectKey, sourceBucket, sourceKey)
	}
	if err != nil {
		writeS3Error(c, err)
		return
	}

	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, copyObjectResult{
		XMLName:      xml.Name{Local: "CopyObjectResult"},
		XMLNS:        s3XMLNamespace,
		LastModified: object.LastModified.UTC().Format(time.RFC3339),
		ETag:         object.ETag,
	})
}

func (h s3NativeHandler) deleteObject(c *gin.Context, bucketName, objectKey string) {
	if err := h.service.DeleteObject(bucketName, objectKey); err != nil {
		writeS3Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h s3NativeHandler) deleteObjects(c *gin.Context, bucketName string) {
	service, ok := h.service.(s3NativeDeleteObjectsService)
	if !ok {
		writeS3Error(c, io.ErrUnexpectedEOF)
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		writeS3Error(c, err)
		return
	}

	var payload deleteObjectsRequest
	if err := xml.Unmarshal(body, &payload); err != nil {
		writeS3Error(c, err)
		return
	}

	keys := make([]string, 0, len(payload.Objects))
	for _, object := range payload.Objects {
		key := strings.TrimSpace(object.Key)
		if key == "" {
			continue
		}
		keys = append(keys, key)
	}

	result, err := service.DeleteObjects(s3domain.DeleteObjectsRequest{
		Bucket: bucketName,
		Keys:   keys,
		Quiet:  payload.Quiet,
	})
	if err != nil {
		writeS3Error(c, err)
		return
	}

	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, deleteObjectsResult{
		XMLName: xml.Name{Local: "DeleteResult"},
		XMLNS:   s3XMLNamespace,
		Deleted: deletedObjectEntries(result.Deleted),
		Errors:  deleteObjectErrorEntries(result.Errors),
	})
}

func (h s3NativeHandler) createMultipartUpload(c *gin.Context, bucketName, objectKey string) {
	service, ok := h.service.(s3NativeMultipartService)
	if !ok {
		writeS3Error(c, io.ErrUnexpectedEOF)
		return
	}

	upload, err := service.CreateMultipartUpload(
		bucketName,
		objectKey,
		strings.TrimSpace(c.GetHeader("Content-Type")),
		metadataFromHeaders(c.Request.Header),
		preservedObjectHeaders(c.Request.Header),
	)
	if err != nil {
		writeS3Error(c, err)
		return
	}

	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, createMultipartUploadResult{
		XMLName:  xml.Name{Local: "InitiateMultipartUploadResult"},
		XMLNS:    s3XMLNamespace,
		Bucket:   upload.Bucket,
		Key:      upload.Key,
		UploadID: upload.UploadID,
	})
}

func (h s3NativeHandler) uploadPart(c *gin.Context, _, _ string) {
	service, ok := h.service.(s3NativeMultipartService)
	if !ok {
		writeS3Error(c, io.ErrUnexpectedEOF)
		return
	}

	partNumber, err := strconv.Atoi(strings.TrimSpace(c.Query("partNumber")))
	if err != nil {
		writeS3Error(c, err)
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		writeS3Error(c, err)
		return
	}

	part, err := service.UploadPart(strings.TrimSpace(c.Query("uploadId")), partNumber, body)
	if err != nil {
		writeS3Error(c, err)
		return
	}

	c.Header("ETag", part.ETag)
	c.Status(http.StatusOK)
}

func (h s3NativeHandler) completeMultipartUpload(c *gin.Context, bucketName, objectKey string) {
	service, ok := h.service.(s3NativeMultipartService)
	if !ok {
		writeS3Error(c, io.ErrUnexpectedEOF)
		return
	}

	object, err := service.CompleteMultipartUpload(strings.TrimSpace(c.Query("uploadId")))
	if err != nil {
		writeS3Error(c, err)
		return
	}

	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, completeMultipartUploadResult{
		XMLName:      xml.Name{Local: "CompleteMultipartUploadResult"},
		XMLNS:        s3XMLNamespace,
		Location:     "/" + bucketName + "/" + objectKey,
		Bucket:       bucketName,
		Key:          objectKey,
		ETag:         object.ETag,
		LastModified: object.LastModified.UTC().Format(time.RFC3339),
	})
}

type listBucketsResult struct {
	XMLName xml.Name             `xml:"ListAllMyBucketsResult"`
	XMLNS   string               `xml:"xmlns,attr"`
	Owner   bucketOwner          `xml:"Owner"`
	Buckets listBucketsContainer `xml:"Buckets"`
}

type bucketOwner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

type listBucketsContainer struct {
	Buckets []bucketEntry `xml:"Bucket"`
}

type bucketEntry struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
	BucketRegion string `xml:"BucketRegion,omitempty"`
	BucketARN    string `xml:"BucketArn,omitempty"`
}

type listObjectsResult struct {
	XMLName        xml.Name            `xml:"ListBucketResult"`
	XMLNS          string              `xml:"xmlns,attr"`
	Name           string              `xml:"Name"`
	Prefix         string              `xml:"Prefix,omitempty"`
	Marker         string              `xml:"Marker,omitempty"`
	Delimiter      string              `xml:"Delimiter,omitempty"`
	MaxKeys        int                 `xml:"MaxKeys"`
	IsTruncated    bool                `xml:"IsTruncated"`
	NextMarker     string              `xml:"NextMarker,omitempty"`
	Contents       []listObjectEntry   `xml:"Contents"`
	CommonPrefixes []commonPrefixEntry `xml:"CommonPrefixes,omitempty"`
}

type listObjectsV2Result struct {
	XMLName               xml.Name            `xml:"ListBucketResult"`
	XMLNS                 string              `xml:"xmlns,attr"`
	Name                  string              `xml:"Name"`
	Prefix                string              `xml:"Prefix,omitempty"`
	Delimiter             string              `xml:"Delimiter,omitempty"`
	MaxKeys               int                 `xml:"MaxKeys"`
	KeyCount              int                 `xml:"KeyCount"`
	IsTruncated           bool                `xml:"IsTruncated"`
	ContinuationToken     string              `xml:"ContinuationToken,omitempty"`
	NextContinuationToken string              `xml:"NextContinuationToken,omitempty"`
	StartAfter            string              `xml:"StartAfter,omitempty"`
	Contents              []listObjectEntry   `xml:"Contents"`
	CommonPrefixes        []commonPrefixEntry `xml:"CommonPrefixes,omitempty"`
}

type listObjectEntry struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

type commonPrefixEntry struct {
	Prefix string `xml:"Prefix"`
}

type deleteObjectsRequest struct {
	Quiet   bool                       `xml:"Quiet"`
	Objects []deleteObjectsRequestItem `xml:"Object"`
}

type deleteObjectsRequestItem struct {
	Key string `xml:"Key"`
}

type deleteObjectsResult struct {
	XMLName xml.Name                 `xml:"DeleteResult"`
	XMLNS   string                   `xml:"xmlns,attr"`
	Deleted []deletedObjectEntry     `xml:"Deleted,omitempty"`
	Errors  []deleteObjectErrorEntry `xml:"Error,omitempty"`
}

type deletedObjectEntry struct {
	Key string `xml:"Key"`
}

type deleteObjectErrorEntry struct {
	Key     string `xml:"Key"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

type copyObjectResult struct {
	XMLName      xml.Name `xml:"CopyObjectResult"`
	XMLNS        string   `xml:"xmlns,attr"`
	LastModified string   `xml:"LastModified"`
	ETag         string   `xml:"ETag"`
}

type createMultipartUploadResult struct {
	XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
	XMLNS    string   `xml:"xmlns,attr"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	UploadID string   `xml:"UploadId"`
}

type completeMultipartUploadResult struct {
	XMLName      xml.Name `xml:"CompleteMultipartUploadResult"`
	XMLNS        string   `xml:"xmlns,attr"`
	Location     string   `xml:"Location,omitempty"`
	Bucket       string   `xml:"Bucket"`
	Key          string   `xml:"Key"`
	ETag         string   `xml:"ETag"`
	LastModified string   `xml:"LastModified,omitempty"`
}

type versioningConfiguration struct {
	XMLName xml.Name `xml:"VersioningConfiguration"`
	XMLNS   string   `xml:"xmlns,attr,omitempty"`
	Status  string   `xml:"Status,omitempty"`
}

type tagEntry struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

type s3TaggingDocument struct {
	XMLName xml.Name   `xml:"Tagging"`
	XMLNS   string     `xml:"xmlns,attr,omitempty"`
	TagSet  []tagEntry `xml:"TagSet>Tag"`
}

type s3ControlTagResourceRequest struct {
	XMLName xml.Name   `xml:"TagResourceRequest"`
	Tags    []tagEntry `xml:"Tags>Tag"`
}

type s3ControlListTagsForResourceResponse struct {
	XMLName xml.Name   `xml:"ListTagsForResourceResult"`
	XMLNS   string     `xml:"xmlns,attr"`
	Tags    []tagEntry `xml:"Tags>Tag,omitempty"`
}

func bucketEntriesFromDomain(buckets []s3domain.Bucket) []bucketEntry {
	entries := make([]bucketEntry, len(buckets))
	for i, bucket := range buckets {
		entries[i] = bucketEntry{
			Name:         bucket.Name,
			CreationDate: bucket.CreatedAt.UTC().Format(time.RFC3339),
			BucketRegion: strings.TrimSpace(bucket.Region),
			BucketARN:    "arn:aws:s3:::" + bucket.Name,
		}
	}
	return entries
}

func listObjectEntriesFromDomain(objects []s3domain.Object) []listObjectEntry {
	entries := make([]listObjectEntry, len(objects))
	for i, object := range objects {
		entries[i] = listObjectEntry{
			Key:          object.Key,
			LastModified: object.LastModified.UTC().Format(time.RFC3339),
			ETag:         object.ETag,
			Size:         object.Size,
			StorageClass: "STANDARD",
		}
	}
	return entries
}

func commonPrefixEntries(prefixes []string) []commonPrefixEntry {
	entries := make([]commonPrefixEntry, len(prefixes))
	for i, prefix := range prefixes {
		entries[i] = commonPrefixEntry{Prefix: prefix}
	}
	return entries
}

func deletedObjectEntries(objects []s3domain.DeletedObject) []deletedObjectEntry {
	entries := make([]deletedObjectEntry, len(objects))
	for i, object := range objects {
		entries[i] = deletedObjectEntry{Key: object.Key}
	}
	return entries
}

func deleteObjectErrorEntries(errors []s3domain.DeleteObjectsError) []deleteObjectErrorEntry {
	entries := make([]deleteObjectErrorEntry, len(errors))
	for i, objectErr := range errors {
		entries[i] = deleteObjectErrorEntry{
			Key:     objectErr.Key,
			Code:    objectErr.Code,
			Message: objectErr.Message,
		}
	}
	return entries
}

type s3ErrorResponse struct {
	XMLName  xml.Name `xml:"Error"`
	Code     string   `xml:"Code"`
	Message  string   `xml:"Message"`
	Resource string   `xml:"Resource,omitempty"`
}

func writeS3Error(c *gin.Context, err error) {
	status, code, message := mapS3Error(err)
	c.Header("Content-Type", "application/xml")
	c.XML(status, s3ErrorResponse{
		Code:    code,
		Message: message,
	})
}

func mapS3Error(err error) (int, string, string) {
	if err == nil {
		return http.StatusInternalServerError, "InternalError", "internal server error"
	}

	message := err.Error()
	switch {
	case strings.Contains(message, "NoSuchBucket"):
		return http.StatusNotFound, "NoSuchBucket", message
	case strings.Contains(message, "NoSuchBucketPolicy"):
		return http.StatusNotFound, "NoSuchBucketPolicy", message
	case strings.Contains(message, "NoSuchKey"):
		return http.StatusNotFound, "NoSuchKey", message
	case strings.Contains(message, "NoSuchTagSet"):
		return http.StatusNotFound, "NoSuchTagSet", message
	case strings.Contains(message, "NoSuchLifecycleConfiguration"):
		return http.StatusNotFound, "NoSuchLifecycleConfiguration", message
	case strings.Contains(message, "NoSuchCORSConfiguration"):
		return http.StatusNotFound, "NoSuchCORSConfiguration", message
	case strings.Contains(message, "ServerSideEncryptionConfigurationNotFoundError"):
		return http.StatusNotFound, "ServerSideEncryptionConfigurationNotFoundError", message
	case strings.Contains(message, "InvalidRange"):
		return http.StatusRequestedRangeNotSatisfiable, "InvalidRange", message
	case strings.Contains(message, "InvalidBucketState"):
		return http.StatusConflict, "InvalidBucketState", message
	case strings.Contains(message, "InvalidArn"):
		return http.StatusBadRequest, "InvalidArn", message
	case strings.Contains(message, "NoSuchUpload"):
		return http.StatusNotFound, "NoSuchUpload", message
	case strings.Contains(message, "BucketNotEmpty"):
		return http.StatusConflict, "BucketNotEmpty", message
	case strings.Contains(message, "InvalidBucketName"):
		return http.StatusBadRequest, "InvalidBucketName", message
	case strings.Contains(message, "bucket name is required"):
		return http.StatusBadRequest, "InvalidBucketName", message
	case strings.Contains(message, "object key is required"):
		return http.StatusBadRequest, "InvalidObjectName", message
	default:
		return http.StatusInternalServerError, "InternalError", message
	}
}

func writeObjectResponse(c *gin.Context, object s3domain.Object, includeBody bool, status int) {
	if status == 0 {
		status = http.StatusOK
	}
	c.Header("ETag", object.ETag)
	c.Header("Last-Modified", object.LastModified.UTC().Format(http.TimeFormat))
	c.Header("Content-Type", object.ContentType)
	c.Header("Content-Length", formatContentLength(object.Size))
	c.Header("Accept-Ranges", "bytes")
	for key, value := range object.PreservedHeaders {
		if strings.TrimSpace(key) == "" {
			continue
		}
		c.Header(key, value)
	}
	for key, value := range object.Metadata {
		if strings.TrimSpace(key) == "" {
			continue
		}
		headerKey := "x-amz-meta-" + strings.ToLower(strings.TrimSpace(key))
		c.Writer.Header()[headerKey] = []string{value}
	}

	if !includeBody {
		c.Status(status)
		return
	}

	c.Data(status, object.ContentType, append([]byte(nil), object.Body...))
}

func formatContentLength(size int64) string {
	if size < 0 {
		size = 0
	}
	return strconv.FormatInt(size, 10)
}

func (h s3NativeHandler) handleBucketSubresource(c *gin.Context, bucket string, query url.Values) bool {
	switch {
	case hasS3QueryParam(query, "policy"):
		return h.handleBucketPolicy(c, bucket)
	case hasS3QueryParam(query, "tagging"):
		return h.handleBucketTagging(c, bucket)
	case hasS3QueryParam(query, "publicAccessBlock"):
		return h.handlePublicAccessBlock(c, bucket)
	case hasS3QueryParam(query, "ownershipControls"):
		return h.handleOwnershipControls(c, bucket)
	case hasS3QueryParam(query, "versioning"):
		return h.handleBucketVersioning(c, bucket)
	case hasS3QueryParam(query, "encryption"):
		return h.handleBucketEncryption(c, bucket)
	case hasS3QueryParam(query, "lifecycle"):
		return h.handleBucketLifecycle(c, bucket)
	case hasS3QueryParam(query, "cors"):
		return h.handleBucketCORS(c, bucket)
	case hasS3QueryParam(query, "acl"):
		return h.handleBucketACL(c, bucket)
	}
	return false
}

func (h s3NativeHandler) handleBucketPolicy(c *gin.Context, bucket string) bool {
	service, ok := h.service.(s3NativeBucketPolicyService)
	if !ok {
		return false
	}
	switch c.Request.Method {
	case http.MethodGet:
		body, err := service.GetBucketPolicy(bucket)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Data(http.StatusOK, "application/json", body)
	case http.MethodPut:
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		if _, err := service.PutBucketPolicy(bucket, body); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusNoContent)
	case http.MethodDelete:
		if err := service.DeleteBucketPolicy(bucket); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusNoContent)
	default:
		return false
	}
	return true
}

func (h s3NativeHandler) handleBucketTagging(c *gin.Context, bucket string) bool {
	service, ok := h.service.(s3NativeBucketTaggingService)
	if !ok {
		return false
	}
	switch c.Request.Method {
	case http.MethodGet:
		body, err := service.GetBucketTagging(bucket)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Data(http.StatusOK, "application/xml", body)
	case http.MethodPut:
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		if _, err := service.PutBucketTagging(bucket, body); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusNoContent)
	case http.MethodDelete:
		if err := service.DeleteBucketTagging(bucket); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusNoContent)
	default:
		return false
	}
	return true
}

func (h s3NativeHandler) handlePublicAccessBlock(c *gin.Context, bucket string) bool {
	service, ok := h.service.(s3NativePublicAccessBlockService)
	if !ok {
		return false
	}
	switch c.Request.Method {
	case http.MethodGet:
		body, err := service.GetPublicAccessBlock(bucket)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Data(http.StatusOK, "application/xml", body)
	case http.MethodPut:
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		if _, err := service.PutPublicAccessBlock(bucket, body); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusNoContent)
	case http.MethodDelete:
		if err := service.DeletePublicAccessBlock(bucket); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusNoContent)
	default:
		return false
	}
	return true
}

func (h s3NativeHandler) handleOwnershipControls(c *gin.Context, bucket string) bool {
	service, ok := h.service.(s3NativeOwnershipControlsService)
	if !ok {
		return false
	}
	switch c.Request.Method {
	case http.MethodGet:
		body, err := service.GetBucketOwnershipControls(bucket)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Data(http.StatusOK, "application/xml", body)
	case http.MethodPut:
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		if _, err := service.PutBucketOwnershipControls(bucket, body); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusNoContent)
	case http.MethodDelete:
		if err := service.DeleteBucketOwnershipControls(bucket); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusNoContent)
	default:
		return false
	}
	return true
}

func (h s3NativeHandler) handleBucketVersioning(c *gin.Context, bucket string) bool {
	service, ok := h.service.(s3NativeVersioningService)
	if !ok {
		return false
	}
	switch c.Request.Method {
	case http.MethodGet:
		config, err := service.GetBucketVersioning(bucket)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Header("Content-Type", "application/xml")
		c.XML(http.StatusOK, versioningConfiguration{
			XMLName: xml.Name{Local: "VersioningConfiguration"},
			XMLNS:   s3XMLNamespace,
			Status:  strings.TrimSpace(config.Status),
		})
	case http.MethodPut:
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		var config versioningConfiguration
		if err := xml.Unmarshal(body, &config); err != nil {
			writeS3Error(c, err)
			return true
		}
		if _, err := service.PutBucketVersioning(bucket, strings.TrimSpace(config.Status)); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusNoContent)
	default:
		return false
	}
	return true
}

func (h s3NativeHandler) handleBucketEncryption(c *gin.Context, bucket string) bool {
	service, ok := h.service.(s3NativeBucketEncryptionService)
	if !ok {
		return false
	}
	switch c.Request.Method {
	case http.MethodGet:
		body, err := service.GetBucketEncryption(bucket)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Data(http.StatusOK, "application/xml", body)
	case http.MethodPut:
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		if _, err := service.PutBucketEncryption(bucket, body); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusNoContent)
	case http.MethodDelete:
		if err := service.DeleteBucketEncryption(bucket); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusNoContent)
	default:
		return false
	}
	return true
}

func (h s3NativeHandler) handleBucketLifecycle(c *gin.Context, bucket string) bool {
	service, ok := h.service.(s3NativeBucketLifecycleService)
	if !ok {
		return false
	}
	switch c.Request.Method {
	case http.MethodGet:
		body, err := service.GetBucketLifecycle(bucket)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Data(http.StatusOK, "application/xml", body)
	case http.MethodPut:
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		if _, err := service.PutBucketLifecycle(bucket, body); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusNoContent)
	case http.MethodDelete:
		if err := service.DeleteBucketLifecycle(bucket); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusNoContent)
	default:
		return false
	}
	return true
}

func (h s3NativeHandler) handleBucketCORS(c *gin.Context, bucket string) bool {
	service, ok := h.service.(s3NativeBucketCORSService)
	if !ok {
		return false
	}
	switch c.Request.Method {
	case http.MethodGet:
		body, err := service.GetBucketCORS(bucket)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Data(http.StatusOK, "application/xml", body)
	case http.MethodPut:
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		if _, err := service.PutBucketCORS(bucket, body); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusNoContent)
	case http.MethodDelete:
		if err := service.DeleteBucketCORS(bucket); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusNoContent)
	default:
		return false
	}
	return true
}

func (h s3NativeHandler) handleBucketACL(c *gin.Context, bucket string) bool {
	service, ok := h.service.(s3NativeBucketACLService)
	if !ok {
		return false
	}
	switch c.Request.Method {
	case http.MethodGet:
		body, err := service.GetBucketACL(bucket)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Data(http.StatusOK, "application/xml", body)
	case http.MethodPut:
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		if _, err := service.PutBucketACL(bucket, body); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusOK)
	default:
		return false
	}
	return true
}

func (h s3NativeHandler) handleS3ControlTags(c *gin.Context, path string) bool {
	trimmed := strings.TrimSpace(path)
	if !strings.HasPrefix(trimmed, "/v20180820/tags/") {
		return false
	}
	service, ok := h.service.(s3NativeBucketTaggingService)
	if !ok {
		return false
	}
	bucket, err := parseBucketFromS3ResourceARN(strings.TrimPrefix(trimmed, "/v20180820/tags/"))
	if err != nil {
		writeS3Error(c, err)
		return true
	}
	switch c.Request.Method {
	case http.MethodGet:
		tags, err := listTagsForResource(service, bucket)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Header("Content-Type", "application/xml")
		c.XML(http.StatusOK, s3ControlListTagsForResourceResponse{
			XMLName: xml.Name{Local: "ListTagsForResourceResult"},
			XMLNS:   s3ControlXMLNamespace,
			Tags:    tags,
		})
	case http.MethodPost, http.MethodPut:
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		var request s3ControlTagResourceRequest
		if err := xml.Unmarshal(body, &request); err != nil {
			writeS3Error(c, err)
			return true
		}
		taggingBody, err := buildTaggingBody(request.Tags)
		if err != nil {
			writeS3Error(c, err)
			return true
		}
		if _, err := service.PutBucketTagging(bucket, taggingBody); err != nil {
			writeS3Error(c, err)
			return true
		}
		c.Status(http.StatusNoContent)
	default:
		return false
	}
	return true
}

func parseBucketFromS3ResourceARN(encodedARN string) (string, error) {
	decoded, err := url.PathUnescape(strings.TrimSpace(encodedARN))
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(decoded, "arn:aws:s3:::") {
		return "", fmt.Errorf("s3: InvalidArn: invalid s3 resource arn")
	}
	bucket := strings.TrimSpace(strings.TrimPrefix(decoded, "arn:aws:s3:::"))
	if bucket == "" {
		return "", fmt.Errorf("s3: InvalidArn: invalid s3 resource arn")
	}
	return bucket, nil
}

func listTagsForResource(service s3NativeBucketTaggingService, bucket string) ([]tagEntry, error) {
	body, err := service.GetBucketTagging(bucket)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchTagSet") {
			return nil, nil
		}
		return nil, err
	}
	var tagging s3TaggingDocument
	if err := xml.Unmarshal(body, &tagging); err != nil {
		return nil, err
	}
	return append([]tagEntry(nil), tagging.TagSet...), nil
}

func buildTaggingBody(tags []tagEntry) ([]byte, error) {
	payload := s3TaggingDocument{
		XMLName: xml.Name{Local: "Tagging"},
		XMLNS:   s3XMLNamespace,
		TagSet:  append([]tagEntry(nil), tags...),
	}
	body, err := xml.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), body...), nil
}

func decodeAWSChunkedBody(reader io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	decoded := make([]byte, 0, len(raw))
	rest := raw
	for {
		lineEnd := bytes.Index(rest, []byte("\r\n"))
		if lineEnd < 0 {
			return nil, fmt.Errorf("s3: InvalidRequest: malformed aws-chunked payload")
		}
		line := string(rest[:lineEnd])
		rest = rest[lineEnd+2:]

		sizeHex := line
		if semi := strings.Index(sizeHex, ";"); semi >= 0 {
			sizeHex = sizeHex[:semi]
		}
		size, parseErr := strconv.ParseInt(strings.TrimSpace(sizeHex), 16, 64)
		if parseErr != nil || size < 0 {
			return nil, fmt.Errorf("s3: InvalidRequest: malformed aws-chunked payload")
		}
		if size == 0 {
			break
		}
		if int64(len(rest)) < size+2 {
			return nil, fmt.Errorf("s3: InvalidRequest: malformed aws-chunked payload")
		}
		decoded = append(decoded, rest[:size]...)
		rest = rest[size:]
		if len(rest) < 2 || !bytes.Equal(rest[:2], []byte("\r\n")) {
			return nil, fmt.Errorf("s3: InvalidRequest: malformed aws-chunked payload")
		}
		rest = rest[2:]
	}
	return decoded, nil
}

func applyS3Range(raw string, body []byte) ([]byte, string, error) {
	if !strings.HasPrefix(raw, "bytes=") {
		return nil, "", fmt.Errorf("s3: InvalidRange: requested range not satisfiable")
	}
	spec := strings.TrimSpace(strings.TrimPrefix(raw, "bytes="))
	if strings.Contains(spec, ",") {
		return nil, "", fmt.Errorf("s3: InvalidRange: requested range not satisfiable")
	}
	dash := strings.Index(spec, "-")
	if dash < 0 {
		return nil, "", fmt.Errorf("s3: InvalidRange: requested range not satisfiable")
	}
	startRaw := strings.TrimSpace(spec[:dash])
	endRaw := strings.TrimSpace(spec[dash+1:])

	size := int64(len(body))
	if size == 0 {
		return nil, "", fmt.Errorf("s3: InvalidRange: requested range not satisfiable")
	}

	var (
		start int64
		end   int64
	)
	switch {
	case startRaw == "":
		suffix, err := strconv.ParseInt(endRaw, 10, 64)
		if err != nil || suffix <= 0 {
			return nil, "", fmt.Errorf("s3: InvalidRange: requested range not satisfiable")
		}
		if suffix > size {
			suffix = size
		}
		start = size - suffix
		end = size - 1
	case endRaw == "":
		parsedStart, err := strconv.ParseInt(startRaw, 10, 64)
		if err != nil || parsedStart < 0 || parsedStart >= size {
			return nil, "", fmt.Errorf("s3: InvalidRange: requested range not satisfiable")
		}
		start = parsedStart
		end = size - 1
	default:
		parsedStart, errStart := strconv.ParseInt(startRaw, 10, 64)
		parsedEnd, errEnd := strconv.ParseInt(endRaw, 10, 64)
		if errStart != nil || errEnd != nil || parsedStart < 0 || parsedEnd < parsedStart || parsedStart >= size {
			return nil, "", fmt.Errorf("s3: InvalidRange: requested range not satisfiable")
		}
		if parsedEnd >= size {
			parsedEnd = size - 1
		}
		start = parsedStart
		end = parsedEnd
	}

	sliced := append([]byte(nil), body[start:end+1]...)
	return sliced, fmt.Sprintf("bytes %d-%d/%d", start, end, size), nil
}

func hasS3QueryParam(values url.Values, key string) bool {
	if values == nil {
		return false
	}
	_, ok := values[key]
	return ok
}

func metadataFromHeaders(headers http.Header) map[string]string {
	metadata := map[string]string{}
	for key, values := range headers {
		lowerKey := strings.ToLower(strings.TrimSpace(key))
		if !strings.HasPrefix(lowerKey, "x-amz-meta-") {
			continue
		}
		metadata[strings.TrimPrefix(lowerKey, "x-amz-meta-")] = strings.Join(values, ",")
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func preservedObjectHeaders(headers http.Header) map[string]string {
	preserved := map[string]string{}
	for _, key := range []string{"Cache-Control", "Content-Disposition", "Content-Encoding", "Content-Language", "Expires"} {
		value := strings.TrimSpace(headers.Get(key))
		if value == "" {
			continue
		}
		preserved[key] = value
	}
	if len(preserved) == 0 {
		return nil
	}
	return preserved
}

func parseCopySource(raw string) (string, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", io.ErrUnexpectedEOF
	}
	decoded, err := url.PathUnescape(trimmed)
	if err != nil {
		return "", "", err
	}
	decoded = strings.TrimPrefix(decoded, "/")
	parts := strings.SplitN(decoded, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", io.ErrUnexpectedEOF
	}
	return parts[0], parts[1], nil
}
