package olive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-martini/martini"
	log "gopkg.in/inconshreveable/log15.v2"
	"gopkg.in/stack.v1"
)

// A recoveredPanic is injected into the martini context by the RecoveryMiddleware when
// it recovers from an unhandled panic.
type recoveredPanic struct {
	Cause interface{}
}

// RecoveryMiddleware catches unhandled panics in the handler chain.
// When a panic is recovered from, the onPanic handler is invoked.
// The panic will be injected into the onPanic handler with type *Panic.
// See the DefaultRecoveryMiddleware for an example.
func recoveryMiddleware(onPanic martini.Handler) martini.Handler {
	return func(w http.ResponseWriter, c martini.Context, l log.Logger) {
		defer func() {
			if r := recover(); r != nil {
				c.Map(&recoveredPanic{r})
				c.Invoke(onPanic)
			}
		}()
		c.Next()
	}
}

// Default handler for recovering from unhandled panics. The
// panic cause and stack trace are written to the response and logged.
func defaultRecoveryMiddleware(debugMode bool) martini.Handler {
	return recoveryMiddleware(func(p *recoveredPanic, w http.ResponseWriter, l log.Logger) {
		s := stack.Trace().TrimRuntime()
		l.Crit("handler crashed", "panic", p.Cause, "stack", fmt.Sprintf("%+v", s))
		debugStack := make([]string, 0)
		for _, frame := range s {
			fr := fmt.Sprintf("%+v", frame)
			l.Debug(fr, "panic", p)
			debugStack = append(debugStack, fr)
		}
		if debugMode {
			http.Error(w, fmt.Sprintf("panic: %v\n\n", p.Cause)+strings.Join(debugStack, "\n"), 500)
		} else {
			enc := json.NewEncoder(w)
			enc.Encode(&Error{
				StatusCode: http.StatusInternalServerError,
				Message:    http.StatusText(http.StatusInternalServerError),
			})
		}
	})
}
