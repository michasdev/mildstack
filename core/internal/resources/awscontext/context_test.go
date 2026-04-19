package awscontext

import "testing"

func TestDefaultReturnsLocalIdentity(t *testing.T) {
	t.Helper()

	ctx := Default()
	if got, want := ctx.AccountID, defaultAccountID; got != want {
		t.Fatalf("unexpected default account id: got %q want %q", got, want)
	}
	if got, want := ctx.Region, defaultRegion; got != want {
		t.Fatalf("unexpected default region: got %q want %q", got, want)
	}
	if got, want := ctx.Partition, defaultPartition; got != want {
		t.Fatalf("unexpected default partition: got %q want %q", got, want)
	}
}

func TestContextCopyHelpersDoNotMutateOriginal(t *testing.T) {
	t.Helper()

	base := Default()
	accountOverride := base.WithAccountID("111122223333")
	regionOverride := base.WithRegion("eu-west-1")
	endpointOverride := base.WithEndpoint("http://localhost:9999")
	partitionOverride := base.WithPartition("aws-cn")

	if got, want := base.AccountID, defaultAccountID; got != want {
		t.Fatalf("base account mutated: got %q want %q", got, want)
	}
	if got, want := base.Region, defaultRegion; got != want {
		t.Fatalf("base region mutated: got %q want %q", got, want)
	}
	if got, want := base.Partition, defaultPartition; got != want {
		t.Fatalf("base partition mutated: got %q want %q", got, want)
	}
	if got, want := accountOverride.AccountID, "111122223333"; got != want {
		t.Fatalf("unexpected overridden account id: got %q want %q", got, want)
	}
	if got, want := regionOverride.Region, "eu-west-1"; got != want {
		t.Fatalf("unexpected overridden region: got %q want %q", got, want)
	}
	if got, want := endpointOverride.Endpoint, "http://localhost:9999"; got != want {
		t.Fatalf("unexpected overridden endpoint: got %q want %q", got, want)
	}
	if got, want := partitionOverride.Partition, "aws-cn"; got != want {
		t.Fatalf("unexpected overridden partition: got %q want %q", got, want)
	}
}

func TestARNHelpersBuildStableAWSShapedStrings(t *testing.T) {
	t.Helper()

	ctx := Default().WithAccountID("111122223333").WithRegion("eu-west-1")

	if got, want := ctx.ARN("lambda", "eu-west-1", "111122223333", "function:demo"), "arn:aws:lambda:eu-west-1:111122223333:function:demo"; got != want {
		t.Fatalf("unexpected generic arn: got %q want %q", got, want)
	}
	if got, want := ctx.IAMRoleARN("replication"), "arn:aws:iam::111122223333:role/replication"; got != want {
		t.Fatalf("unexpected iam role arn: got %q want %q", got, want)
	}
	if got, want := ctx.S3BucketARN("mildstack-assets"), "arn:aws:s3:::mildstack-assets"; got != want {
		t.Fatalf("unexpected s3 bucket arn: got %q want %q", got, want)
	}
	if got, want := ctx.DynamoDBTableARN("mildstack-records"), "arn:aws:dynamodb:eu-west-1:111122223333:table/mildstack-records"; got != want {
		t.Fatalf("unexpected dynamodb table arn: got %q want %q", got, want)
	}
}
