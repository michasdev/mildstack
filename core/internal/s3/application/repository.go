package application

import "github.com/michasdev/mildstack/core/internal/s3/domain"

type Repository interface {
	Load() (domain.State, error)
	Save(state domain.State) error
}
