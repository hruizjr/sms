package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
)

// maxStackFrames bounds the number of frames recorded for a recovered panic.
const maxStackFrames = 64

// Recover converts a panic in a handler into a logged error and a generic response produced by
// onPanic (typically a 500 problem document).
//
// The raw panic value is never logged or sent to the client, because it may contain connection
// strings, credentials, tokens, or personal data. The log records only the panic value's Go type
// and a stack of function names with file:line positions (see panicStack), which contains no
// argument values. http.ErrAbortHandler is re-panicked so net/http can abort the connection as
// designed.
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
					slog.String("panic_type", fmt.Sprintf("%T", p)),
					slog.String("stack", panicStack()),
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

// panicStack returns the current goroutine's stack as "function\n\tfile:line" entries.
//
// It is built from program counters rather than runtime/debug.Stack, whose output includes the
// raw argument words of every frame (which can carry integers or other request data). Called from
// a deferred recovery function, the stack still includes the frames that panicked.
func panicStack() string {
	pcs := make([]uintptr, maxStackFrames)
	n := runtime.Callers(2, pcs) // skip runtime.Callers and panicStack
	if n == 0 {
		return ""
	}
	frames := runtime.CallersFrames(pcs[:n])

	var b strings.Builder
	for {
		frame, more := frames.Next()
		fmt.Fprintf(&b, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}
	return b.String()
}
