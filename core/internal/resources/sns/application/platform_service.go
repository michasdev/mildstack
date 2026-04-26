package application

import (
	"strings"
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

func (s *Service) CreatePlatformApplication(name, platform string, attributes map[string]string) (domain.PlatformApplication, error) {
	if err := s.ensureStore(); err != nil {
		return domain.PlatformApplication{}, err
	}

	tenant := s.defaultTenant()
	application, err := domain.NewPlatformApplication(tenant, name, platform, attributes, nil, time.Now().UTC())
	if err != nil {
		return domain.PlatformApplication{}, err
	}
	return s.platformRepository().CreateApplication(application)
}

func (s *Service) DeletePlatformApplication(platformApplicationARN string) error {
	if err := s.ensureStore(); err != nil {
		return err
	}
	return s.platformRepository().DeleteApplicationByARN(s.defaultTenant().Key(), platformApplicationARN)
}

func (s *Service) GetPlatformApplicationAttributes(platformApplicationARN string) (map[string]string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, err
	}
	application, err := s.platformRepository().GetApplicationByARN(s.defaultTenant().Key(), platformApplicationARN)
	if err != nil {
		return nil, err
	}
	return application.AttributesView(), nil
}

func (s *Service) SetPlatformApplicationAttributes(platformApplicationARN string, attributes map[string]string) (map[string]string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, err
	}

	tenantKey := s.defaultTenant().Key()
	current, err := s.platformRepository().GetApplicationByARN(tenantKey, platformApplicationARN)
	if err != nil {
		return nil, err
	}
	updated := current.WithAttributes(attributes, time.Now().UTC())
	if err := s.platformRepository().UpdateApplication(updated); err != nil {
		return nil, err
	}
	return updated.AttributesView(), nil
}

func (s *Service) ListPlatformApplications(nextToken string) ([]domain.PlatformApplication, string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, "", err
	}
	return s.platformRepository().ListApplicationsByTenant(s.defaultTenant().Key(), nextToken, 100)
}

func (s *Service) CreatePlatformEndpoint(platformApplicationARN, token, customUserData string, attributes map[string]string) (domain.PlatformEndpoint, error) {
	if err := s.ensureStore(); err != nil {
		return domain.PlatformEndpoint{}, err
	}

	tenant := s.defaultTenant()
	application, err := s.platformRepository().GetApplicationByARN(tenant.Key(), platformApplicationARN)
	if err != nil {
		return domain.PlatformEndpoint{}, err
	}

	endpoint, err := domain.NewPlatformEndpoint(
		tenant,
		application.ARN,
		application.Platform,
		application.Name,
		token,
		customUserData,
		attributes,
		nil,
		time.Now().UTC(),
	)
	if err != nil {
		return domain.PlatformEndpoint{}, err
	}
	return s.platformRepository().CreateEndpoint(endpoint)
}

func (s *Service) DeleteEndpoint(endpointARN string) error {
	if err := s.ensureStore(); err != nil {
		return err
	}
	return s.platformRepository().DeleteEndpointByARN(s.defaultTenant().Key(), endpointARN)
}

func (s *Service) GetEndpointAttributes(endpointARN string) (map[string]string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, err
	}
	endpoint, err := s.platformRepository().GetEndpointByARN(s.defaultTenant().Key(), endpointARN)
	if err != nil {
		return nil, err
	}
	return endpoint.AttributesView(), nil
}

func (s *Service) SetEndpointAttributes(endpointARN string, attributes map[string]string) (map[string]string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, err
	}

	tenantKey := s.defaultTenant().Key()
	current, err := s.platformRepository().GetEndpointByARN(tenantKey, endpointARN)
	if err != nil {
		return nil, err
	}
	updated := current.WithAttributes(attributes, time.Now().UTC())
	if strings.TrimSpace(updated.Token) == "" {
		updated.Token = current.Token
	}
	if err := s.platformRepository().UpdateEndpoint(updated); err != nil {
		return nil, err
	}
	return updated.AttributesView(), nil
}

func (s *Service) ListEndpointsByPlatformApplication(platformApplicationARN, nextToken string) ([]domain.PlatformEndpoint, string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, "", err
	}
	if _, err := s.platformRepository().GetApplicationByARN(s.defaultTenant().Key(), platformApplicationARN); err != nil {
		return nil, "", err
	}
	return s.platformRepository().ListEndpointsByApplication(s.defaultTenant().Key(), platformApplicationARN, nextToken, 100)
}
