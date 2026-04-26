package infrastructure

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

const snsTagLimitPerResource = 50

type AdminRepository struct {
	store *SQLiteStore
}

func NewAdminRepository(store *SQLiteStore) AdminRepository {
	return AdminRepository{store: store}
}

func (r AdminRepository) AddPermission(tenantKey, topicARN, label string, awsAccountIDs, actions []string, now time.Time) (string, error) {
	topic, err := r.topicRepository().GetByARN(strings.TrimSpace(tenantKey), strings.TrimSpace(topicARN))
	if err != nil {
		return "", err
	}

	label = strings.TrimSpace(label)
	if label == "" {
		return "", fmt.Errorf("%w: permission label is required", domain.ErrValidation)
	}
	awsAccountIDs = sortedUniqueNonEmpty(awsAccountIDs)
	actions = sortedUniqueNonEmpty(actions)
	if len(awsAccountIDs) == 0 {
		return "", fmt.Errorf("%w: at least one AWSAccountId is required", domain.ErrValidation)
	}
	if len(actions) == 0 {
		return "", fmt.Errorf("%w: at least one ActionName is required", domain.ErrValidation)
	}

	policy, err := decodePermissionPolicy(topic.PolicyJSON)
	if err != nil {
		return "", err
	}
	statement := permissionStatement{
		Sid:      label,
		Effect:   "Allow",
		Resource: topic.ARN,
		Principal: permissionPrincipal{
			AWS: awsAccountIDs,
		},
		Action: actions,
	}

	filtered := make([]permissionStatement, 0, len(policy.Statement)+1)
	for _, item := range policy.Statement {
		if strings.TrimSpace(item.Sid) == label {
			continue
		}
		filtered = append(filtered, item)
	}
	filtered = append(filtered, statement)
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Sid < filtered[j].Sid })
	policy.Statement = filtered

	encoded, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("sns: marshal policy: %w", err)
	}

	topic.PolicyJSON = string(encoded)
	if topic.Attributes == nil {
		topic.Attributes = map[string]string{}
	}
	topic.Attributes["Policy"] = topic.PolicyJSON
	topic.UpdatedAt = now.UTC()
	if err := r.topicRepository().Update(topic); err != nil {
		return "", err
	}
	return topic.PolicyJSON, nil
}

func (r AdminRepository) RemovePermission(tenantKey, topicARN, label string, now time.Time) (string, error) {
	topic, err := r.topicRepository().GetByARN(strings.TrimSpace(tenantKey), strings.TrimSpace(topicARN))
	if err != nil {
		return "", err
	}

	label = strings.TrimSpace(label)
	if label == "" {
		return "", fmt.Errorf("%w: permission label is required", domain.ErrValidation)
	}

	policy, err := decodePermissionPolicy(topic.PolicyJSON)
	if err != nil {
		return "", err
	}

	filtered := make([]permissionStatement, 0, len(policy.Statement))
	removed := false
	for _, item := range policy.Statement {
		if strings.TrimSpace(item.Sid) == label {
			removed = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !removed {
		return "", domain.ErrNotFound
	}
	policy.Statement = filtered

	encoded, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("sns: marshal policy: %w", err)
	}

	topic.PolicyJSON = string(encoded)
	if topic.Attributes == nil {
		topic.Attributes = map[string]string{}
	}
	topic.Attributes["Policy"] = topic.PolicyJSON
	topic.UpdatedAt = now.UTC()
	if err := r.topicRepository().Update(topic); err != nil {
		return "", err
	}
	return topic.PolicyJSON, nil
}

func (r AdminRepository) PutDataProtectionPolicy(tenantKey, resourceARN, policyDocument string, now time.Time) error {
	topic, err := r.topicRepository().GetByARN(strings.TrimSpace(tenantKey), strings.TrimSpace(resourceARN))
	if err != nil {
		return err
	}
	policyDocument = strings.TrimSpace(policyDocument)
	if policyDocument == "" {
		return fmt.Errorf("%w: DataProtectionPolicy is required", domain.ErrValidation)
	}

	if topic.Attributes == nil {
		topic.Attributes = map[string]string{}
	}
	topic.Attributes["DataProtectionPolicy"] = policyDocument
	topic.UpdatedAt = now.UTC()
	return r.topicRepository().Update(topic)
}

func (r AdminRepository) GetDataProtectionPolicy(tenantKey, resourceARN string) (string, error) {
	topic, err := r.topicRepository().GetByARN(strings.TrimSpace(tenantKey), strings.TrimSpace(resourceARN))
	if err != nil {
		return "", err
	}
	policyDocument := strings.TrimSpace(topic.Attributes["DataProtectionPolicy"])
	if policyDocument == "" {
		return "", domain.ErrNotFound
	}
	return policyDocument, nil
}

func (r AdminRepository) TagResource(tenantKey, resourceARN string, tags map[string]string, now time.Time) error {
	tags = normalizeTags(tags)
	if len(tags) == 0 {
		return fmt.Errorf("%w: at least one tag is required", domain.ErrValidation)
	}

	current, err := r.ListTagsForResource(tenantKey, resourceARN)
	if err != nil {
		return err
	}
	if current == nil {
		current = map[string]string{}
	}
	for key, value := range tags {
		current[key] = value
	}
	if len(current) > snsTagLimitPerResource {
		return fmt.Errorf("%w: tag limit exceeded for resource", domain.ErrValidation)
	}
	return r.replaceTags(tenantKey, resourceARN, current, now)
}

func (r AdminRepository) UntagResource(tenantKey, resourceARN string, tagKeys []string, now time.Time) error {
	tagKeys = sortedUniqueNonEmpty(tagKeys)
	if len(tagKeys) == 0 {
		return fmt.Errorf("%w: at least one tag key is required", domain.ErrValidation)
	}

	current, err := r.ListTagsForResource(tenantKey, resourceARN)
	if err != nil {
		return err
	}
	if current == nil {
		current = map[string]string{}
	}
	for _, tagKey := range tagKeys {
		delete(current, tagKey)
	}
	return r.replaceTags(tenantKey, resourceARN, current, now)
}

func (r AdminRepository) ListTagsForResource(tenantKey, resourceARN string) (map[string]string, error) {
	parsed, err := domain.ParseResourceARN(resourceARN)
	if err != nil {
		return nil, err
	}
	tenantKey = strings.TrimSpace(tenantKey)
	resourceARN = strings.TrimSpace(resourceARN)

	switch parsed.Kind {
	case "topic":
		topic, err := r.topicRepository().GetByARN(tenantKey, resourceARN)
		if err != nil {
			return nil, err
		}
		return cloneTags(topic.Tags), nil
	case "app":
		raw, err := r.getJSONColumnByARN("platform_applications", "platform_application_arn", tenantKey, resourceARN, "tags_json")
		if err != nil {
			return nil, err
		}
		return unmarshalStringMap(raw)
	case "endpoint":
		raw, err := r.getJSONColumnByARN("platform_endpoints", "endpoint_arn", tenantKey, resourceARN, "tags_json")
		if err != nil {
			return nil, err
		}
		return unmarshalStringMap(raw)
	default:
		return nil, fmt.Errorf("%w: unsupported taggable resource", domain.ErrValidation)
	}
}

func (r AdminRepository) replaceTags(tenantKey, resourceARN string, tags map[string]string, now time.Time) error {
	parsed, err := domain.ParseResourceARN(resourceARN)
	if err != nil {
		return err
	}
	tenantKey = strings.TrimSpace(tenantKey)
	resourceARN = strings.TrimSpace(resourceARN)
	tagsJSON, err := marshalStringMap(tags)
	if err != nil {
		return err
	}
	updatedAt := now.UTC().Format(time.RFC3339Nano)

	switch parsed.Kind {
	case "topic":
		topic, err := r.topicRepository().GetByARN(tenantKey, resourceARN)
		if err != nil {
			return err
		}
		topic.Tags = cloneTags(tags)
		topic.UpdatedAt = now.UTC()
		return r.topicRepository().Update(topic)
	case "app":
		return r.updateJSONColumnByARN("platform_applications", "platform_application_arn", tenantKey, resourceARN, "tags_json", tagsJSON, updatedAt)
	case "endpoint":
		return r.updateJSONColumnByARN("platform_endpoints", "endpoint_arn", tenantKey, resourceARN, "tags_json", tagsJSON, updatedAt)
	default:
		return fmt.Errorf("%w: unsupported taggable resource", domain.ErrValidation)
	}
}

func (r AdminRepository) getJSONColumnByARN(tableName, arnColumn, tenantKey, arn, jsonColumn string) (string, error) {
	db, err := r.ensureDB()
	if err != nil {
		return "", err
	}

	query := fmt.Sprintf(`SELECT %s FROM %s WHERE tenant_key = ? AND %s = ?`, jsonColumn, tableName, arnColumn)
	row := db.QueryRow(query, tenantKey, arn)
	var raw string
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", domain.ErrNotFound
		}
		return "", fmt.Errorf("sns: query %s tags: %w", tableName, err)
	}
	return raw, nil
}

func (r AdminRepository) updateJSONColumnByARN(tableName, arnColumn, tenantKey, arn, jsonColumn, jsonValue, updatedAt string) error {
	db, err := r.ensureDB()
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`UPDATE %s SET %s = ?, updated_at = ? WHERE tenant_key = ? AND %s = ?`, tableName, jsonColumn, arnColumn)
	result, err := db.Exec(query, jsonValue, updatedAt, tenantKey, arn)
	if err != nil {
		return fmt.Errorf("sns: update %s tags: %w", tableName, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("sns: update %s tags affected rows: %w", tableName, err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r AdminRepository) ensureDB() (*sql.DB, error) {
	if r.store == nil || r.store.db == nil {
		return nil, fmt.Errorf("sns: admin repository not initialized")
	}
	return r.store.db, nil
}

func (r AdminRepository) topicRepository() TopicRepository {
	return NewTopicRepository(r.store)
}

type permissionPolicyDocument struct {
	Version   string                `json:"Version"`
	Statement []permissionStatement `json:"Statement"`
}

type permissionStatement struct {
	Sid       string              `json:"Sid"`
	Effect    string              `json:"Effect"`
	Principal permissionPrincipal `json:"Principal"`
	Action    []string            `json:"Action"`
	Resource  string              `json:"Resource"`
}

type permissionPrincipal struct {
	AWS []string `json:"AWS"`
}

func decodePermissionPolicy(raw string) (permissionPolicyDocument, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return permissionPolicyDocument{Version: "2012-10-17", Statement: []permissionStatement{}}, nil
	}
	var policy permissionPolicyDocument
	if err := json.Unmarshal([]byte(raw), &policy); err != nil {
		return permissionPolicyDocument{}, fmt.Errorf("%w: invalid policy document", domain.ErrValidation)
	}
	if strings.TrimSpace(policy.Version) == "" {
		policy.Version = "2012-10-17"
	}
	if policy.Statement == nil {
		policy.Statement = []permissionStatement{}
	}
	return policy, nil
}

func sortedUniqueNonEmpty(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		seen[trimmed] = struct{}{}
	}
	if len(seen) == 0 {
		return nil
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func normalizeTags(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	result := map[string]string{}
	for key, value := range tags {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		result[trimmedKey] = strings.TrimSpace(value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func cloneTags(tags map[string]string) map[string]string {
	if tags == nil {
		return map[string]string{}
	}
	copied := make(map[string]string, len(tags))
	for key, value := range tags {
		copied[key] = value
	}
	return copied
}
