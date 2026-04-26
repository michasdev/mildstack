package http

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
)

const snsAPIVersion = "2010-03-31"

var (
	ErrSNSNotOwned         = fmt.Errorf("sns: request is not owned by the SNS native adapter")
	ErrSNSMalformedRequest = fmt.Errorf("sns: malformed query request")
	ErrSNSMissingAction    = fmt.Errorf("sns: action is required")
	ErrSNSMissingVersion   = fmt.Errorf("sns: version is required")
	ErrSNSInvalidVersion   = fmt.Errorf("sns: version is invalid")
)

// SNSRequestContext stores parsed AWS Query metadata for SNS calls.
type SNSRequestContext struct {
	Method         string
	RawPath        string
	NormalizedPath string
	Action         string
	Version        string
	Values         url.Values
}

func ParseSNSRequest(req *http.Request, registry SNSRegistry) (SNSRequestContext, error) {
	if req == nil || req.URL == nil {
		return SNSRequestContext{}, ErrSNSMalformedRequest
	}

	rawPath := normalizeSNSPath(req.URL.Path)
	if rawPath == "" {
		rawPath = "/"
	}
	if rawPath != "/" {
		return SNSRequestContext{}, ErrSNSNotOwned
	}
	if strings.HasPrefix(rawPath, "/api/") {
		return SNSRequestContext{}, ErrSNSNotOwned
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method != http.MethodGet && method != http.MethodPost {
		return SNSRequestContext{}, ErrSNSNotOwned
	}

	if method == http.MethodPost {
		mediaType, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
		if err != nil {
			return SNSRequestContext{}, fmt.Errorf("%w: %v", ErrSNSMalformedRequest, err)
		}
		if mediaType != "" && mediaType != "application/x-www-form-urlencoded" {
			return SNSRequestContext{}, ErrSNSMalformedRequest
		}
	}

	if err := req.ParseForm(); err != nil {
		return SNSRequestContext{}, fmt.Errorf("%w: %v", ErrSNSMalformedRequest, err)
	}

	values := cloneValues(req.Form)
	if values == nil {
		values = url.Values{}
	}
	action := strings.TrimSpace(values.Get("Action"))
	version := strings.TrimSpace(values.Get("Version"))
	if !shouldOwnSNSRequest(action, version, registry) {
		return SNSRequestContext{}, ErrSNSNotOwned
	}

	if action == "" {
		return SNSRequestContext{}, ErrSNSMissingAction
	}
	if version == "" {
		return SNSRequestContext{}, ErrSNSMissingVersion
	}
	if version != snsAPIVersion {
		return SNSRequestContext{}, ErrSNSInvalidVersion
	}

	if _, ok := registry.Lookup(action); !ok {
		return SNSRequestContext{}, ErrSNSInvalidAction
	}

	return SNSRequestContext{
		Method:         method,
		RawPath:        rawPath,
		NormalizedPath: "/",
		Action:         action,
		Version:        version,
		Values:         values,
	}, nil
}

func shouldOwnSNSRequest(action, version string, registry SNSRegistry) bool {
	action = strings.TrimSpace(action)
	version = strings.TrimSpace(version)

	if action == "" && version == "" {
		return false
	}
	if version == snsAPIVersion {
		return true
	}
	if action != "" {
		_, ok := registry.Lookup(action)
		return ok
	}
	return false
}

func normalizeSNSPath(rawPath string) string {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return path.Clean(trimmed)
}
