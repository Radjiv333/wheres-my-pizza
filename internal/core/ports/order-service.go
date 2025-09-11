package ports

import "net/http"

type OrderServiceInterface interface {
	Stop()
	PostOrder(w http.ResponseWriter, r *http.Request)
}
