package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recover converts a panic in a handler into a logged error and a generic response produced by
// onPanic (typically a 500 problem document).
//
// The panic value and stack trace are logged server-side only; they are never sent to the
// client. http.ErrAbortHandler is re-panicked so net/http can abort the connection as designed.
func Recover(logger *slog.Logger, onPanic http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				p := recover()
				if p == nil {
					return
				}
				if err, ok := p.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(p)
				}
				logger.LogAttrs(r.Context(), slog.LevelError, "panic recovered",
					slog.String("request_id", RequestIDFromContext(r.Context())),
					slog.String("method", r.Method),
					slog.String("route", RoutePattern(r)),
					slog.String("panic", fmt.Sprint(p)),
					slog.String("stack", string(debug.Stack())),
				)
				if rec, ok := w.(*statusRecorder); ok && rec.wroteHeader {
					// The response has already started; nothing more can be sent safely.
					return
				}
				onPanic.ServeHTTP(w, r)
			}()
			next.ServeHTTP(w, r)
		})
	}
}
