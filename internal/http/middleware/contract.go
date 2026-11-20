package middleware

import "context"

type panicLogger interface {
	ErrorContext(ctx context.Context, msg string, args ...any)
}

type requestLogger interface {
	InfoContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}
