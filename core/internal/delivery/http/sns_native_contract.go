package http

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
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
	if strings.TrimSpace(values.Get("Version")) == "" && strings.EqualFold(strings.TrimSpace(values.Get("Action")), "PublishBatch") {
		values.Set("Version", snsAPIVersion)
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

func snsPublishRequestFromValues(values url.Values) domain.PublishRequest {
	return domain.PublishRequest{
		TopicARN:               strings.TrimSpace(values.Get("TopicArn")),
		TargetARN:              strings.TrimSpace(values.Get("TargetArn")),
		PhoneNumber:            strings.TrimSpace(values.Get("PhoneNumber")),
		Message:                values.Get("Message"),
		Subject:                values.Get("Subject"),
		MessageStructure:       strings.TrimSpace(values.Get("MessageStructure")),
		MessageAttributes:      snsMessageAttributesFromValues(values, "MessageAttributes.entry."),
		MessageGroupID:         strings.TrimSpace(values.Get("MessageGroupId")),
		MessageDeduplicationID: strings.TrimSpace(values.Get("MessageDeduplicationId")),
	}
}

func snsPublishBatchRequestFromValues(values url.Values) domain.PublishBatchRequest {
	entriesByIndex := map[int]url.Values{}
	for key, bucket := range values {
		if len(bucket) == 0 {
			continue
		}
		if !strings.HasPrefix(key, "PublishBatchRequestEntries.member.") {
			continue
		}

		suffix := strings.TrimPrefix(key, "PublishBatchRequestEntries.member.")
		parts := strings.Split(suffix, ".")
		if len(parts) < 2 {
			continue
		}
		index, err := strconv.Atoi(parts[0])
		if err != nil || index <= 0 {
			continue
		}
		entryValues := entriesByIndex[index]
		if entryValues == nil {
			entryValues = url.Values{}
			entriesByIndex[index] = entryValues
		}
		entryValues.Set(strings.Join(parts[1:], "."), bucket[0])
	}

	indices := make([]int, 0, len(entriesByIndex))
	for index := range entriesByIndex {
		indices = append(indices, index)
	}
	sort.Ints(indices)

	entries := make([]domain.PublishBatchRequestEntry, 0, len(indices))
	for _, index := range indices {
		entryValues := entriesByIndex[index]
		entries = append(entries, domain.PublishBatchRequestEntry{
			ID:                     strings.TrimSpace(entryValues.Get("Id")),
			Message:                entryValues.Get("Message"),
			Subject:                entryValues.Get("Subject"),
			MessageStructure:       strings.TrimSpace(entryValues.Get("MessageStructure")),
			MessageAttributes:      snsMessageAttributesFromValues(entryValues, "MessageAttributes.entry."),
			MessageGroupID:         strings.TrimSpace(entryValues.Get("MessageGroupId")),
			MessageDeduplicationID: strings.TrimSpace(entryValues.Get("MessageDeduplicationId")),
		})
	}

	return domain.PublishBatchRequest{
		TopicARN: strings.TrimSpace(values.Get("TopicArn")),
		Entries:  entries,
	}
}

func snsMessageAttributesFromValues(values url.Values, prefix string) map[string]domain.MessageAttributeValue {
	type parsedAttribute struct {
		name  string
		value domain.MessageAttributeValue
	}

	byIndex := map[int]*parsedAttribute{}
	for rawKey, bucket := range values {
		if !strings.HasPrefix(rawKey, prefix) {
			continue
		}
		if len(bucket) == 0 {
			continue
		}

		suffix := strings.TrimPrefix(rawKey, prefix)
		parts := strings.Split(suffix, ".")
		if len(parts) < 2 {
			continue
		}
		index, err := strconv.Atoi(parts[0])
		if err != nil || index <= 0 {
			continue
		}

		entry := byIndex[index]
		if entry == nil {
			entry = &parsedAttribute{}
			byIndex[index] = entry
		}

		switch strings.ToLower(parts[1]) {
		case "name":
			entry.name = strings.TrimSpace(bucket[0])
		case "value":
			if len(parts) < 3 {
				continue
			}
			switch strings.ToLower(parts[2]) {
			case "datatype":
				entry.value.DataType = strings.TrimSpace(bucket[0])
			case "stringvalue":
				entry.value.StringValue = bucket[0]
			case "binaryvalue":
				entry.value.BinaryValue = bucket[0]
			}
		}
	}

	if len(byIndex) == 0 {
		return map[string]domain.MessageAttributeValue{}
	}

	indices := make([]int, 0, len(byIndex))
	for index := range byIndex {
		indices = append(indices, index)
	}
	sort.Ints(indices)

	result := make(map[string]domain.MessageAttributeValue, len(indices))
	for _, index := range indices {
		entry := byIndex[index]
		if entry == nil || strings.TrimSpace(entry.name) == "" {
			continue
		}
		result[entry.name] = entry.value
	}
	return result
}

func snsPermissionAWSAccountIDsFromValues(values url.Values) []string {
	return snsUniqueValues(
		append(
			snsMemberValues(values, "AWSAccountId"),
			snsMemberValues(values, "AWSAccountIds")...,
		),
	)
}

func snsPermissionActionNamesFromValues(values url.Values) []string {
	return snsUniqueValues(
		append(
			snsMemberValues(values, "ActionName"),
			snsMemberValues(values, "ActionNames")...,
		),
	)
}

func snsTagKeysFromValues(values url.Values) []string {
	return snsUniqueValues(
		append(
			snsMemberValues(values, "TagKeys"),
			snsMemberValues(values, "TagKey")...,
		),
	)
}

func snsSMSAttributeNamesFromValues(values url.Values) []string {
	return snsUniqueValues(
		append(
			append(
				snsMemberValues(values, "attributes"),
				snsMemberValues(values, "AttributeNames")...,
			),
			snsMemberValues(values, "AttributeName")...,
		),
	)
}

func snsPhoneNumberFromValues(values url.Values) string {
	for _, key := range []string{"PhoneNumber", "phoneNumber"} {
		if value := strings.TrimSpace(values.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func snsNextTokenFromValues(values url.Values) string {
	for _, key := range []string{"NextToken", "nextToken"} {
		if value := strings.TrimSpace(values.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func snsMemberValues(values url.Values, root string) []string {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil
	}

	type indexedValue struct {
		index int
		value string
	}

	direct := make([]string, 0)
	indexed := make([]indexedValue, 0)

	for rawKey, bucket := range values {
		if len(bucket) == 0 {
			continue
		}
		value := strings.TrimSpace(bucket[0])
		if value == "" {
			continue
		}

		if strings.EqualFold(strings.TrimSpace(rawKey), root) {
			direct = append(direct, value)
			continue
		}

		parts := strings.Split(rawKey, ".")
		if len(parts) < 2 || !strings.EqualFold(parts[0], root) {
			continue
		}

		switch {
		case strings.EqualFold(parts[1], "member") && len(parts) >= 3:
			index, err := strconv.Atoi(parts[2])
			if err != nil || index <= 0 {
				continue
			}
			indexed = append(indexed, indexedValue{index: index, value: value})
		default:
			index, err := strconv.Atoi(parts[1])
			if err != nil || index <= 0 {
				continue
			}
			indexed = append(indexed, indexedValue{index: index, value: value})
		}
	}

	sort.Slice(indexed, func(i, j int) bool { return indexed[i].index < indexed[j].index })

	result := make([]string, 0, len(direct)+len(indexed))
	result = append(result, direct...)
	for _, item := range indexed {
		result = append(result, item.value)
	}
	return result
}

func snsUniqueValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
