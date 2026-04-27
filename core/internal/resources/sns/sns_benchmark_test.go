package sns_test

import (
	"context"
	"fmt"
	"testing"

	snsapplication "github.com/michasdev/mildstack/core/internal/resources/sns/application"
	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

var (
	benchmarkSNSPublishSink            domain.PublishResult
	benchmarkSNSListTopicsSink         []domain.Topic
	benchmarkSNSListSubscriptionsSink  []domain.Subscription
	benchmarkSNSListSubscriptionsToken string
)

func BenchmarkSNSServicePublishAndList(b *testing.B) {
	b.Run("Publish", benchmarkSNSPublish)
	b.Run("ListTopics", benchmarkSNSListTopics)
	b.Run("ListSubscriptionsByTopic", benchmarkSNSListSubscriptionsByTopic)
}

func benchmarkSNSPublish(b *testing.B) {
	service := newSNSBenchmarkService(b)

	topic, err := service.CreateTopic("bench-publish", nil)
	if err != nil {
		b.Fatalf("create benchmark topic: %v", err)
	}

	subscription, err := service.Subscribe(topic.ARN, "https", "https://example.invalid/sns", nil, true)
	if err != nil {
		b.Fatalf("create benchmark subscription: %v", err)
	}
	if _, err := service.ConfirmSubscription(topic.ARN, subscription.Subscription.Token); err != nil {
		b.Fatalf("confirm benchmark subscription: %v", err)
	}

	request := domain.PublishRequest{
		TopicARN: topic.ARN,
		Message:  "benchmark-message",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkSNSPublishSink, err = service.Publish(request)
		if err != nil {
			b.Fatalf("publish benchmark call failed: %v", err)
		}
	}
}

func benchmarkSNSListTopics(b *testing.B) {
	service := newSNSBenchmarkService(b)

	for i := 0; i < 300; i++ {
		if _, err := service.CreateTopic(fmt.Sprintf("bench-list-topic-%03d", i), nil); err != nil {
			b.Fatalf("seed topic %d: %v", i, err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var err error
		benchmarkSNSListTopicsSink, _, err = service.ListTopics("")
		if err != nil {
			b.Fatalf("list topics benchmark call failed: %v", err)
		}
	}
}

func benchmarkSNSListSubscriptionsByTopic(b *testing.B) {
	service := newSNSBenchmarkService(b)

	topic, err := service.CreateTopic("bench-list-subscriptions", nil)
	if err != nil {
		b.Fatalf("create benchmark topic: %v", err)
	}

	for i := 0; i < 300; i++ {
		endpoint := fmt.Sprintf("http://127.0.0.1:9/bench-%03d", i)
		if _, err := service.Subscribe(topic.ARN, "http", endpoint, nil, true); err != nil {
			b.Fatalf("seed subscription %d: %v", i, err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkSNSListSubscriptionsSink, benchmarkSNSListSubscriptionsToken, err = service.ListSubscriptionsByTopic(topic.ARN, "")
		if err != nil {
			b.Fatalf("list subscriptions by topic benchmark call failed: %v", err)
		}
	}
}

func newSNSBenchmarkService(tb testing.TB) *snsapplication.Service {
	tb.Helper()

	service, err := snsapplication.NewWithPersistence(snsapplication.StorageConfig{
		BaseDir:    tb.TempDir(),
		InstanceID: "sns-benchmark",
	})
	if err != nil {
		tb.Fatalf("new sns benchmark service: %v", err)
	}

	tb.Cleanup(func() {
		_ = service.Stop(context.Background())
	})

	return service
}
