package handlers

import "net/http"

func PostOrder(w http.ResponseWriter, r *http.Request) {
	hello := []byte("hello")
	w.Write(hello)
}
