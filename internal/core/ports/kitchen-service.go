package ports

import "context"

type KitchenServiceInterface interface {
	Start(ctx context.Context) error
}
