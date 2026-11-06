package handler

import (
	"context"
	"encoding/json"
	"net/http"

	api "github.com/olgapark/avito-labs/internal/generated"
)

func (h *Handler) Ready(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(
		r.Context(),
		h.queryTimeout,
	)
	defer cancel()

	if err := h.pool.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)

		_ = json.NewEncoder(w).Encode(api.HealthResponse{
			Status: api.Unavailable,
		})
		return
	}

	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(api.HealthResponse{
		Status: api.Ok,
	})
}
