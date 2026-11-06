package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/olgapark/avito-labs/internal/http/domainerror"
)

func Logging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		writer := &responseWriter{ResponseWriter: w}
		completed := false
		defer func() {
			status := writer.finalStatus(completed)
			level := slog.LevelInfo
			args := []any{"method", r.Method, "route", r.Pattern, "status", status, "duration_ms", time.Since(started).Milliseconds()}
			if status >= http.StatusInternalServerError {
				level = slog.LevelError
				if state := domainerror.RequestState(r.Context()); state != nil && state.Err != nil {
					args = append(args, "error", state.Err)
				}
			}
			logger.Log(r.Context(), level, "request handled", args...)
		}()
		next.ServeHTTP(writer, r)
		completed = true
	})
}
