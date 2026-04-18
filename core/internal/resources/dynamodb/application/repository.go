package application

import "github.com/michasdev/mildstack/core/internal/resources/dynamodb/domain"

type Repository interface {
	Load() (domain.State, error)
	Save(state domain.State) error
	Close() error
}
