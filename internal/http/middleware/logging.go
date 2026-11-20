package middleware

import (
	"net/http"
	"time"

	"github.com/olgapark/avito-labs/internal/http/domainerror"
)

func Logging(logger requestLogger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		writer := &responseWriter{ResponseWriter: w}
		completed := false
		defer func() {
			status := writer.finalStatus(completed)
			args := []any{"method", r.Method, "route", r.Pattern, "status", status, "duration_ms", time.Since(started).Milliseconds()}
			if status >= http.StatusInternalServerError {
				if state := domainerror.RequestState(r.Context()); state != nil && state.Err != nil {
					args = append(args, "error", state.Err)
				}
				logger.ErrorContext(r.Context(), "request handled", args...)
				return
			}
			logger.InfoContext(r.Context(), "request handled", args...)
		}()
		next.ServeHTTP(writer, r)
		completed = true
	})
}
