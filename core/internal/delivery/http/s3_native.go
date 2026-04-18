package http

import (
	"encoding/xml"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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
		case http.MethodHead:
			h.headBucket(c, bucket)
			return true
		case http.MethodDelete:
			h.deleteBucket(c, bucket)
			return true
		case http.MethodGet:
			h.listObjects(c, bucket)
			return true
		}
	default:
		bucket := segments[0]
		key := strings.Join(segments[1:], "/")
		switch c.Request.Method {
		case http.MethodGet:
			h.getObject(c, bucket, key)
			return true
		case http.MethodPut:
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
	c.Header("Content-Type", "application/xml")
	c.XML(http.StatusOK, listBucketsResult{
		XMLName: xml.Name{Local: "ListAllMyBucketsResult"},
		XMLNS:   s3XMLNamespace,
		Owner: bucketOwner{
			ID:          "mildstack",
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

func (h s3NativeHandler) createBucket(c *gin.Context, bucketName string) {
	region := strings.TrimSpace(c.GetHeader("x-amz-bucket-region"))
	if region == "" {
		region = "us-east-1"
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

	object, err := h.service.PutObject(bucketName, objectKey, c.Request.Body, contentType)
	if err != nil {
		writeS3Error(c, err)
		return
	}

	c.Header("ETag", object.ETag)
	c.Status(http.StatusOK)
}

func (h s3NativeHandler) deleteObject(c *gin.Context, bucketName, objectKey string) {
	if err := h.service.DeleteObject(bucketName, objectKey); err != nil {
		writeS3Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
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

type listObjectEntry struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
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
