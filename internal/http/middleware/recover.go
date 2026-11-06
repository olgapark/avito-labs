package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/olgapark/avito-labs/internal/http/domainerror"
)

func Recover(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, state := domainerror.WithState(r.Context())
		r = r.WithContext(ctx)
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(state.Context, "request panic", "panic", recovered, "stack", string(debug.Stack()))
				domainerror.Write(w, r, errors.New("request panic"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
