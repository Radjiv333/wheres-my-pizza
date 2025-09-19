package ports

import (
	"context"
	"net/http"
)

type OrderServiceInterface interface {
	Stop(ctx context.Context, server *http.Server)
	PostOrder(w http.ResponseWriter, r *http.Request)
}
