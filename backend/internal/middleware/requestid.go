package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// RequestIDHeader is the header used to accept and return the request correlation ID.
const RequestIDHeader = "X-Request-ID"

// maxRequestIDLength bounds client-supplied request IDs.
const maxRequestIDLength = 128

type requestIDKey struct{}

// IDGenerator returns a new request ID.
type IDGenerator func() string

// RequestID assigns every request a correlation ID, stores it in the request context, and returns
// it in the X-Request-ID response header.
//
// A client-supplied X-Request-ID is reused only when it is 1-128 characters of [A-Za-z0-9._-];
// anything else is replaced, so that untrusted input cannot inject content into logs.
func RequestID(generate IDGenerator) func(http.Handler) http.Handler {
	if generate == nil {
		generate = NewRandomID
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(RequestIDHeader)
			if !validRequestID(id) {
				id = generate()
			}
			w.Header().Set(RequestIDHeader, id)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
		})
	}
}

// RequestIDFromContext returns the request ID stored by RequestID, or "" when there is none.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// NewRandomID returns 128 random bits encoded as 32 hexadecimal characters.
func NewRandomID() string {
	var b [16]byte
	// crypto/rand.Read never returns an error on supported platforms (it panics instead).
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLength {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '_', c == '-':
		default:
			return false
		}
	}
	return true
}
