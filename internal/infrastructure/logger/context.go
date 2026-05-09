package logger

import "context"

// Private context keys — unexported structs prevent collisions with other packages.
type loggerCtxKey struct{}
type correlationIDCtxKey struct{}
type userIDCtxKey struct{}

// WithLogger stores l in ctx so handlers can retrieve it via FromContext.
func WithLogger(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, l)
}

// WithCorrelationID stores the request ID in ctx.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDCtxKey{}, id)
}

// CorrelationIDFromContext retrieves the correlation/request ID from ctx.
// Returns "" if not set.
func CorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDCtxKey{}).(string); ok {
		return id
	}
	return ""
}

// WithUserID stores the authenticated user ID in ctx.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDCtxKey{}, id)
}

// UserIDFromContext retrieves the user ID from ctx.
func UserIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(userIDCtxKey{}).(string); ok {
		return id
	}
	return ""
}
