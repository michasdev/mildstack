package application

import (
	"time"

	"github.com/michasdev/mildstack/core/internal/resources/sns/domain"
)

func (s *Service) SetSMSAttributes(attributes map[string]string) error {
	if err := s.ensureStore(); err != nil {
		return err
	}
	return s.smsRepository().SetSMSAttributes(s.defaultTenant().Key(), attributes, time.Now().UTC())
}

func (s *Service) GetSMSAttributes(attributeNames []string) (map[string]string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, err
	}

	all, err := s.smsRepository().GetSMSAttributes(s.defaultTenant().Key())
	if err != nil {
		return nil, err
	}
	if len(attributeNames) == 0 {
		return all, nil
	}

	selected := map[string]string{}
	for _, name := range attributeNames {
		if value, ok := all[name]; ok {
			selected[name] = value
		}
	}
	return selected, nil
}

func (s *Service) CheckIfPhoneNumberIsOptedOut(phoneNumber string) (bool, error) {
	if err := s.ensureStore(); err != nil {
		return false, err
	}
	return s.smsRepository().IsOptedOut(s.defaultTenant().Key(), phoneNumber)
}

func (s *Service) OptInPhoneNumber(phoneNumber string) error {
	if err := s.ensureStore(); err != nil {
		return err
	}
	entry, err := domain.NewOptOutPhoneNumber(s.defaultTenant(), phoneNumber, false, time.Now().UTC())
	if err != nil {
		return err
	}
	return s.smsRepository().UpsertOptOutPhone(entry)
}

func (s *Service) ListPhoneNumbersOptedOut(nextToken string) ([]string, string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, "", err
	}
	return s.smsRepository().ListOptedOutPhoneNumbers(s.defaultTenant().Key(), nextToken, 100)
}

func (s *Service) ListOriginationNumbers(nextToken string) ([]string, string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, "", err
	}
	return s.smsRepository().ListVerifiedSMSSandboxPhoneNumbers(s.defaultTenant().Key(), nextToken, 100)
}

func (s *Service) GetSMSSandboxAccountStatus() (bool, error) {
	if err := s.ensureStore(); err != nil {
		return false, err
	}
	count, err := s.smsRepository().CountVerifiedSMSSandboxPhoneNumbers(s.defaultTenant().Key())
	if err != nil {
		return false, err
	}
	// Emulated runtime remains in sandbox mode while no verified sender exists.
	return count == 0, nil
}

func (s *Service) CreateSMSSandboxPhoneNumber(phoneNumber, languageCode string) error {
	if err := s.ensureStore(); err != nil {
		return err
	}
	entry, err := domain.NewSMSSandboxPhoneNumber(s.defaultTenant(), phoneNumber, languageCode, time.Now().UTC())
	if err != nil {
		return err
	}
	_, err = s.smsRepository().CreateSMSSandboxPhoneNumber(entry)
	return err
}

func (s *Service) VerifySMSSandboxPhoneNumber(phoneNumber, oneTimePassword string) error {
	if err := s.ensureStore(); err != nil {
		return err
	}

	current, err := s.smsRepository().GetSMSSandboxPhoneNumber(s.defaultTenant().Key(), phoneNumber)
	if err != nil {
		return err
	}
	verified, err := current.Verify(oneTimePassword, time.Now().UTC())
	if err != nil {
		return err
	}
	return s.smsRepository().UpdateSMSSandboxPhoneNumber(verified)
}

func (s *Service) DeleteSMSSandboxPhoneNumber(phoneNumber string) error {
	if err := s.ensureStore(); err != nil {
		return err
	}
	return s.smsRepository().DeleteSMSSandboxPhoneNumber(s.defaultTenant().Key(), phoneNumber)
}

func (s *Service) ListSMSSandboxPhoneNumbers(nextToken string) ([]domain.SMSSandboxPhoneNumber, string, error) {
	if err := s.ensureStore(); err != nil {
		return nil, "", err
	}
	return s.smsRepository().ListSMSSandboxPhoneNumbers(s.defaultTenant().Key(), nextToken, 100)
}
