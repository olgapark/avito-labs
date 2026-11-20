package handler

import (
	"encoding/json"
	"net/http"

	api "github.com/olgapark/avito-labs/internal/generated"
)

func HandleAPIError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusBadRequest)

	_ = json.NewEncoder(w).Encode(api.Problem{
		Instance: &r.URL.Path,
		Type:     "https://tripgo.example/problems/invalid-request",
		Title:    "Invalid request",
		Status:   http.StatusBadRequest,
		Code:     "invalid_request",
	})
}
