package application

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/michasdev/mildstack/core/internal/resources/dynamodb/domain"
)

var (
	queryPartitionMatcher      = regexp.MustCompile(`(?i)^\s*(.+?)\s*=\s*(:[A-Za-z0-9_]+)\s*(?:AND\s*(.+))?$`)
	querySortBeginsWithMatcher = regexp.MustCompile(`(?i)^begins_with\(\s*(.+?)\s*,\s*(:[A-Za-z0-9_]+)\s*\)$`)
	querySortBetweenMatcher    = regexp.MustCompile(`(?i)^(.+?)\s+BETWEEN\s+(:[A-Za-z0-9_]+)\s+AND\s+(:[A-Za-z0-9_]+)$`)
	querySortComparisonMatcher = regexp.MustCompile(`(?i)^(.+?)\s*(=|<=|<|>=|>)\s*(:[A-Za-z0-9_]+)$`)
	filterExistsMatcher        = regexp.MustCompile(`(?i)^attribute_exists\(\s*(.+?)\s*\)$`)
	filterNotExistsMatcher     = regexp.MustCompile(`(?i)^attribute_not_exists\(\s*(.+?)\s*\)$`)
	filterBeginsWithMatcher    = regexp.MustCompile(`(?i)^begins_with\(\s*(.+?)\s*,\s*(:[A-Za-z0-9_]+)\s*\)$`)
	filterComparisonMatcher    = regexp.MustCompile(`(?i)^(.+?)\s*(=|<=|<|>=|>|<>)\s*(:[A-Za-z0-9_]+)$`)
)

type queryPlan struct {
	partitionKeyName string
	partitionValue   domain.AttributeValue
	sortKeyName      string
	sortPredicate    sortPredicate
}

type sortPredicate struct {
	kind   string
	values []domain.AttributeValue
}

type filterClause struct {
	kind   string
	path   string
	op     string
	values []domain.AttributeValue
}

func buildQueryPlan(table domain.Table, keyConditionExpression string, expressionAttributeNames map[string]string, expressionAttributeValues map[string]domain.AttributeValue) (queryPlan, error) {
	expression := strings.TrimSpace(keyConditionExpression)
	if expression == "" {
		return queryPlan{}, fmt.Errorf("dynamodb: key condition expression is required")
	}
	if strings.ContainsAny(expression, "[]") {
		return queryPlan{}, fmt.Errorf("dynamodb: unsupported nested key condition expression %q", keyConditionExpression)
	}

	matches := queryPartitionMatcher.FindStringSubmatch(expression)
	if len(matches) != 4 {
		return queryPlan{}, fmt.Errorf("dynamodb: unsupported key condition expression %q", keyConditionExpression)
	}

	partitionKeyName, err := resolveExpressionPath(matches[1], expressionAttributeNames)
	if err != nil {
		return queryPlan{}, err
	}
	if partitionKeyName != table.PartitionKey {
		return queryPlan{}, fmt.Errorf("dynamodb: unsupported key condition partition key %q", partitionKeyName)
	}

	partitionValue, err := resolveExpressionValue(matches[2], expressionAttributeValues)
	if err != nil {
		return queryPlan{}, err
	}

	plan := queryPlan{
		partitionKeyName: partitionKeyName,
		partitionValue:   partitionValue.Clone(),
	}

	sortExpression := strings.TrimSpace(matches[3])
	if sortExpression == "" {
		return plan, nil
	}
	if table.SortKey == "" {
		return queryPlan{}, fmt.Errorf("dynamodb: sort key conditions are not supported for table %q", table.Name)
	}

	sortPath, predicate, err := parseSortPredicate(sortExpression, expressionAttributeNames, expressionAttributeValues)
	if err != nil {
		return queryPlan{}, err
	}
	if sortPath != table.SortKey {
		return queryPlan{}, fmt.Errorf("dynamodb: unsupported key condition sort key %q", sortPath)
	}

	plan.sortKeyName = sortPath
	plan.sortPredicate = predicate
	return plan, nil
}

func parseSortPredicate(expression string, expressionAttributeNames map[string]string, expressionAttributeValues map[string]domain.AttributeValue) (string, sortPredicate, error) {
	if matches := querySortBeginsWithMatcher.FindStringSubmatch(expression); len(matches) == 3 {
		path, err := resolveExpressionPath(matches[1], expressionAttributeNames)
		if err != nil {
			return "", sortPredicate{}, err
		}
		value, err := resolveExpressionValue(matches[2], expressionAttributeValues)
		if err != nil {
			return "", sortPredicate{}, err
		}
		if value.S == nil {
			return "", sortPredicate{}, fmt.Errorf("dynamodb: begins_with requires a string value")
		}
		return path, sortPredicate{kind: "begins_with", values: []domain.AttributeValue{value.Clone()}}, nil
	}

	if matches := querySortBetweenMatcher.FindStringSubmatch(expression); len(matches) == 4 {
		path, err := resolveExpressionPath(matches[1], expressionAttributeNames)
		if err != nil {
			return "", sortPredicate{}, err
		}
		start, err := resolveExpressionValue(matches[2], expressionAttributeValues)
		if err != nil {
			return "", sortPredicate{}, err
		}
		end, err := resolveExpressionValue(matches[3], expressionAttributeValues)
		if err != nil {
			return "", sortPredicate{}, err
		}
		return path, sortPredicate{kind: "between", values: []domain.AttributeValue{start.Clone(), end.Clone()}}, nil
	}

	if matches := querySortComparisonMatcher.FindStringSubmatch(expression); len(matches) == 4 {
		path, err := resolveExpressionPath(matches[1], expressionAttributeNames)
		if err != nil {
			return "", sortPredicate{}, err
		}
		value, err := resolveExpressionValue(matches[3], expressionAttributeValues)
		if err != nil {
			return "", sortPredicate{}, err
		}
		return path, sortPredicate{kind: strings.ToLower(strings.TrimSpace(matches[2])), values: []domain.AttributeValue{value.Clone()}}, nil
	}

	return "", sortPredicate{}, fmt.Errorf("dynamodb: unsupported sort key condition %q", expression)
}

func (p queryPlan) matches(item domain.Item, table domain.Table) (bool, error) {
	partitionValue, ok := item.Attributes[p.partitionKeyName]
	if !ok || !attributeValueEquals(partitionValue, p.partitionValue) {
		return false, nil
	}
	if p.sortKeyName == "" {
		return true, nil
	}

	sortValue, ok := item.Attributes[p.sortKeyName]
	if !ok {
		return false, nil
	}

	switch p.sortPredicate.kind {
	case "":
		return true, nil
	case "begins_with":
		if sortValue.S == nil || p.sortPredicate.values[0].S == nil {
			return false, nil
		}
		return strings.HasPrefix(*sortValue.S, *p.sortPredicate.values[0].S), nil
	case "between":
		return compareAttributeValues(sortValue, p.sortPredicate.values[0]) >= 0 && compareAttributeValues(sortValue, p.sortPredicate.values[1]) <= 0, nil
	case "=":
		return attributeValueEquals(sortValue, p.sortPredicate.values[0]), nil
	case "<":
		return compareAttributeValues(sortValue, p.sortPredicate.values[0]) < 0, nil
	case "<=":
		return compareAttributeValues(sortValue, p.sortPredicate.values[0]) <= 0, nil
	case ">":
		return compareAttributeValues(sortValue, p.sortPredicate.values[0]) > 0, nil
	case ">=":
		return compareAttributeValues(sortValue, p.sortPredicate.values[0]) >= 0, nil
	default:
		return false, fmt.Errorf("dynamodb: unsupported sort key predicate %q", p.sortPredicate.kind)
	}
}

func buildExpressionFilter(filterExpression string, expressionAttributeNames map[string]string, expressionAttributeValues map[string]domain.AttributeValue) (func(domain.Item) (bool, error), error) {
	expression := strings.TrimSpace(filterExpression)
	if expression == "" {
		return nil, nil
	}

	clauses, err := parseFilterClauses(expression, expressionAttributeNames, expressionAttributeValues)
	if err != nil {
		return nil, err
	}

	return func(item domain.Item) (bool, error) {
		for _, clause := range clauses {
			matches, err := clause.matches(item)
			if err != nil {
				return false, err
			}
			if !matches {
				return false, nil
			}
		}
		return true, nil
	}, nil
}

func parseFilterClauses(expression string, expressionAttributeNames map[string]string, expressionAttributeValues map[string]domain.AttributeValue) ([]filterClause, error) {
	parts, err := splitFilterExpression(expression)
	if err != nil {
		return nil, err
	}

	clauses := make([]filterClause, 0, len(parts))
	for _, part := range parts {
		switch {
		case filterExistsMatcher.MatchString(part):
			matches := filterExistsMatcher.FindStringSubmatch(part)
			path, err := resolveExpressionPath(matches[1], expressionAttributeNames)
			if err != nil {
				return nil, err
			}
			clauses = append(clauses, filterClause{kind: "exists", path: path})
		case filterNotExistsMatcher.MatchString(part):
			matches := filterNotExistsMatcher.FindStringSubmatch(part)
			path, err := resolveExpressionPath(matches[1], expressionAttributeNames)
			if err != nil {
				return nil, err
			}
			clauses = append(clauses, filterClause{kind: "not_exists", path: path})
		case filterBeginsWithMatcher.MatchString(part):
			matches := filterBeginsWithMatcher.FindStringSubmatch(part)
			path, err := resolveExpressionPath(matches[1], expressionAttributeNames)
			if err != nil {
				return nil, err
			}
			value, err := resolveExpressionValue(matches[2], expressionAttributeValues)
			if err != nil {
				return nil, err
			}
			if value.S == nil {
				return nil, fmt.Errorf("dynamodb: begins_with requires a string value")
			}
			clauses = append(clauses, filterClause{kind: "begins_with", path: path, values: []domain.AttributeValue{value.Clone()}})
		case filterComparisonMatcher.MatchString(part):
			matches := filterComparisonMatcher.FindStringSubmatch(part)
			path, err := resolveExpressionPath(matches[1], expressionAttributeNames)
			if err != nil {
				return nil, err
			}
			value, err := resolveExpressionValue(matches[3], expressionAttributeValues)
			if err != nil {
				return nil, err
			}
			clauses = append(clauses, filterClause{kind: "comparison", path: path, op: strings.ToLower(strings.TrimSpace(matches[2])), values: []domain.AttributeValue{value.Clone()}})
		default:
			return nil, fmt.Errorf("dynamodb: unsupported filter expression %q", part)
		}
	}

	return clauses, nil
}

func splitFilterExpression(expression string) ([]string, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return nil, nil
	}

	clauses := make([]string, 0, 4)
	start := 0
	depth := 0
	for i := 0; i < len(expression); i++ {
		switch expression[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("dynamodb: invalid filter expression %q", expression)
			}
		default:
			if depth == 0 && hasStandaloneAnd(expression, i) {
				clause := strings.TrimSpace(expression[start:i])
				if clause == "" {
					return nil, fmt.Errorf("dynamodb: invalid filter expression %q", expression)
				}
				clauses = append(clauses, clause)
				i += 2
				start = i + 1
			}
		}
	}

	clause := strings.TrimSpace(expression[start:])
	if clause == "" {
		return nil, fmt.Errorf("dynamodb: invalid filter expression %q", expression)
	}
	clauses = append(clauses, clause)
	return clauses, nil
}

func hasStandaloneAnd(expression string, index int) bool {
	if index+3 > len(expression) {
		return false
	}
	if !strings.EqualFold(expression[index:index+3], "AND") {
		return false
	}

	before := byte(' ')
	if index > 0 {
		before = expression[index-1]
	}
	after := byte(' ')
	if index+3 < len(expression) {
		after = expression[index+3]
	}

	return isSpaceBoundary(before) && isSpaceBoundary(after)
}

func isSpaceBoundary(value byte) bool {
	switch value {
	case ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}

func (c filterClause) matches(item domain.Item) (bool, error) {
	value, ok := item.Attributes[c.path]
	switch c.kind {
	case "exists":
		return ok, nil
	case "not_exists":
		return !ok, nil
	case "begins_with":
		if !ok || value.S == nil || c.values[0].S == nil {
			return false, nil
		}
		return strings.HasPrefix(*value.S, *c.values[0].S), nil
	case "comparison":
		if !ok {
			return false, nil
		}
		cmp := compareAttributeValues(value, c.values[0])
		switch c.op {
		case "=":
			return attributeValueEquals(value, c.values[0]), nil
		case "<>":
			return !attributeValueEquals(value, c.values[0]), nil
		case "<":
			return cmp < 0, nil
		case "<=":
			return cmp <= 0, nil
		case ">":
			return cmp > 0, nil
		case ">=":
			return cmp >= 0, nil
		default:
			return false, fmt.Errorf("dynamodb: unsupported filter operator %q", c.op)
		}
	default:
		return false, fmt.Errorf("dynamodb: unsupported filter clause %q", c.kind)
	}
}

func pageReadItems(items []domain.Item, table domain.Table, startIndex int, limit *int, filter func(domain.Item) (bool, error)) (domain.ReadPage, error) {
	if limit != nil && *limit <= 0 {
		return domain.ReadPage{}, fmt.Errorf("dynamodb: limit must be greater than zero")
	}

	if startIndex < 0 || startIndex > len(items) {
		return domain.ReadPage{}, fmt.Errorf("dynamodb: invalid exclusive start key")
	}

	if startIndex == len(items) {
		return domain.ReadPage{}, nil
	}

	end := len(items)
	if limit != nil && startIndex+*limit < end {
		end = startIndex + *limit
	}

	page := domain.ReadPage{}
	for i := startIndex; i < end; i++ {
		page.ScannedCount++
		matches := true
		var err error
		if filter != nil {
			matches, err = filter(items[i])
			if err != nil {
				return domain.ReadPage{}, err
			}
		}
		if matches {
			page.Items = append(page.Items, items[i])
		}
	}
	page.Count = len(page.Items)

	if end < len(items) {
		cursor, err := keyAttributesForItem(table, items[end-1])
		if err != nil {
			return domain.ReadPage{}, err
		}
		page.LastEvaluatedKey = cursor
	}

	return page, nil
}

func locateExclusiveStartKey(items []domain.Item, table domain.Table, exclusiveStartKey map[string]domain.AttributeValue) (int, error) {
	key, err := normalizeKeyAttributes(table, exclusiveStartKey)
	if err != nil {
		return 0, err
	}
	if len(key) == 0 {
		return 0, nil
	}

	for i, item := range items {
		itemKey, err := keyAttributesForItem(table, item)
		if err != nil {
			return 0, err
		}
		if attributeDocumentsEqual(itemKey, key) {
			return i + 1, nil
		}
	}

	return 0, fmt.Errorf("dynamodb: exclusive start key not found")
}

func orderQueryItems(items []domain.Item, table domain.Table, scanIndexForward *bool) []domain.Item {
	ordered := make([]domain.Item, len(items))
	copy(ordered, items)

	forward := true
	if scanIndexForward != nil {
		forward = *scanIndexForward
	}

	sort.SliceStable(ordered, func(i, j int) bool {
		cmp := compareQueryItems(ordered[i], ordered[j], table)
		if forward {
			return cmp < 0
		}
		return cmp > 0
	})

	return ordered
}

func compareQueryItems(left, right domain.Item, table domain.Table) int {
	if table.SortKey != "" {
		leftSort, leftOK := left.Attributes[table.SortKey]
		rightSort, rightOK := right.Attributes[table.SortKey]
		if leftOK && rightOK {
			if cmp := compareAttributeValues(leftSort, rightSort); cmp != 0 {
				return cmp
			}
		}
		if leftOK != rightOK {
			if leftOK {
				return -1
			}
			return 1
		}
	}

	if cmp := strings.Compare(left.Key, right.Key); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(left.Table, right.Table); cmp != 0 {
		return cmp
	}
	return 0
}

func keyAttributesForItem(table domain.Table, item domain.Item) (map[string]domain.AttributeValue, error) {
	if table.PartitionKey == "" {
		return nil, fmt.Errorf("dynamodb: table %q has no partition key", table.Name)
	}

	partitionValue, ok := item.Attributes[table.PartitionKey]
	if !ok {
		return nil, fmt.Errorf("dynamodb: item %s/%s is missing partition key %q", item.Table, item.Key, table.PartitionKey)
	}

	attributes := map[string]domain.AttributeValue{
		table.PartitionKey: partitionValue.Clone(),
	}
	if table.SortKey != "" {
		sortValue, ok := item.Attributes[table.SortKey]
		if !ok {
			return nil, fmt.Errorf("dynamodb: item %s/%s is missing sort key %q", item.Table, item.Key, table.SortKey)
		}
		attributes[table.SortKey] = sortValue.Clone()
	}
	return attributes, nil
}

func normalizeKeyAttributes(table domain.Table, attributes map[string]domain.AttributeValue) (map[string]domain.AttributeValue, error) {
	if len(attributes) == 0 {
		return nil, nil
	}

	expected := []string{table.PartitionKey}
	if table.SortKey != "" {
		expected = append(expected, table.SortKey)
	}

	if len(attributes) != len(expected) {
		return nil, fmt.Errorf("dynamodb: unsupported key attributes %q", strings.Join(sortedMapKeys(attributes), ", "))
	}

	normalized := make(map[string]domain.AttributeValue, len(expected))
	for _, name := range expected {
		value, ok := attributes[name]
		if !ok {
			return nil, fmt.Errorf("dynamodb: missing key attribute %q", name)
		}
		normalized[name] = value.Clone()
	}

	for name := range attributes {
		if name != table.PartitionKey && (table.SortKey == "" || name != table.SortKey) {
			return nil, fmt.Errorf("dynamodb: unsupported key attribute %q", name)
		}
	}

	return normalized, nil
}

func attributeDocumentsEqual(left, right map[string]domain.AttributeValue) bool {
	if len(left) != len(right) {
		return false
	}
	for name, value := range left {
		other, ok := right[name]
		if !ok || !attributeValueEquals(value, other) {
			return false
		}
	}
	return true
}

func attributeValueEquals(left, right domain.AttributeValue) bool {
	return left.S != nil && right.S != nil && *left.S == *right.S ||
		left.N != nil && right.N != nil && *left.N == *right.N ||
		left.BOOL != nil && right.BOOL != nil && *left.BOOL == *right.BOOL ||
		left.NULL && right.NULL ||
		(left.M != nil && right.M != nil && attributeDocumentsEqual(*left.M, *right.M)) ||
		(left.L != nil && right.L != nil && attributeListsEqual(*left.L, *right.L))
}

func attributeListsEqual(left, right []domain.AttributeValue) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if !attributeValueEquals(left[i], right[i]) {
			return false
		}
	}
	return true
}

func compareAttributeValues(left, right domain.AttributeValue) int {
	if left.S != nil && right.S != nil {
		return strings.Compare(*left.S, *right.S)
	}

	if left.N != nil && right.N != nil {
		leftValue, err := strconv.ParseFloat(strings.TrimSpace(*left.N), 64)
		if err != nil {
			return 0
		}
		rightValue, err := strconv.ParseFloat(strings.TrimSpace(*right.N), 64)
		if err != nil {
			return 0
		}
		switch {
		case leftValue < rightValue:
			return -1
		case leftValue > rightValue:
			return 1
		default:
			return 0
		}
	}

	return 0
}

func resolveExpressionPath(raw string, expressionAttributeNames map[string]string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", fmt.Errorf("dynamodb: expression attribute path is required")
	}
	if strings.ContainsAny(path, ".[]") {
		return "", fmt.Errorf("dynamodb: unsupported nested expression attribute path %q", path)
	}
	if strings.HasPrefix(path, "#") {
		resolved, ok := expressionAttributeNames[path]
		if !ok {
			return "", fmt.Errorf("dynamodb: unresolved expression attribute name %q", path)
		}
		path = strings.TrimSpace(resolved)
	}
	if path == "" {
		return "", fmt.Errorf("dynamodb: expression attribute path is required")
	}
	if strings.ContainsAny(path, ".[]") {
		return "", fmt.Errorf("dynamodb: unsupported nested expression attribute path %q", path)
	}
	return path, nil
}

func resolveExpressionValue(token string, expressionAttributeValues map[string]domain.AttributeValue) (domain.AttributeValue, error) {
	valueToken := strings.TrimSpace(token)
	if valueToken == "" {
		return domain.AttributeValue{}, fmt.Errorf("dynamodb: expression attribute value is required")
	}
	if !strings.HasPrefix(valueToken, ":") {
		return domain.AttributeValue{}, fmt.Errorf("dynamodb: unsupported literal expression value %q", valueToken)
	}
	value, ok := expressionAttributeValues[valueToken]
	if !ok {
		return domain.AttributeValue{}, fmt.Errorf("dynamodb: unresolved expression attribute value %q", valueToken)
	}
	return value.Clone(), nil
}

func sortedMapKeys(attributes map[string]domain.AttributeValue) []string {
	keys := make([]string, 0, len(attributes))
	for name := range attributes {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	return keys
}
