package handler

import (
	"encoding/json"
	"net/http"

	api "github.com/olgapark/avito-labs/internal/generated"
)

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: api.Ok})
}
