package application

import (
	"time"
)

func (s *Service) AddPermission(topicARN, label string, awsAccountIDs, actionNames []string) error {
	if err := s.ensureStore(); err != nil {
		return err
	}
	_, err := s.adminRepository().AddPermission(s.defaultTenant().Key(), topicARN, label, awsAccountIDs, actionNames, time.Now().UTC())
	return err
}

func (s *Service) RemovePermission(topicARN, label string) error {
	if err := s.ensureStore(); err != nil {
		return err
	}
	_, err := s.adminRepository().RemovePermission(s.defaultTenant().Key(), topicARN, label, time.Now().UTC())
	return err
}

func (s *Service) TagResource(resourceARN string, tags map[string]string) error {
	if err := s.ensureStore(); err != nil {
		return err
	}
	return s.adminRepository().TagResource(s.defaultTenant().Key(), resourceARN, tags, time.Now().UTC())
}

func (s *Service) UntagResource(resourceARN string, tagKeys []string) error {
	if err := s.ensureStore(); err != nil {
		return err
	}
	return s.adminRepository().UntagResource(s.defaultTenant().Key(), resourceARN, tagKeys, time.Now().UTC())
}

func (s *Service) ListTagsForResource(resourceARN string) (map[string]string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, err
	}
	return s.adminRepository().ListTagsForResource(s.defaultTenant().Key(), resourceARN)
}

func (s *Service) PutDataProtectionPolicy(resourceARN, policyDocument string) error {
	if err := s.ensureStore(); err != nil {
		return err
	}
	return s.adminRepository().PutDataProtectionPolicy(s.defaultTenant().Key(), resourceARN, policyDocument, time.Now().UTC())
}

func (s *Service) GetDataProtectionPolicy(resourceARN string) (string, error) {
	if err := s.ensureStore(); err != nil {
		return "", err
	}
	return s.adminRepository().GetDataProtectionPolicy(s.defaultTenant().Key(), resourceARN)
}
