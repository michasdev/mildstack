package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
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
	TargetStyle    bool
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
		if mediaType != "" && mediaType != "application/x-www-form-urlencoded" && mediaType != "application/x-amz-json-1.0" {
			return SQSRequestContext{}, ErrSQSMalformedRequest
		}
	}

	if err := req.ParseForm(); err != nil {
		return SQSRequestContext{}, fmt.Errorf("%w: %v", ErrSQSMalformedRequest, err)
	}

	values := cloneValues(req.Form)
	if values == nil {
		values = url.Values{}
	}

	targetAction, jsonMode, err := parseSQSTarget(req.Header.Get("X-Amz-Target"))
	if err != nil {
		return SQSRequestContext{}, err
	}
	if jsonMode {
		if req.Body != nil {
			bodyBytes, _ := io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			if len(bodyBytes) > 0 {
				var payload map[string]any
				if err := json.Unmarshal(bodyBytes, &payload); err == nil {
					mergeJSONValues(values, payload)
				}
			}
		}
		if targetAction != "" {
			values.Set("Action", targetAction)
		}
		if values.Get("Version") == "" {
			values.Set("Version", sqsQueryVersion)
		}
	}

	if strings.TrimSpace(values.Get("Action")) == "" || strings.TrimSpace(values.Get("Version")) == "" {
		if req.Body != nil {
			bodyBytes, _ := io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			if parsedBody, err := url.ParseQuery(string(bodyBytes)); err == nil {
				mergeValues(values, parsedBody)
			}
		}
		mergeValues(values, req.URL.Query())
	}

	action := strings.TrimSpace(values.Get("Action"))
	if action == "" {
		return SQSRequestContext{}, ErrSQSMissingAction
	}
	version := strings.TrimSpace(values.Get("Version"))
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
		TargetStyle:    jsonMode,
		Values:         values,
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

func mergeValues(dst, src url.Values) {
	if dst == nil || len(src) == 0 {
		return
	}

	for key, values := range src {
		if len(values) == 0 {
			continue
		}
		dst[key] = append([]string(nil), values...)
	}
}

func mergeJSONValues(dst url.Values, payload map[string]any) {
	if dst == nil || len(payload) == 0 {
		return
	}

	for key, value := range payload {
		switch typed := value.(type) {
		case string:
			dst.Set(key, typed)
		case bool:
			dst.Set(key, strconv.FormatBool(typed))
		case float64:
			dst.Set(key, strconv.FormatFloat(typed, 'f', -1, 64))
		case map[string]any:
			if strings.EqualFold(key, "Attributes") {
				keys := make([]string, 0, len(typed))
				for attrName := range typed {
					keys = append(keys, attrName)
				}
				sort.Strings(keys)
				for i, attrName := range keys {
					dst.Set(fmt.Sprintf("Attribute.%d.Name", i+1), attrName)
					dst.Set(fmt.Sprintf("Attribute.%d.Value.StringValue", i+1), fmt.Sprint(typed[attrName]))
				}
				continue
			}
			for nestedKey, nestedValue := range typed {
				dst.Set(fmt.Sprintf("%s.%s", key, nestedKey), fmt.Sprint(nestedValue))
			}
		case []any:
			if strings.EqualFold(key, "AttributeNames") {
				for i, item := range typed {
					dst.Set(fmt.Sprintf("AttributeName.%d", i+1), fmt.Sprint(item))
				}
			}
		default:
			dst.Set(key, fmt.Sprint(value))
		}
	}
}

func parseSQSTarget(raw string) (string, bool, error) {
	target := strings.TrimSpace(raw)
	if target == "" {
		return "", false, nil
	}
	if !strings.HasPrefix(target, "AmazonSQS.") {
		return "", false, fmt.Errorf("sqs: X-Amz-Target %q must start with %q", target, "AmazonSQS.")
	}
	action := strings.TrimSpace(strings.TrimPrefix(target, "AmazonSQS."))
	if action == "" {
		return "", false, fmt.Errorf("sqs: X-Amz-Target %q is missing an operation name", target)
	}
	return action, true, nil
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

func queueOwnerAccountID(values url.Values) string {
	return strings.TrimSpace(values.Get("QueueOwnerAWSAccountId"))
}

func queueNamePrefix(values url.Values) string {
	return strings.TrimSpace(values.Get("QueueNamePrefix"))
}

func queueNextToken(values url.Values) string {
	return strings.TrimSpace(values.Get("NextToken"))
}

func queueMaxResults(values url.Values) int {
	raw := strings.TrimSpace(values.Get("MaxResults"))
	if raw == "" {
		return 0
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func queueAttributeNames(values url.Values) []string {
	names := make([]string, 0)
	for key, list := range values {
		if !strings.HasPrefix(key, "AttributeName") || len(list) == 0 {
			continue
		}
		for _, value := range list {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				names = append(names, trimmed)
			}
		}
	}
	sort.Strings(names)
	return names
}

func queueAttributesFromValues(values url.Values) map[string]string {
	type queueAttribute struct {
		name  string
		value string
	}

	byIndex := make(map[int]*queueAttribute)
	for key, list := range values {
		if !strings.HasPrefix(key, "Attribute.") || len(list) == 0 {
			continue
		}

		segments := strings.Split(key, ".")
		if len(segments) < 3 {
			continue
		}

		index, err := strconv.Atoi(segments[1])
		if err != nil {
			continue
		}

		entry, ok := byIndex[index]
		if !ok {
			entry = &queueAttribute{}
			byIndex[index] = entry
		}

		switch segments[2] {
		case "Name":
			entry.name = strings.TrimSpace(list[0])
		case "Value":
			if len(segments) >= 4 && segments[3] == "StringValue" {
				entry.value = list[0]
			}
		}
	}

	if len(byIndex) == 0 {
		return nil
	}

	indices := make([]int, 0, len(byIndex))
	for index := range byIndex {
		indices = append(indices, index)
	}
	sort.Ints(indices)

	attributes := make(map[string]string, len(indices))
	for _, index := range indices {
		entry := byIndex[index]
		if entry == nil || entry.name == "" {
			continue
		}
		attributes[entry.name] = entry.value
	}
	if len(attributes) == 0 {
		return nil
	}
	return attributes
}
