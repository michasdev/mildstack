package http

import (
	"encoding/xml"
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

const s3XMLNamespace = "http://s3.amazonaws.com/doc/2006-03-01/"

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
			h.deleteBucket(c, bucket)
			return true
		case http.MethodGet:
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
	objects, err := h.service.ListObjects(bucketName)
	if err != nil {
		writeS3Error(c, err)
		return
	}

	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, listObjectsResult{
		XMLName:  xml.Name{Local: "ListBucketResult"},
		XMLNS:    s3XMLNamespace,
		Name:     bucketName,
		Contents: listObjectEntriesFromDomain(objects),
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

	writeObjectResponse(c, object, true)
}

func (h s3NativeHandler) headObject(c *gin.Context, bucketName, objectKey string) {
	object, err := h.service.HeadObject(bucketName, objectKey)
	if err != nil {
		writeS3Error(c, err)
		return
	}

	writeObjectResponse(c, object, false)
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
	if writer, ok := h.service.(s3NativeMetadataWriter); ok {
		object, err = writer.PutObjectWithMetadata(bucketName, objectKey, c.Request.Body, contentType, metadata, preservedHeaders)
	} else {
		object, err = h.service.PutObject(bucketName, objectKey, c.Request.Body, contentType)
	}
	if err != nil {
		writeS3Error(c, err)
		return
	}

	c.Header("ETag", object.ETag)
	c.Status(http.StatusOK)
}

func (h s3NativeHandler) copyObject(c *gin.Context, bucketName, objectKey string) {
	service, ok := h.service.(s3NativeCopyObjectService)
	if !ok {
		writeS3Error(c, io.ErrUnexpectedEOF)
		return
	}

	sourceBucket, sourceKey, err := parseCopySource(c.GetHeader("x-amz-copy-source"))
	if err != nil {
		writeS3Error(c, err)
		return
	}

	object, err := service.CopyObject(bucketName, objectKey, sourceBucket, sourceKey)
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
}

type listObjectsResult struct {
	XMLName  xml.Name          `xml:"ListBucketResult"`
	XMLNS    string            `xml:"xmlns,attr"`
	Name     string            `xml:"Name"`
	Contents []listObjectEntry `xml:"Contents"`
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

func bucketEntriesFromDomain(buckets []s3domain.Bucket) []bucketEntry {
	entries := make([]bucketEntry, len(buckets))
	for i, bucket := range buckets {
		entries[i] = bucketEntry{
			Name:         bucket.Name,
			CreationDate: bucket.CreatedAt.UTC().Format(time.RFC3339),
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
	case strings.Contains(message, "NoSuchKey"):
		return http.StatusNotFound, "NoSuchKey", message
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

func writeObjectResponse(c *gin.Context, object s3domain.Object, includeBody bool) {
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
		c.Header("x-amz-meta-"+strings.ToLower(key), value)
	}

	if !includeBody {
		c.Status(http.StatusOK)
		return
	}

	c.Data(http.StatusOK, object.ContentType, append([]byte(nil), object.Body...))
}

func formatContentLength(size int64) string {
	if size < 0 {
		size = 0
	}
	return strconv.FormatInt(size, 10)
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
