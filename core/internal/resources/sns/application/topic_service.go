package application

import (
	"fmt"
	"strings"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/awscontext"
	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
	"github.com/michasdev/mildstack/core/internal/resources/sns/infrastructure"
)

func (s *Service) CreateTopic(name string, attributes map[string]string) (domain.Topic, error) {
	if err := s.ensureStore(); err != nil {
		return domain.Topic{}, err
	}

	tenant := s.defaultTenant()
	topic, err := domain.NewTopic(tenant, name, attributes, time.Now().UTC())
	if err != nil {
		return domain.Topic{}, err
	}

	persisted, err := s.topicRepository().Create(topic)
	if err != nil {
		return domain.Topic{}, err
	}
	s.syncStateSnapshot(tenant)
	return persisted, nil
}

func (s *Service) DeleteTopic(topicARN string) error {
	if err := s.ensureStore(); err != nil {
		return err
	}

	tenant := s.defaultTenant()
	if err := s.topicRepository().DeleteByARN(tenant.Key(), topicARN); err != nil {
		return err
	}
	s.syncStateSnapshot(tenant)
	return nil
}

func (s *Service) GetTopicAttributes(topicARN string) (map[string]string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, err
	}
	tenant := s.defaultTenant()
	topic, err := s.topicRepository().GetByARN(tenant.Key(), topicARN)
	if err != nil {
		return nil, err
	}
	confirmed, pending, deleted, err := s.subscriptionRepository().CountByTopicAndStatus(tenant.Key(), topicARN)
	if err != nil {
		return nil, err
	}
	return topic.AttributesView(tenant.AccountID, confirmed, pending, deleted), nil
}

func (s *Service) SetTopicAttributes(topicARN, attributeName, attributeValue string) (map[string]string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, err
	}

	tenant := s.defaultTenant()
	topic, err := s.topicRepository().GetByARN(tenant.Key(), topicARN)
	if err != nil {
		return nil, err
	}

	updated, err := topic.WithAttribute(attributeName, attributeValue, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if err := s.topicRepository().Update(updated); err != nil {
		return nil, err
	}
	return s.GetTopicAttributes(topicARN)
}

func (s *Service) ListTopics(nextToken string) ([]domain.Topic, string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, "", err
	}
	tenant := s.defaultTenant()
	return s.topicRepository().ListByTenant(tenant.Key(), nextToken, 100)
}

func (s *Service) ensureStore() error {
	if s == nil || s.store == nil {
		return fmt.Errorf("sns: storage backend is not configured")
	}
	return nil
}

func (s *Service) defaultTenant() domain.Tenant {
	aws := awscontext.Default().Normalize()
	return domain.NewTenant(aws.AccountID, aws.Region, aws.Partition)
}

func (s *Service) topicRepository() infrastructure.TopicRepository {
	return infrastructure.NewTopicRepository(s.store)
}

func (s *Service) subscriptionRepository() infrastructure.SubscriptionRepository {
	return infrastructure.NewSubscriptionRepository(s.store)
}

func (s *Service) syncStateSnapshot(tenant domain.Tenant) {
	if s == nil || s.stateHook == nil {
		return
	}
	topics, _, err := s.topicRepository().ListByTenant(tenant.Key(), "", 100)
	if err != nil {
		return
	}

	topicARNs := make([]string, 0, len(topics))
	for _, topic := range topics {
		topicARNs = append(topicARNs, topic.ARN)
	}

	s.stateHook.Set(domain.StateKey, map[string]any{
		"service":       "sns",
		"tenant":        strings.TrimSpace(tenant.Key()),
		"topics":        topicARNs,
		"observability": s.observability.snapshot(),
	})
}
