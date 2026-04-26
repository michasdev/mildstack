package application

import (
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

func (s *Service) Subscribe(topicARN, protocol, endpoint string, attributes map[string]string, returnSubscriptionARN bool) (domain.SubscribeOutput, error) {
	if err := s.ensureStore(); err != nil {
		return domain.SubscribeOutput{}, err
	}

	tenant := s.defaultTenant()
	if _, err := s.topicRepository().GetByARN(tenant.Key(), topicARN); err != nil {
		return domain.SubscribeOutput{}, err
	}

	subscription, err := domain.NewSubscription(tenant, topicARN, protocol, endpoint, attributes, time.Now().UTC())
	if err != nil {
		return domain.SubscribeOutput{}, err
	}

	persisted, err := s.subscriptionRepository().Create(subscription)
	if err != nil {
		return domain.SubscribeOutput{}, err
	}

	return domain.SubscribeOutput{
		Subscription:         persisted,
		ResponseSubscription: persisted.SubscribeResponseARN(returnSubscriptionARN),
	}, nil
}

func (s *Service) ConfirmSubscription(topicARN, token string) (domain.Subscription, error) {
	if err := s.ensureStore(); err != nil {
		return domain.Subscription{}, err
	}

	tenant := s.defaultTenant()
	if _, err := s.topicRepository().GetByARN(tenant.Key(), topicARN); err != nil {
		return domain.Subscription{}, err
	}

	current, err := s.subscriptionRepository().GetByToken(tenant.Key(), topicARN, token)
	if err != nil {
		return domain.Subscription{}, err
	}

	confirmed, err := current.Confirm(token, time.Now().UTC())
	if err != nil {
		return domain.Subscription{}, err
	}
	if err := s.subscriptionRepository().Update(confirmed); err != nil {
		return domain.Subscription{}, err
	}
	return confirmed, nil
}

func (s *Service) Unsubscribe(subscriptionARN string) error {
	if err := s.ensureStore(); err != nil {
		return err
	}
	return s.subscriptionRepository().DeleteByARN(s.defaultTenant().Key(), subscriptionARN)
}

func (s *Service) GetSubscriptionAttributes(subscriptionARN string) (map[string]string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, err
	}
	subscription, err := s.subscriptionRepository().GetByARN(s.defaultTenant().Key(), subscriptionARN)
	if err != nil {
		return nil, err
	}
	return subscription.AttributesView(), nil
}

func (s *Service) SetSubscriptionAttributes(subscriptionARN, attributeName, attributeValue string) (map[string]string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, err
	}

	tenantKey := s.defaultTenant().Key()
	current, err := s.subscriptionRepository().GetByARN(tenantKey, subscriptionARN)
	if err != nil {
		return nil, err
	}
	updated, err := current.WithAttribute(attributeName, attributeValue, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if err := s.subscriptionRepository().Update(updated); err != nil {
		return nil, err
	}
	return updated.AttributesView(), nil
}

func (s *Service) ListSubscriptions(nextToken string) ([]domain.Subscription, string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, "", err
	}
	return s.subscriptionRepository().ListByTenant(s.defaultTenant().Key(), nextToken, 100)
}

func (s *Service) ListSubscriptionsByTopic(topicARN, nextToken string) ([]domain.Subscription, string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, "", err
	}

	tenant := s.defaultTenant()
	if _, err := s.topicRepository().GetByARN(tenant.Key(), topicARN); err != nil {
		return nil, "", err
	}
	return s.subscriptionRepository().ListByTopic(tenant.Key(), topicARN, nextToken, 100)
}
