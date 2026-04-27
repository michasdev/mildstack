package domain

import (
	"fmt"
	"strings"

	"github.com/michasdev/mildstack/core/internal/resources/awscontext"
)

const StateKey = "services/sns"

// Tenant represents the account+region isolation boundary for SNS entities.
type Tenant struct {
	AccountID string
	Region    string
	Partition string
}

func NewTenant(accountID, region, partition string) Tenant {
	ctx := awscontext.Default().WithAccountID(accountID).WithRegion(region).WithPartition(partition).Normalize()
	return Tenant{
		AccountID: ctx.AccountID,
		Region:    ctx.Region,
		Partition: ctx.Partition,
	}
}

func (t Tenant) Key() string {
	return strings.TrimSpace(t.AccountID) + ":" + strings.TrimSpace(t.Region)
}

func (t Tenant) TopicARN(topicName string) string {
	topicName = strings.TrimSpace(topicName)
	if topicName == "" {
		return ""
	}
	return fmt.Sprintf("arn:%s:sns:%s:%s:%s", strings.TrimSpace(t.Partition), strings.TrimSpace(t.Region), strings.TrimSpace(t.AccountID), topicName)
}

func (t Tenant) PlatformApplicationARN(platform, applicationName string) string {
	platform = strings.ToUpper(strings.TrimSpace(platform))
	applicationName = strings.TrimSpace(applicationName)
	if platform == "" || applicationName == "" {
		return ""
	}
	return fmt.Sprintf(
		"arn:%s:sns:%s:%s:app/%s/%s",
		strings.TrimSpace(t.Partition),
		strings.TrimSpace(t.Region),
		strings.TrimSpace(t.AccountID),
		platform,
		applicationName,
	)
}

func (t Tenant) PlatformEndpointARN(platform, applicationName, endpointID string) string {
	platform = strings.ToUpper(strings.TrimSpace(platform))
	applicationName = strings.TrimSpace(applicationName)
	endpointID = strings.TrimSpace(endpointID)
	if platform == "" || applicationName == "" || endpointID == "" {
		return ""
	}
	return fmt.Sprintf(
		"arn:%s:sns:%s:%s:endpoint/%s/%s/%s",
		strings.TrimSpace(t.Partition),
		strings.TrimSpace(t.Region),
		strings.TrimSpace(t.AccountID),
		platform,
		applicationName,
		endpointID,
	)
}

type ParsedResourceARN struct {
	Partition  string
	Service    string
	Region     string
	AccountID  string
	Resource   string
	Kind       string
	Platform   string
	EntityName string
}

func ParseResourceARN(arn string) (ParsedResourceARN, error) {
	arn = strings.TrimSpace(arn)
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) != 6 || !strings.EqualFold(parts[0], "arn") {
		return ParsedResourceARN{}, fmt.Errorf("%w: invalid arn format", ErrValidation)
	}

	parsed := ParsedResourceARN{
		Partition: strings.TrimSpace(parts[1]),
		Service:   strings.TrimSpace(parts[2]),
		Region:    strings.TrimSpace(parts[3]),
		AccountID: strings.TrimSpace(parts[4]),
		Resource:  strings.TrimSpace(parts[5]),
	}
	if !strings.EqualFold(parsed.Service, "sns") {
		return ParsedResourceARN{}, fmt.Errorf("%w: arn is not an sns resource", ErrValidation)
	}
	if parsed.Resource == "" {
		return ParsedResourceARN{}, fmt.Errorf("%w: arn resource segment is required", ErrValidation)
	}

	switch {
	case strings.HasPrefix(parsed.Resource, "app/"):
		segments := strings.Split(parsed.Resource, "/")
		if len(segments) != 3 {
			return ParsedResourceARN{}, fmt.Errorf("%w: invalid platform application arn resource", ErrValidation)
		}
		parsed.Kind = "app"
		parsed.Platform = strings.TrimSpace(segments[1])
		parsed.EntityName = strings.TrimSpace(segments[2])
	case strings.HasPrefix(parsed.Resource, "endpoint/"):
		segments := strings.Split(parsed.Resource, "/")
		if len(segments) != 4 {
			return ParsedResourceARN{}, fmt.Errorf("%w: invalid endpoint arn resource", ErrValidation)
		}
		parsed.Kind = "endpoint"
		parsed.Platform = strings.TrimSpace(segments[1])
		parsed.EntityName = strings.TrimSpace(segments[2])
	default:
		parsed.Kind = "topic"
		parsed.EntityName = parsed.Resource
	}

	return parsed, nil
}
