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
