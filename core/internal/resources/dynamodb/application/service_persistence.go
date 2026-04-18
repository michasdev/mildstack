package application

import "fmt"

func NewWithPersistence(config StorageConfig) (*Service, error) {
	storagePath, err := ResolveStoragePath(config)
	if err != nil {
		return nil, err
	}

	repo, err := NewSQLiteRepository(storagePath)
	if err != nil {
		return nil, err
	}

	state, err := repo.Load()
	if err != nil {
		_ = repo.Close()
		return nil, fmt.Errorf("dynamodb: load persisted state: %w", err)
	}

	return newService(state, repo), nil
}
