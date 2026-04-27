package sns_test

import (
	"context"
	"testing"

	snsapplication "github.com/michasdev/mildstack/core/internal/resources/sns/application"
)

func TestSNSAdminControlsLifecycle(t *testing.T) {
	t.Helper()

	service, err := snsapplication.NewWithPersistence(snsapplication.StorageConfig{
		BaseDir:    t.TempDir(),
		InstanceID: "integration-admin-controls",
	})
	if err != nil {
		t.Fatalf("new sns service: %v", err)
	}
	t.Cleanup(func() { _ = service.Stop(context.Background()) })

	topic, err := service.CreateTopic("orders", nil)
	if err != nil {
		t.Fatalf("create topic: %v", err)
	}

	if err := service.TagResource(topic.ARN, map[string]string{"env": "dev", "team": "core"}); err != nil {
		t.Fatalf("tag topic resource: %v", err)
	}
	tags, err := service.ListTagsForResource(topic.ARN)
	if err != nil {
		t.Fatalf("list topic tags: %v", err)
	}
	if got, want := tags["env"], "dev"; got != want {
		t.Fatalf("unexpected topic tag value: got %q want %q", got, want)
	}

	if err := service.AddPermission(topic.ARN, "AllowPublish", []string{"111111111111"}, []string{"Publish"}); err != nil {
		t.Fatalf("add permission: %v", err)
	}
	attributes, err := service.GetTopicAttributes(topic.ARN)
	if err != nil {
		t.Fatalf("get topic attributes after add permission: %v", err)
	}
	if attributes["Policy"] == "" {
		t.Fatal("expected policy attribute to be populated")
	}

	if err := service.RemovePermission(topic.ARN, "AllowPublish"); err != nil {
		t.Fatalf("remove permission: %v", err)
	}
	if err := service.PutDataProtectionPolicy(topic.ARN, `{"Name":"mask"}`); err != nil {
		t.Fatalf("put data protection policy: %v", err)
	}
	policy, err := service.GetDataProtectionPolicy(topic.ARN)
	if err != nil {
		t.Fatalf("get data protection policy: %v", err)
	}
	if policy != `{"Name":"mask"}` {
		t.Fatalf("unexpected data protection policy: got %q", policy)
	}

	app, err := service.CreatePlatformApplication("newsapp", "APNS", map[string]string{"PlatformCredential": "cred"})
	if err != nil {
		t.Fatalf("create platform application: %v", err)
	}
	if _, err := service.SetPlatformApplicationAttributes(app.ARN, map[string]string{"EventEndpointCreated": "arn:aws:sns:us-east-1:00000000000:events"}); err != nil {
		t.Fatalf("set platform application attributes: %v", err)
	}
	apps, _, err := service.ListPlatformApplications("")
	if err != nil {
		t.Fatalf("list platform applications: %v", err)
	}
	if got, want := len(apps), 1; got != want {
		t.Fatalf("unexpected platform application count: got %d want %d", got, want)
	}

	endpoint, err := service.CreatePlatformEndpoint(app.ARN, "device-token-1", "customer-a", nil)
	if err != nil {
		t.Fatalf("create platform endpoint: %v", err)
	}
	if _, err := service.SetEndpointAttributes(endpoint.ARN, map[string]string{"Enabled": "false"}); err != nil {
		t.Fatalf("set endpoint attributes: %v", err)
	}
	endpoints, _, err := service.ListEndpointsByPlatformApplication(app.ARN, "")
	if err != nil {
		t.Fatalf("list endpoints by platform application: %v", err)
	}
	if got, want := len(endpoints), 1; got != want {
		t.Fatalf("unexpected endpoint count: got %d want %d", got, want)
	}
	if err := service.DeleteEndpoint(endpoint.ARN); err != nil {
		t.Fatalf("delete endpoint: %v", err)
	}
	if err := service.DeletePlatformApplication(app.ARN); err != nil {
		t.Fatalf("delete platform application: %v", err)
	}

	if err := service.SetSMSAttributes(map[string]string{"DefaultSenderID": "MILD"}); err != nil {
		t.Fatalf("set sms attributes: %v", err)
	}
	smsAttributes, err := service.GetSMSAttributes([]string{"DefaultSenderID"})
	if err != nil {
		t.Fatalf("get sms attributes: %v", err)
	}
	if got, want := smsAttributes["DefaultSenderID"], "MILD"; got != want {
		t.Fatalf("unexpected sms attribute value: got %q want %q", got, want)
	}

	if err := service.CreateSMSSandboxPhoneNumber("+12065550100", "en-US"); err != nil {
		t.Fatalf("create sms sandbox phone number: %v", err)
	}
	phones, _, err := service.ListSMSSandboxPhoneNumbers("")
	if err != nil {
		t.Fatalf("list sms sandbox phone numbers: %v", err)
	}
	if got, want := len(phones), 1; got != want {
		t.Fatalf("unexpected sms sandbox phone count: got %d want %d", got, want)
	}
	if err := service.VerifySMSSandboxPhoneNumber("+12065550100", "123456"); err != nil {
		t.Fatalf("verify sms sandbox phone number: %v", err)
	}

	isOptedOut, err := service.CheckIfPhoneNumberIsOptedOut("+12065550100")
	if err != nil {
		t.Fatalf("check phone opt-out status: %v", err)
	}
	if isOptedOut {
		t.Fatal("expected phone number to be opted-in by default")
	}
	if err := service.OptInPhoneNumber("+12065550100"); err != nil {
		t.Fatalf("opt-in phone number: %v", err)
	}
	optedOutNumbers, _, err := service.ListPhoneNumbersOptedOut("")
	if err != nil {
		t.Fatalf("list opted-out phone numbers: %v", err)
	}
	if got, want := len(optedOutNumbers), 0; got != want {
		t.Fatalf("unexpected opted-out phone count: got %d want %d", got, want)
	}

	originationNumbers, _, err := service.ListOriginationNumbers("")
	if err != nil {
		t.Fatalf("list origination numbers: %v", err)
	}
	if got, want := len(originationNumbers), 1; got != want {
		t.Fatalf("unexpected origination numbers count: got %d want %d", got, want)
	}

	if err := service.DeleteSMSSandboxPhoneNumber("+12065550100"); err != nil {
		t.Fatalf("delete sms sandbox phone number: %v", err)
	}
}
