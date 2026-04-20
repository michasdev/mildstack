package http

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/michasdev/mildstack/core/internal/resources/sqs/contracts"
)

const sqsQueryVersion = "2012-11-05"

var (
	ErrSQSNotOwned          = errors.New("sqs: request is not owned by the SQS native adapter")
	ErrSQSMalformedRequest  = errors.New("sqs: malformed query request")
	ErrSQSInvalidAction     = errors.New("sqs: invalid action")
	ErrSQSMissingAction     = errors.New("sqs: missing action")
	ErrSQSInvalidVersion    = errors.New("sqs: invalid version")
	ErrSQSQueuePathMismatch = errors.New("sqs: queue path mismatch")
	ErrSQSUnsupported       = errors.New("sqs: unsupported action")
)

type SQSRequestKind string

const (
	SQSRequestKindRoot  SQSRequestKind = "root"
	SQSRequestKindQueue SQSRequestKind = "queue"
)

type SQSRequestContext struct {
	Method         string
	RawPath        string
	NormalizedPath string
	Kind           SQSRequestKind
	AccountID      string
	QueueName      string
	Action         string
	Version        string
	Values         url.Values
}

func (c SQSRequestContext) QueueScoped() bool {
	return c.Kind == SQSRequestKindQueue
}

func ParseSQSRequest(req *http.Request) (SQSRequestContext, error) {
	if req == nil || req.URL == nil {
		return SQSRequestContext{}, ErrSQSMalformedRequest
	}

	pathContext, err := classifySQSPath(req.URL.Path)
	if err != nil {
		return SQSRequestContext{}, err
	}
	if pathContext.Kind == "" {
		return SQSRequestContext{}, ErrSQSNotOwned
	}

	if req.Method == http.MethodPost {
		mediaType, _, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
		if err != nil {
			return SQSRequestContext{}, fmt.Errorf("%w: %v", ErrSQSMalformedRequest, err)
		}
		if mediaType != "" && mediaType != "application/x-www-form-urlencoded" {
			return SQSRequestContext{}, ErrSQSMalformedRequest
		}
	}

	if err := req.ParseForm(); err != nil {
		return SQSRequestContext{}, fmt.Errorf("%w: %v", ErrSQSMalformedRequest, err)
	}

	action := strings.TrimSpace(req.Form.Get("Action"))
	if action == "" {
		return SQSRequestContext{}, ErrSQSMissingAction
	}
	version := strings.TrimSpace(req.Form.Get("Version"))
	if version == "" {
		return SQSRequestContext{}, ErrSQSInvalidVersion
	}
	if version != sqsQueryVersion {
		return SQSRequestContext{}, ErrSQSInvalidVersion
	}

	return SQSRequestContext{
		Method:         strings.ToUpper(strings.TrimSpace(req.Method)),
		RawPath:        normalizeRequestPath(req.URL.Path),
		NormalizedPath: pathContext.NormalizedPath,
		Kind:           pathContext.Kind,
		AccountID:      pathContext.AccountID,
		QueueName:      pathContext.QueueName,
		Action:         action,
		Version:        version,
		Values:         cloneValues(req.Form),
	}, nil
}

type sqsPathContext struct {
	Kind           SQSRequestKind
	NormalizedPath string
	AccountID      string
	QueueName      string
}

func classifySQSPath(rawPath string) (sqsPathContext, error) {
	trimmed := normalizeRequestPath(rawPath)
	if trimmed == "" || trimmed == "/" {
		return sqsPathContext{Kind: SQSRequestKindRoot, NormalizedPath: "/"}, nil
	}
	if strings.HasPrefix(trimmed, "/api/") {
		return sqsPathContext{}, ErrSQSNotOwned
	}

	segments := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(segments) != 2 {
		return sqsPathContext{}, ErrSQSNotOwned
	}
	if segments[0] == "" || segments[1] == "" {
		return sqsPathContext{}, ErrSQSQueuePathMismatch
	}

	return sqsPathContext{
		Kind:           SQSRequestKindQueue,
		NormalizedPath: "/" + strings.Join(segments, "/"),
		AccountID:      segments[0],
		QueueName:      segments[1],
	}, nil
}

func normalizeRequestPath(rawPath string) string {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return path.Clean(trimmed)
}

func cloneValues(values url.Values) url.Values {
	if values == nil {
		return nil
	}

	cloned := make(url.Values, len(values))
	for key, list := range values {
		cloned[key] = append([]string(nil), list...)
	}
	return cloned
}

func validateSQSRequestContext(ctx SQSRequestContext, spec SQSRegistrySpec) error {
	if spec.Action == "" {
		return ErrSQSInvalidAction
	}
	if ctx.Action != spec.Action {
		return ErrSQSInvalidAction
	}
	if ctx.Version != spec.Version {
		return ErrSQSInvalidVersion
	}
	switch spec.Scope {
	case contracts.ScopeRoot:
		if ctx.Kind != SQSRequestKindRoot {
			return ErrSQSQueuePathMismatch
		}
	case contracts.ScopeQueue:
		if ctx.Kind != SQSRequestKindQueue {
			return ErrSQSQueuePathMismatch
		}
	}
	return nil
}
