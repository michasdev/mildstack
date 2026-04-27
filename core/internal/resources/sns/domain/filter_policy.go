package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	FilterPolicyScopeMessageAttributes = "MessageAttributes"
	FilterPolicyScopeMessageBody       = "MessageBody"
)

// EvaluateSubscriptionFilter evaluates SNS filter policy against publish payload.
// It returns true when no filter policy is configured.
func EvaluateSubscriptionFilter(filterPolicyJSON, scope string, message PublishedMessage) (bool, error) {
	filterPolicyJSON = strings.TrimSpace(filterPolicyJSON)
	if filterPolicyJSON == "" {
		return true, nil
	}

	var policy map[string]any
	decoder := json.NewDecoder(strings.NewReader(filterPolicyJSON))
	decoder.UseNumber()
	if err := decoder.Decode(&policy); err != nil {
		return false, fmt.Errorf("%w: invalid filter policy json", ErrValidation)
	}
	if len(policy) == 0 {
		return true, nil
	}

	candidateValues, err := filterScopeValues(scope, message)
	if err != nil {
		return false, err
	}

	for key, rule := range policy {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		value, exists := candidateValues[key]
		if !evaluateRuleSet(rule, value, exists) {
			return false, nil
		}
	}

	return true, nil
}

func filterScopeValues(scope string, message PublishedMessage) (map[string]any, error) {
	scope = strings.TrimSpace(scope)
	if scope == "" || strings.EqualFold(scope, FilterPolicyScopeMessageAttributes) {
		values := map[string]any{}
		for key, attribute := range message.MessageAttributes {
			if strings.TrimSpace(attribute.StringValue) != "" {
				values[key] = attribute.StringValue
				continue
			}
			values[key] = attribute.BinaryValue
		}
		return values, nil
	}

	if !strings.EqualFold(scope, FilterPolicyScopeMessageBody) {
		return nil, fmt.Errorf("%w: unsupported filter policy scope %q", ErrValidation, scope)
	}

	if strings.TrimSpace(message.Payload) == "" {
		return map[string]any{}, nil
	}

	var body map[string]any
	decoder := json.NewDecoder(strings.NewReader(message.Payload))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		// AWS treats unparsable message body as non-match for MessageBody scope.
		return map[string]any{}, nil
	}
	if body == nil {
		return map[string]any{}, nil
	}
	return body, nil
}

func evaluateRuleSet(rule any, value any, exists bool) bool {
	conditions, ok := rule.([]any)
	if !ok {
		conditions = []any{rule}
	}
	if len(conditions) == 0 {
		return false
	}

	for _, condition := range conditions {
		if evaluateSingleCondition(condition, value, exists) {
			return true
		}
	}
	return false
}

func evaluateSingleCondition(condition any, value any, exists bool) bool {
	switch typed := condition.(type) {
	case map[string]any:
		return evaluateOperatorCondition(typed, value, exists)
	case string:
		return exists && stringValue(value) == typed
	case json.Number:
		actual, ok := floatValue(value)
		if !ok {
			return false
		}
		expected, err := typed.Float64()
		if err != nil {
			return false
		}
		return actual == expected
	case float64:
		actual, ok := floatValue(value)
		return ok && actual == typed
	case bool:
		actual, ok := boolValue(value)
		return ok && actual == typed
	default:
		return false
	}
}

func evaluateOperatorCondition(condition map[string]any, value any, exists bool) bool {
	if rawExists, ok := condition["exists"]; ok {
		expected, ok := rawExists.(bool)
		return ok && exists == expected
	}
	if !exists {
		return false
	}

	if rawPrefix, ok := condition["prefix"]; ok {
		prefix, ok := rawPrefix.(string)
		return ok && strings.HasPrefix(stringValue(value), prefix)
	}

	if rawAnythingBut, ok := condition["anything-but"]; ok {
		return !matchesAnythingBut(rawAnythingBut, value)
	}

	if rawNumeric, ok := condition["numeric"]; ok {
		return evaluateNumericCondition(rawNumeric, value)
	}

	return false
}

func matchesAnythingBut(criterion any, value any) bool {
	switch typed := criterion.(type) {
	case []any:
		for _, item := range typed {
			if evaluateSingleCondition(item, value, true) {
				return true
			}
		}
		return false
	default:
		return evaluateSingleCondition(typed, value, true)
	}
}

func evaluateNumericCondition(raw any, value any) bool {
	tokens, ok := raw.([]any)
	if !ok || len(tokens) < 2 || len(tokens)%2 != 0 {
		return false
	}
	actual, ok := floatValue(value)
	if !ok {
		return false
	}

	for i := 0; i < len(tokens); i += 2 {
		op := stringValue(tokens[i])
		reference, ok := floatValue(tokens[i+1])
		if !ok {
			return false
		}
		if !evaluateNumericOperator(actual, op, reference) {
			return false
		}
	}
	return true
}

func evaluateNumericOperator(actual float64, operator string, reference float64) bool {
	switch strings.TrimSpace(operator) {
	case "=":
		return actual == reference
	case ">":
		return actual > reference
	case ">=":
		return actual >= reference
	case "<":
		return actual < reference
	case "<=":
		return actual <= reference
	default:
		return false
	}
}

func floatValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func boolValue(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		lower := strings.ToLower(strings.TrimSpace(typed))
		switch lower {
		case "1", "true", "yes", "on", "y":
			return true, true
		case "0", "false", "no", "off", "n":
			return false, true
		}
		return false, false
	default:
		return false, false
	}
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", typed)
	}
}
