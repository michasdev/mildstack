package application

import "fmt"

func NewWithPersistence(config StorageConfig) (*Service, error) {
	storagePath, err := ResolveStoragePath(config)
	if err != nil {
		return nil, err
	}

	repo := NewFSRepository(storagePath)
	state, err := repo.Load()
	if err != nil {
		return nil, fmt.Errorf("s3: load persisted state: %w", err)
	}

	return newService(state, repo), nil
}

func (s *Service) persist() error {
	if s.repo == nil {
		return nil
	}
	if err := s.repo.Save(s.state); err != nil {
		return fmt.Errorf("s3: persist state: %w", err)
	}
	return nil
}
