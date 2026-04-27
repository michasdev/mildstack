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

	"github.com/michasdev/mildstack/core/internal/resources/awscontext"
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

var sqsKnownActions = func() map[string]struct{} {
	names := contracts.ActionNames()
	index := make(map[string]struct{}, len(names))
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		index[trimmed] = struct{}{}
	}
	return index
}()

var sqsAmbiguousSNSActions = map[string]struct{}{
	"AddPermission":    {},
	"RemovePermission": {},
}

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
			return SQSRequestContext{}, ErrSQSNotOwned
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
		if errors.Is(err, ErrSQSNotOwned) {
			return SQSRequestContext{}, ErrSQSNotOwned
		}
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
	version := strings.TrimSpace(values.Get("Version"))

	if !shouldOwnSQSRequest(action, version) {
		return SQSRequestContext{}, ErrSQSNotOwned
	}

	if action == "" {
		return SQSRequestContext{}, ErrSQSMissingAction
	}
	if version == "" {
		return SQSRequestContext{}, ErrSQSInvalidVersion
	}
	if version != sqsQueryVersion {
		return SQSRequestContext{}, ErrSQSInvalidVersion
	}

	if pathContext.Kind == SQSRequestKindRoot && isQueueScopedAction(action) {
		queueName, accountID, err := queueContextFromValuesForAction(action, values)
		if err != nil {
			return SQSRequestContext{}, err
		}
		pathContext.Kind = SQSRequestKindQueue
		pathContext.QueueName = queueName
		pathContext.AccountID = accountID
		pathContext.NormalizedPath = "/" + strings.Trim(accountID, "/") + "/" + strings.Trim(queueName, "/")
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

func shouldOwnSQSRequest(action, version string) bool {
	action = strings.TrimSpace(action)
	version = strings.TrimSpace(version)

	if action == "" && version == "" {
		return false
	}
	if version == sqsQueryVersion {
		return true
	}
	if version != "" {
		if action == "" {
			return true
		}
		if _, ambiguous := sqsAmbiguousSNSActions[action]; ambiguous {
			return false
		}
		_, ok := sqsKnownActions[action]
		return ok
	}
	if action != "" {
		_, ok := sqsKnownActions[action]
		return ok
	}
	return false
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
				flattenQueueAttributes(dst, typed)
				continue
			}
			if strings.EqualFold(key, "MessageAttributes") {
				flattenMessageAttributes(dst, "MessageAttribute", typed)
				continue
			}
			if strings.EqualFold(key, "MessageSystemAttributes") {
				flattenMessageAttributes(dst, "MessageSystemAttribute", typed)
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
				continue
			}
			if strings.EqualFold(key, "MessageAttributeNames") {
				for i, item := range typed {
					dst.Set(fmt.Sprintf("MessageAttributeName.%d", i+1), fmt.Sprint(item))
				}
				continue
			}
			if strings.EqualFold(key, "MessageSystemAttributeNames") {
				for i, item := range typed {
					dst.Set(fmt.Sprintf("MessageSystemAttributeName.%d", i+1), fmt.Sprint(item))
				}
				continue
			}
			if strings.EqualFold(key, "Entries") {
				for i, item := range typed {
					if entry, ok := item.(map[string]any); ok {
						flattenJSONEntry(dst, i+1, entry)
					}
				}
				continue
			}
			for i, item := range typed {
				dst.Set(fmt.Sprintf("%s.%d", key, i+1), fmt.Sprint(item))
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
		return "", false, ErrSQSNotOwned
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

func queueNameFromQueueURL(raw string) (string, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", fmt.Errorf("sqs: QueueUrl is required")
	}
	if !strings.Contains(trimmed, "://") {
		return trimmed, awscontext.Default().AccountID, nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", "", fmt.Errorf("sqs: invalid QueueUrl: %w", err)
	}

	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(segments) < 2 {
		return "", "", fmt.Errorf("sqs: invalid QueueUrl: missing account or queue name")
	}
	accountID := segments[len(segments)-2]
	queueName := segments[len(segments)-1]
	if queueName == "" {
		return "", "", fmt.Errorf("sqs: invalid QueueUrl: missing queue name")
	}
	return queueName, accountID, nil
}

func isQueueScopedAction(action string) bool {
	switch action {
	case "AddPermission", "CancelMessageMoveTask", "ChangeMessageVisibility", "ChangeMessageVisibilityBatch", "DeleteMessage", "DeleteMessageBatch", "DeleteQueue", "GetQueueAttributes", "ListDeadLetterSourceQueues", "ListMessageMoveTasks", "ListQueueTags", "PurgeQueue", "ReceiveMessage", "RemovePermission", "SendMessage", "SendMessageBatch", "SetQueueAttributes", "StartMessageMoveTask", "TagQueue", "UntagQueue":
		return true
	default:
		return false
	}
}

func queueContextFromValuesForAction(action string, values url.Values) (string, string, error) {
	if queueName, accountID, err := queueNameFromQueueURL(values.Get("QueueUrl")); err == nil {
		return queueName, accountID, nil
	}

	switch action {
	case "StartMessageMoveTask", "ListMessageMoveTasks", "ListDeadLetterSourceQueues":
		queueName, accountID, err := queueNameFromQueueARN(values.Get("SourceArn"))
		if err != nil {
			return "", "", err
		}
		return queueName, accountID, nil
	case "CancelMessageMoveTask":
		queueName, accountID, err := queueNameFromTaskHandle(values.Get("TaskHandle"))
		if err != nil {
			return "", "", err
		}
		return queueName, accountID, nil
	default:
		queueName, accountID, err := queueNameFromQueueURL(values.Get("QueueUrl"))
		if err != nil {
			return "", "", err
		}
		return queueName, accountID, nil
	}
}

func queueNameFromQueueARN(raw string) (string, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", fmt.Errorf("sqs: SourceArn is required")
	}

	parts := strings.Split(trimmed, ":")
	if len(parts) < 6 || !strings.HasPrefix(trimmed, "arn:") {
		return "", "", fmt.Errorf("sqs: invalid SourceArn: %s", raw)
	}
	queueName := parts[len(parts)-1]
	accountID := parts[len(parts)-2]
	if queueName == "" || accountID == "" {
		return "", "", fmt.Errorf("sqs: invalid SourceArn: %s", raw)
	}
	return queueName, accountID, nil
}

func queueNameFromTaskHandle(raw string) (string, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", fmt.Errorf("sqs: TaskHandle is required")
	}
	parts := strings.SplitN(trimmed, "|", 2)
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return "", "", fmt.Errorf("sqs: invalid TaskHandle: %s", raw)
	}
	return queueNameFromQueueARN(parts[0])
}

func sendMessageRequestFromValues(values url.Values) contracts.SendMessageRequest {
	return contracts.SendMessageRequest{
		DelaySeconds:            parseIntValue(values.Get("DelaySeconds")),
		MessageAttributes:       messageAttributesFromValues(values, "MessageAttribute"),
		MessageBody:             values.Get("MessageBody"),
		MessageDeduplicationId:  values.Get("MessageDeduplicationId"),
		MessageGroupId:          values.Get("MessageGroupId"),
		MessageSystemAttributes: messageAttributesFromValues(values, "MessageSystemAttribute"),
		QueueUrl:                values.Get("QueueUrl"),
	}
}

func sendMessageBatchRequestFromValues(values url.Values) contracts.SendMessageBatchRequest {
	return contracts.SendMessageBatchRequest{
		Entries:  sendMessageBatchEntriesFromValues(values),
		QueueUrl: values.Get("QueueUrl"),
	}
}

func deleteMessageRequestFromValues(values url.Values) contracts.DeleteMessageRequest {
	return contracts.DeleteMessageRequest{
		QueueUrl:      values.Get("QueueUrl"),
		ReceiptHandle: values.Get("ReceiptHandle"),
	}
}

func deleteMessageBatchRequestFromValues(values url.Values) contracts.DeleteMessageBatchRequest {
	return contracts.DeleteMessageBatchRequest{
		Entries:  deleteMessageBatchEntriesFromValues(values),
		QueueUrl: values.Get("QueueUrl"),
	}
}

func changeMessageVisibilityRequestFromValues(values url.Values) contracts.ChangeMessageVisibilityRequest {
	return contracts.ChangeMessageVisibilityRequest{
		QueueUrl:          values.Get("QueueUrl"),
		ReceiptHandle:     values.Get("ReceiptHandle"),
		VisibilityTimeout: parseIntValue(values.Get("VisibilityTimeout")),
	}
}

func changeMessageVisibilityBatchRequestFromValues(values url.Values) contracts.ChangeMessageVisibilityBatchRequest {
	return contracts.ChangeMessageVisibilityBatchRequest{
		Entries:  changeMessageVisibilityBatchEntriesFromValues(values),
		QueueUrl: values.Get("QueueUrl"),
	}
}

func receiveMessageRequestFromValues(values url.Values) contracts.ReceiveMessageRequest {
	return contracts.ReceiveMessageRequest{
		AttributeNames:              queueAttributeNames(values),
		MaxNumberOfMessages:         parseIntValue(values.Get("MaxNumberOfMessages")),
		MessageAttributeNames:       listValuesFromPrefix(values, "MessageAttributeName"),
		MessageSystemAttributeNames: listValuesFromPrefix(values, "MessageSystemAttributeName"),
		QueueUrl:                    values.Get("QueueUrl"),
		VisibilityTimeout:           parseIntValue(values.Get("VisibilityTimeout")),
		WaitTimeSeconds:             parseIntValue(values.Get("WaitTimeSeconds")),
	}
}

func tagQueueTagsFromValues(values url.Values) map[string]string {
	type queueTag struct {
		key   string
		value string
	}

	direct := map[string]string{}
	byIndex := map[int]*queueTag{}
	for key, list := range values {
		if len(list) == 0 {
			continue
		}

		parts := strings.Split(key, ".")
		if len(parts) == 2 && strings.EqualFold(parts[0], "Tags") {
			direct[strings.TrimSpace(parts[1])] = list[0]
			continue
		}
		if len(parts) < 4 || !strings.EqualFold(parts[0], "Tags") || !strings.EqualFold(parts[1], "entry") {
			continue
		}
		index, err := strconv.Atoi(parts[2])
		if err != nil || index <= 0 {
			continue
		}
		entry := byIndex[index]
		if entry == nil {
			entry = &queueTag{}
			byIndex[index] = entry
		}
		switch strings.ToLower(parts[3]) {
		case "key":
			entry.key = strings.TrimSpace(list[0])
		case "value":
			entry.value = list[0]
		}
	}

	if len(byIndex) == 0 {
		if len(direct) == 0 {
			return nil
		}
		return direct
	}

	indices := make([]int, 0, len(byIndex))
	for index := range byIndex {
		indices = append(indices, index)
	}
	sort.Ints(indices)

	tags := make(map[string]string, len(indices))
	for _, index := range indices {
		entry := byIndex[index]
		if entry == nil || entry.key == "" {
			continue
		}
		tags[entry.key] = entry.value
	}
	if len(tags) == 0 {
		return nil
	}
	for key, value := range direct {
		tags[key] = value
	}
	return tags
}

func queueTagKeysFromValues(values url.Values) []string {
	keys := make([]string, 0)
	for _, raw := range append(values["TagKey"], values["TagKeys"]...) {
		if trimmed := strings.TrimSpace(raw); trimmed != "" {
			keys = append(keys, trimmed)
		}
	}
	keys = append(keys, listValuesFromPrefix(values, "TagKey")...)
	keys = append(keys, listValuesFromPrefix(values, "TagKeys")...)
	sort.Strings(keys)
	keys = uniqueStringSlice(keys)
	return keys
}

func permissionAccountsFromValues(values url.Values) []string {
	return uniqueStringSlice(append(listValuesFromPrefix(values, "AWSAccountIds"), values["AWSAccountIds"]...))
}

func permissionActionsFromValues(values url.Values) []string {
	return uniqueStringSlice(append(listValuesFromPrefix(values, "Actions"), values["Actions"]...))
}

func uniqueStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			seen[trimmed] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	ordered := make([]string, 0, len(seen))
	for value := range seen {
		ordered = append(ordered, value)
	}
	sort.Strings(ordered)
	return ordered
}

func startMessageMoveTaskRequestFromValues(values url.Values) (string, string, int) {
	return strings.TrimSpace(values.Get("SourceArn")), strings.TrimSpace(values.Get("DestinationArn")), parseIntValue(values.Get("MaxNumberOfMessagesPerSecond"))
}

func cancelMessageMoveTaskRequestFromValues(values url.Values) string {
	return strings.TrimSpace(values.Get("TaskHandle"))
}

func listMessageMoveTasksRequestFromValues(values url.Values) (string, int) {
	return strings.TrimSpace(values.Get("SourceArn")), parseIntValue(values.Get("MaxResults"))
}

func sendMessageBatchEntriesFromValues(values url.Values) []contracts.SendMessageBatchRequestEntry {
	return batchEntriesFromValues(values, []string{"Entries", "SendMessageBatchRequestEntry"}, func(entry url.Values, item contracts.SendMessageBatchRequestEntry) contracts.SendMessageBatchRequestEntry {
		item.Id = entry.Get("Id")
		item.DelaySeconds = parseIntValue(entry.Get("DelaySeconds"))
		item.MessageAttributes = messageAttributesFromValues(entry, "MessageAttribute")
		item.MessageBody = entry.Get("MessageBody")
		item.MessageDeduplicationId = entry.Get("MessageDeduplicationId")
		item.MessageGroupId = entry.Get("MessageGroupId")
		item.MessageSystemAttributes = messageAttributesFromValues(entry, "MessageSystemAttribute")
		return item
	})
}

func deleteMessageBatchEntriesFromValues(values url.Values) []contracts.DeleteMessageBatchRequestEntry {
	return batchEntriesFromValues(values, []string{"Entries", "DeleteMessageBatchRequestEntry"}, func(entry url.Values, item contracts.DeleteMessageBatchRequestEntry) contracts.DeleteMessageBatchRequestEntry {
		item.Id = entry.Get("Id")
		item.ReceiptHandle = entry.Get("ReceiptHandle")
		return item
	})
}

func changeMessageVisibilityBatchEntriesFromValues(values url.Values) []contracts.ChangeMessageVisibilityBatchRequestEntry {
	return batchEntriesFromValues(values, []string{"Entries", "ChangeMessageVisibilityBatchRequestEntry"}, func(entry url.Values, item contracts.ChangeMessageVisibilityBatchRequestEntry) contracts.ChangeMessageVisibilityBatchRequestEntry {
		item.Id = entry.Get("Id")
		item.ReceiptHandle = entry.Get("ReceiptHandle")
		item.VisibilityTimeout = parseIntValue(entry.Get("VisibilityTimeout"))
		return item
	})
}

func batchEntriesFromValues[T any](values url.Values, prefixes []string, build func(url.Values, T) T) []T {
	byIndex := map[int]url.Values{}
	for key, list := range values {
		if len(list) == 0 {
			continue
		}
		parts := strings.Split(key, ".")
		if len(parts) < 3 {
			continue
		}
		if !hasAnyPrefix(parts[0], prefixes) {
			continue
		}
		index, err := strconv.Atoi(parts[1])
		if err != nil || index <= 0 {
			continue
		}
		entry := byIndex[index]
		if entry == nil {
			entry = url.Values{}
			byIndex[index] = entry
		}
		entry.Set(strings.Join(parts[2:], "."), list[0])
	}

	if len(byIndex) == 0 {
		return nil
	}

	indices := make([]int, 0, len(byIndex))
	for index := range byIndex {
		indices = append(indices, index)
	}
	sort.Ints(indices)

	result := make([]T, 0, len(indices))
	for _, index := range indices {
		var item T
		result = append(result, build(byIndex[index], item))
	}
	return result
}

func messageAttributesFromValues(values url.Values, prefix string) map[string]contracts.MessageAttributeValue {
	type messageAttribute struct {
		name  string
		value contracts.MessageAttributeValue
	}

	byIndex := map[int]*messageAttribute{}
	for key, list := range values {
		if len(list) == 0 {
			continue
		}
		parts := strings.Split(key, ".")
		if len(parts) < 3 || !strings.EqualFold(parts[0], prefix) {
			continue
		}
		index, err := strconv.Atoi(parts[1])
		if err != nil || index <= 0 {
			continue
		}
		entry := byIndex[index]
		if entry == nil {
			entry = &messageAttribute{}
			byIndex[index] = entry
		}
		switch strings.ToLower(parts[2]) {
		case "name":
			entry.name = list[0]
		case "value":
			if len(parts) < 4 {
				continue
			}
			switch strings.ToLower(parts[3]) {
			case "datatype":
				entry.value.DataType = list[0]
			case "stringvalue":
				entry.value.StringValue = list[0]
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

	result := map[string]contracts.MessageAttributeValue{}
	for _, index := range indices {
		entry := byIndex[index]
		if entry == nil || trimSpace(entry.name) == "" {
			continue
		}
		result[entry.name] = entry.value
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func listValuesFromPrefix(values url.Values, prefix string) []string {
	items := make([]string, 0)
	for key, list := range values {
		if len(list) == 0 {
			continue
		}
		if !strings.EqualFold(strings.Split(key, ".")[0], prefix) {
			continue
		}
		items = append(items, list[0])
	}
	sort.Strings(items)
	return items
}

func hasAnyPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.EqualFold(value, prefix) {
			return true
		}
	}
	return false
}

func flattenJSONEntry(dst url.Values, index int, entry map[string]any) {
	if index <= 0 || len(entry) == 0 {
		return
	}
	for key, value := range entry {
		switch typed := value.(type) {
		case map[string]any:
			if strings.EqualFold(key, "MessageAttributes") {
				flattenIndexedMessageAttributes(dst, index, "MessageAttribute", typed)
				continue
			}
			if strings.EqualFold(key, "MessageSystemAttributes") {
				flattenIndexedMessageAttributes(dst, index, "MessageSystemAttribute", typed)
				continue
			}
			for nestedKey, nestedValue := range typed {
				dst.Set(fmt.Sprintf("Entries.%d.%s.%s", index, key, nestedKey), fmt.Sprint(nestedValue))
			}
		default:
			dst.Set(fmt.Sprintf("Entries.%d.%s", index, key), fmt.Sprint(value))
		}
	}
}

func flattenQueueAttributes(dst url.Values, typed map[string]any) {
	keys := make([]string, 0, len(typed))
	for attrName := range typed {
		keys = append(keys, attrName)
	}
	sort.Strings(keys)
	for i, attrName := range keys {
		dst.Set(fmt.Sprintf("Attribute.%d.Name", i+1), attrName)
		dst.Set(fmt.Sprintf("Attribute.%d.Value.StringValue", i+1), fmt.Sprint(typed[attrName]))
	}
}

func flattenMessageAttributes(dst url.Values, prefix string, typed map[string]any) {
	keys := make([]string, 0, len(typed))
	for attrName := range typed {
		keys = append(keys, attrName)
	}
	sort.Strings(keys)
	for i, attrName := range keys {
		dst.Set(fmt.Sprintf("%s.%d.Name", prefix, i+1), attrName)
		if nested, ok := typed[attrName].(map[string]any); ok {
			flattenMessageAttributeValue(dst, prefix, i+1, nested)
			continue
		}
		dst.Set(fmt.Sprintf("%s.%d.Value.StringValue", prefix, i+1), fmt.Sprint(typed[attrName]))
	}
}

func flattenIndexedMessageAttributes(dst url.Values, index int, prefix string, typed map[string]any) {
	keys := make([]string, 0, len(typed))
	for attrName := range typed {
		keys = append(keys, attrName)
	}
	sort.Strings(keys)
	for i, attrName := range keys {
		base := fmt.Sprintf("Entries.%d.%s.%d", index, prefix, i+1)
		dst.Set(base+".Name", attrName)
		if nested, ok := typed[attrName].(map[string]any); ok {
			flattenMessageAttributeValue(dst, base, 0, nested)
			continue
		}
		dst.Set(base+".Value.StringValue", fmt.Sprint(typed[attrName]))
	}
}

func flattenMessageAttributeValue(dst url.Values, base string, index int, typed map[string]any) {
	for key, value := range typed {
		switch {
		case strings.EqualFold(key, "DataType"):
			if index > 0 {
				dst.Set(fmt.Sprintf("%s.%d.Value.DataType", base, index), fmt.Sprint(value))
			} else {
				dst.Set(base+".Value.DataType", fmt.Sprint(value))
			}
		case strings.EqualFold(key, "StringValue"):
			if index > 0 {
				dst.Set(fmt.Sprintf("%s.%d.Value.StringValue", base, index), fmt.Sprint(value))
			} else {
				dst.Set(base+".Value.StringValue", fmt.Sprint(value))
			}
		case strings.EqualFold(key, "BinaryValue"):
			if index > 0 {
				dst.Set(fmt.Sprintf("%s.%d.Value.BinaryValue", base, index), fmt.Sprint(value))
			} else {
				dst.Set(base+".Value.BinaryValue", fmt.Sprint(value))
			}
		}
	}
}

func parseIntValue(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return value
}

func trimSpace(value string) string {
	return strings.TrimSpace(value)
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
