package application

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/domain"
)

var updateClauseMatcher = regexp.MustCompile(`(?i)\b(SET|REMOVE|ADD)\b`)

func (s *Service) UpdateItem(table, key, updateExpression, conditionExpression string, expressionAttributeNames map[string]string, expressionAttributeValues map[string]domain.AttributeValue) (domain.Item, error) {
	table = strings.TrimSpace(table)
	key = strings.TrimSpace(key)
	if table == "" {
		return domain.Item{}, fmt.Errorf("dynamodb: table name is required")
	}
	if key == "" {
		return domain.Item{}, fmt.Errorf("dynamodb: item key is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tableInfo, ok := s.state.Table(table)
	if !ok {
		return domain.Item{}, fmt.Errorf("dynamodb: table %q not found", table)
	}

	current, _ := s.state.Item(table, key)
	attrs := cloneDocument(current.Attributes)
	if attrs == nil {
		attrs = make(map[string]domain.AttributeValue)
	}

	if err := evaluateUpdateCondition(attrs, conditionExpression, expressionAttributeNames, expressionAttributeValues); err != nil {
		return domain.Item{}, err
	}

	operations, err := parseUpdateExpression(updateExpression, expressionAttributeNames, expressionAttributeValues)
	if err != nil {
		return domain.Item{}, err
	}
	if len(operations) == 0 {
		return domain.Item{}, fmt.Errorf("dynamodb: update expression is required")
	}

	for _, operation := range operations {
		path := operation.path
		if path == tableInfo.PartitionKey || (tableInfo.SortKey != "" && path == tableInfo.SortKey) {
			return domain.Item{}, fmt.Errorf("dynamodb: unsupported update to key attribute %q", path)
		}

		switch operation.kind {
		case "SET":
			attrs[path] = operation.value.Clone()
		case "REMOVE":
			delete(attrs, path)
		case "ADD":
			updated, err := addAttribute(attrs[path], operation.value)
			if err != nil {
				return domain.Item{}, err
			}
			attrs[path] = updated
		default:
			return domain.Item{}, fmt.Errorf("dynamodb: unsupported update operation %q", operation.kind)
		}
	}

	next := s.state.Clone()
	next.UpsertItem(domain.Item{
		Table:      table,
		Key:        key,
		Attributes: attrs,
	})
	if err := s.commitStateLocked(next); err != nil {
		return domain.Item{}, err
	}

	updated, ok := s.state.Item(table, key)
	if !ok {
		return domain.Item{}, fmt.Errorf("dynamodb: item %s/%s not found after update", table, key)
	}
	return updated, nil
}

type updateOperation struct {
	kind  string
	path  string
	value domain.AttributeValue
}

func parseUpdateExpression(updateExpression string, expressionAttributeNames map[string]string, expressionAttributeValues map[string]domain.AttributeValue) ([]updateOperation, error) {
	expression := strings.TrimSpace(updateExpression)
	if expression == "" {
		return nil, fmt.Errorf("dynamodb: update expression is required")
	}
	upper := strings.ToUpper(expression)
	if strings.Contains(upper, "DELETE") {
		return nil, fmt.Errorf("dynamodb: unsupported update expression %q", updateExpression)
	}
	if strings.ContainsAny(expression, "()[]") {
		return nil, fmt.Errorf("dynamodb: unsupported nested update expression %q", updateExpression)
	}

	matches := updateClauseMatcher.FindAllStringSubmatchIndex(expression, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("dynamodb: invalid update expression %q", updateExpression)
	}

	operations := make([]updateOperation, 0, len(matches))
	for i, match := range matches {
		kind := strings.ToUpper(expression[match[2]:match[3]])
		start := match[1]
		end := len(expression)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		body := strings.TrimSpace(expression[start:end])
		if body == "" {
			return nil, fmt.Errorf("dynamodb: invalid %s clause in update expression %q", kind, updateExpression)
		}

		parsed, err := parseUpdateClause(kind, body, expressionAttributeNames, expressionAttributeValues)
		if err != nil {
			return nil, err
		}
		operations = append(operations, parsed...)
	}

	return operations, nil
}

func parseUpdateClause(kind, body string, expressionAttributeNames map[string]string, expressionAttributeValues map[string]domain.AttributeValue) ([]updateOperation, error) {
	parts := splitAndTrim(body)
	operations := make([]updateOperation, 0, len(parts))

	switch kind {
	case "SET":
		for _, part := range parts {
			equalIndex := strings.Index(part, "=")
			if equalIndex <= 0 {
				return nil, fmt.Errorf("dynamodb: invalid SET assignment %q", part)
			}
			path, err := resolveUpdatePath(strings.TrimSpace(part[:equalIndex]), expressionAttributeNames)
			if err != nil {
				return nil, err
			}
			rawValue := strings.TrimSpace(part[equalIndex+1:])
			if addValue, ok, err := resolveSelfAddUpdateValue(path, rawValue, expressionAttributeNames, expressionAttributeValues); err != nil {
				return nil, err
			} else if ok {
				operations = append(operations, updateOperation{kind: "ADD", path: path, value: addValue})
				continue
			}

			value, err := resolveUpdateValue(rawValue, expressionAttributeValues)
			if err != nil {
				return nil, err
			}
			operations = append(operations, updateOperation{kind: kind, path: path, value: value})
		}
	case "REMOVE":
		for _, part := range parts {
			path, err := resolveUpdatePath(part, expressionAttributeNames)
			if err != nil {
				return nil, err
			}
			operations = append(operations, updateOperation{kind: kind, path: path})
		}
	case "ADD":
		for _, part := range parts {
			fields := strings.Fields(part)
			if len(fields) != 2 {
				return nil, fmt.Errorf("dynamodb: invalid ADD operation %q", part)
			}
			path, err := resolveUpdatePath(fields[0], expressionAttributeNames)
			if err != nil {
				return nil, err
			}
			value, err := resolveUpdateValue(fields[1], expressionAttributeValues)
			if err != nil {
				return nil, err
			}
			operations = append(operations, updateOperation{kind: kind, path: path, value: value})
		}
	default:
		return nil, fmt.Errorf("dynamodb: unsupported update operation %q", kind)
	}

	return operations, nil
}

func evaluateUpdateCondition(attributes map[string]domain.AttributeValue, conditionExpression string, expressionAttributeNames map[string]string, expressionAttributeValues map[string]domain.AttributeValue) error {
	expression := strings.TrimSpace(conditionExpression)
	if expression == "" {
		return nil
	}
	if strings.ContainsAny(expression, "[]") {
		return fmt.Errorf("dynamodb: unsupported condition expression %q", conditionExpression)
	}

	lower := strings.ToLower(expression)
	switch {
	case strings.HasPrefix(lower, "attribute_exists(") && strings.HasSuffix(expression, ")"):
		path := strings.TrimSpace(expression[len("attribute_exists(") : len(expression)-1])
		resolved, err := resolveUpdatePath(path, expressionAttributeNames)
		if err != nil {
			return err
		}
		if _, ok := attributes[resolved]; !ok {
			return fmt.Errorf("dynamodb: conditional check failed")
		}
		return nil
	case strings.HasPrefix(lower, "attribute_not_exists(") && strings.HasSuffix(expression, ")"):
		path := strings.TrimSpace(expression[len("attribute_not_exists(") : len(expression)-1])
		resolved, err := resolveUpdatePath(path, expressionAttributeNames)
		if err != nil {
			return err
		}
		if _, ok := attributes[resolved]; ok {
			return fmt.Errorf("dynamodb: conditional check failed")
		}
		return nil
	case strings.Contains(expression, "="):
		parts := strings.SplitN(expression, "=", 2)
		path, err := resolveUpdatePath(parts[0], expressionAttributeNames)
		if err != nil {
			return err
		}
		value, err := resolveUpdateValue(parts[1], expressionAttributeValues)
		if err != nil {
			return err
		}
		existing, ok := attributes[path]
		if !ok || !reflect.DeepEqual(existing, value) {
			return fmt.Errorf("dynamodb: conditional check failed")
		}
		return nil
	default:
		return fmt.Errorf("dynamodb: unsupported condition expression %q", conditionExpression)
	}
}

func resolveUpdatePath(raw string, expressionAttributeNames map[string]string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", fmt.Errorf("dynamodb: update path is required")
	}
	if strings.ContainsAny(path, ".[]") {
		return "", fmt.Errorf("dynamodb: unsupported nested update path %q", path)
	}
	if strings.HasPrefix(path, "#") {
		resolved, ok := expressionAttributeNames[path]
		if !ok {
			return "", fmt.Errorf("dynamodb: unresolved expression attribute name %q", path)
		}
		path = strings.TrimSpace(resolved)
	}
	if path == "" {
		return "", fmt.Errorf("dynamodb: update path is required")
	}
	if strings.ContainsAny(path, ".[]") {
		return "", fmt.Errorf("dynamodb: unsupported nested update path %q", path)
	}
	return path, nil
}

func resolveUpdateValue(raw string, expressionAttributeValues map[string]domain.AttributeValue) (domain.AttributeValue, error) {
	token := strings.TrimSpace(raw)
	if token == "" {
		return domain.AttributeValue{}, fmt.Errorf("dynamodb: update value is required")
	}
	if !strings.HasPrefix(token, ":") {
		return domain.AttributeValue{}, fmt.Errorf("dynamodb: unsupported literal update value %q", token)
	}
	value, ok := expressionAttributeValues[token]
	if !ok {
		return domain.AttributeValue{}, fmt.Errorf("dynamodb: unresolved expression attribute value %q", token)
	}
	return value.Clone(), nil
}

func resolveSelfAddUpdateValue(targetPath, raw string, expressionAttributeNames map[string]string, expressionAttributeValues map[string]domain.AttributeValue) (domain.AttributeValue, bool, error) {
	parts := strings.Split(raw, "+")
	if len(parts) != 2 {
		return domain.AttributeValue{}, false, nil
	}

	leftPath, err := resolveUpdatePath(parts[0], expressionAttributeNames)
	if err != nil {
		return domain.AttributeValue{}, false, err
	}
	if leftPath != targetPath {
		return domain.AttributeValue{}, false, fmt.Errorf("dynamodb: unsupported update expression %q", raw)
	}

	value, err := resolveUpdateValue(parts[1], expressionAttributeValues)
	if err != nil {
		return domain.AttributeValue{}, false, err
	}
	if value.N == nil {
		return domain.AttributeValue{}, false, fmt.Errorf("dynamodb: ADD requires a numeric value")
	}

	return value, true, nil
}

func addAttribute(existing domain.AttributeValue, delta domain.AttributeValue) (domain.AttributeValue, error) {
	if delta.N == nil {
		return domain.AttributeValue{}, fmt.Errorf("dynamodb: ADD requires a numeric value")
	}

	base := "0"
	if existing.N != nil {
		base = *existing.N
	} else if existing.S != nil || existing.BOOL != nil || existing.NULL || existing.M != nil || existing.L != nil {
		return domain.AttributeValue{}, fmt.Errorf("dynamodb: ADD requires an existing numeric attribute")
	}

	sum, err := addNumericStrings(base, *delta.N)
	if err != nil {
		return domain.AttributeValue{}, err
	}
	return domain.NumberValue(sum), nil
}

func addNumericStrings(left, right string) (string, error) {
	leftValue, err := strconv.ParseFloat(strings.TrimSpace(left), 64)
	if err != nil {
		return "", fmt.Errorf("dynamodb: invalid numeric value %q", left)
	}
	rightValue, err := strconv.ParseFloat(strings.TrimSpace(right), 64)
	if err != nil {
		return "", fmt.Errorf("dynamodb: invalid numeric value %q", right)
	}
	sum := leftValue + rightValue
	return strconv.FormatFloat(sum, 'f', -1, 64), nil
}

func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	trimmed := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		trimmed = append(trimmed, part)
	}
	return trimmed
}

func cloneDocument(attributes map[string]domain.AttributeValue) map[string]domain.AttributeValue {
	if attributes == nil {
		return nil
	}

	cloned := make(map[string]domain.AttributeValue, len(attributes))
	for key, value := range attributes {
		cloned[key] = value.Clone()
	}
	return cloned
}
