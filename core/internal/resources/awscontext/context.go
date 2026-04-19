package awscontext

import (
	"fmt"
	"strings"
)

const (
	defaultAccountID = "123456789012"
	defaultRegion    = "us-east-1"
	defaultPartition = "aws"
)

// Context carries the AWS-facing identity used by local services.
type Context struct {
	AccountID string
	Region    string
	Partition string
	Endpoint  string
}

// Default returns the local AWS identity used by MildStack.
func Default() Context {
	return Context{
		AccountID: defaultAccountID,
		Region:    defaultRegion,
		Partition: defaultPartition,
	}
}

// Normalize fills missing fields with local defaults.
func (c Context) Normalize() Context {
	if strings.TrimSpace(c.AccountID) == "" {
		c.AccountID = defaultAccountID
	}
	if strings.TrimSpace(c.Region) == "" {
		c.Region = defaultRegion
	}
	if strings.TrimSpace(c.Partition) == "" {
		c.Partition = defaultPartition
	}
	c.Endpoint = strings.TrimSpace(c.Endpoint)
	return c
}

// WithAccountID returns a copy with a different account ID.
func (c Context) WithAccountID(accountID string) Context {
	c = c.Normalize()
	c.AccountID = strings.TrimSpace(accountID)
	return c
}

// WithRegion returns a copy with a different region.
func (c Context) WithRegion(region string) Context {
	c = c.Normalize()
	c.Region = strings.TrimSpace(region)
	return c
}

// WithPartition returns a copy with a different partition.
func (c Context) WithPartition(partition string) Context {
	c = c.Normalize()
	c.Partition = strings.TrimSpace(partition)
	return c
}

// WithEndpoint returns a copy with a different endpoint value.
func (c Context) WithEndpoint(endpoint string) Context {
	c = c.Normalize()
	c.Endpoint = strings.TrimSpace(endpoint)
	return c
}

// ARN returns a generic AWS ARN using the configured defaults unless explicit
// overrides are provided.
func (c Context) ARN(service, region, accountID, resource string) string {
	c = c.Normalize()
	service = strings.TrimSpace(service)
	resource = strings.TrimSpace(resource)
	if service == "" || resource == "" {
		return ""
	}
	if strings.TrimSpace(region) == "" {
		region = c.Region
	}
	if strings.TrimSpace(accountID) == "" {
		accountID = c.AccountID
	}
	return fmt.Sprintf("arn:%s:%s:%s:%s:%s", c.Partition, service, region, accountID, resource)
}

// ServiceARN returns an ARN using the configured region and account ID.
func (c Context) ServiceARN(service, resource string) string {
	c = c.Normalize()
	return c.ARN(service, c.Region, c.AccountID, resource)
}

// IAMRoleARN returns an IAM role ARN for the configured identity.
func (c Context) IAMRoleARN(role string) string {
	c = c.Normalize()
	role = strings.TrimSpace(role)
	role = strings.TrimPrefix(role, "role/")
	if role == "" {
		return ""
	}
	return fmt.Sprintf("arn:%s:iam::%s:role/%s", c.Partition, c.AccountID, role)
}

// S3BucketARN returns an S3 bucket ARN for the configured partition.
func (c Context) S3BucketARN(bucket string) string {
	c = c.Normalize()
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return ""
	}
	return fmt.Sprintf("arn:%s:s3:::%s", c.Partition, bucket)
}

// DynamoDBTableARN returns a DynamoDB table ARN for the configured identity.
func (c Context) DynamoDBTableARN(table string) string {
	c = c.Normalize()
	table = strings.TrimSpace(table)
	if table == "" {
		return ""
	}
	return fmt.Sprintf("arn:%s:dynamodb:%s:%s:table/%s", c.Partition, c.Region, c.AccountID, table)
}
