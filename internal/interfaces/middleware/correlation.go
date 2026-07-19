package middleware

import (
	"net/http"
	"regexp"

	applogger "github.com/FelipePn10/panossoerp/internal/infrastructure/logger"
	"github.com/google/uuid"
)

const correlationIDHeader = "X-Request-ID"

var validCorrelationID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

// CorrelationMiddleware ensures every request carries a unique ID throughout
// its entire lifecycle — from the moment it enters the API to the response
// sent back to the client.
//
// Behaviour:
//  1. If the caller already sent an X-Request-ID header, that value is reused
//     (useful for tracing calls that originate from another service).
//  2. Otherwise a new UUID v4 is generated.
//  3. The ID is stored in the request context (retrievable via
//     logger.CorrelationIDFromContext) and echoed back in the response header.
func CorrelationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(correlationIDHeader)
		if !validCorrelationID.MatchString(id) {
			id = uuid.NewString()
		}

		// Propagate forward (context) and backward (response header).
		w.Header().Set(correlationIDHeader, id)
		ctx := applogger.WithCorrelationID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
