package domain

import (
	"testing"
	"time"
)

func TestStateObjectLockAccessorsReturnCopiesAndBucketDeleteCleansProtection(t *testing.T) {
	t.Helper()

	state := NewState()
	state.UpsertBucket(Bucket{Name: "governed-bucket", Region: "us-west-2"})
	state.BucketObjectLock = map[string]ObjectLockConfiguration{
		"governed-bucket": {
			Enabled: true,
			DefaultRetention: &ObjectLockRetention{
				Mode:  "GOVERNANCE",
				Days:  30,
				Years: 0,
			},
		},
	}
	state.ObjectRetention = map[string]map[string]ObjectRetention{
		"governed-bucket": {
			"archive.txt": {
				Mode:            "GOVERNANCE",
				RetainUntilDate: time.Date(2026, time.April, 18, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	state.ObjectLegalHold = map[string]map[string]ObjectLegalHold{
		"governed-bucket": {
			"archive.txt": {
				Status: "ON",
			},
		},
	}

	lock, ok := state.BucketObjectLockConfig("governed-bucket")
	if !ok {
		t.Fatal("expected object lock configuration to be present")
	}
	lock.DefaultRetention.Mode = "COMPLIANCE"

	retention, ok := state.ObjectRetentionConfig("governed-bucket", "archive.txt")
	if !ok {
		t.Fatal("expected object retention to be present")
	}
	retention.Mode = "COMPLIANCE"

	hold, ok := state.ObjectLegalHoldConfig("governed-bucket", "archive.txt")
	if !ok {
		t.Fatal("expected legal hold to be present")
	}
	hold.Status = "OFF"

	if got, want := state.BucketObjectLock["governed-bucket"].DefaultRetention.Mode, "GOVERNANCE"; got != want {
		t.Fatalf("object lock config was aliased: got %q want %q", got, want)
	}
	if got, want := state.ObjectRetention["governed-bucket"]["archive.txt"].Mode, "GOVERNANCE"; got != want {
		t.Fatalf("object retention was aliased: got %q want %q", got, want)
	}
	if got, want := state.ObjectLegalHold["governed-bucket"]["archive.txt"].Status, "ON"; got != want {
		t.Fatalf("legal hold was aliased: got %q want %q", got, want)
	}

	if deleted := state.DeleteBucket("governed-bucket"); !deleted {
		t.Fatal("expected governed bucket delete to succeed")
	}
	if _, ok := state.BucketObjectLockConfig("governed-bucket"); ok {
		t.Fatal("expected bucket object lock config to be removed with the bucket")
	}
	if _, ok := state.ObjectRetentionConfig("governed-bucket", "archive.txt"); ok {
		t.Fatal("expected object retention to be removed with the bucket")
	}
	if _, ok := state.ObjectLegalHoldConfig("governed-bucket", "archive.txt"); ok {
		t.Fatal("expected legal hold to be removed with the bucket")
	}
}
