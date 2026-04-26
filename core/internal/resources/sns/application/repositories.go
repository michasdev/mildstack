package application

import "github.com/michasdev/mildstack/core/internal/resources/sns/infrastructure"

func (s *Service) adminRepository() infrastructure.AdminRepository {
	return infrastructure.NewAdminRepository(s.store)
}

func (s *Service) platformRepository() infrastructure.PlatformRepository {
	return infrastructure.NewPlatformRepository(s.store)
}

func (s *Service) smsRepository() infrastructure.SMSRepository {
	return infrastructure.NewSMSRepository(s.store)
}
