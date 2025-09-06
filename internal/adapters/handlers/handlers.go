package handlers

import (
	"encoding/json"
	"net/http"

	"wheres-my-pizza/internal/core/domain"
)

func PostOrder(w http.ResponseWriter, r *http.Request) {
	var order domain.Order
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		http.Error(w, "Cannot decode the order", http.StatusBadRequest)
	}
	defer r.Body.Close()

	response := domain.PutOrderResponse{OrderNumber: "", Status: "", TotalAmount: 234.234}

	responseByte, err := json.Marshal(response)
	if err != nil {
		
	}
	w.Write(responseByte)
	w.WriteHeader(http.StatusOK)
}
