package services

import (
	"encoding/json"
	"net/http"
)

// ---------- Helpers ----------

func WriteJSON(w http.ResponseWriter, v interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
