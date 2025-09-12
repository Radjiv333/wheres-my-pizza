package ports

import (
	"context"

	"wheres-my-pizza/internal/core/domain"
)

type RepositoryInterface interface {
	InsertOrder(ctx context.Context, order *domain.Order) (string, error)
}
