package olive

import (
	"net/http"

	"github.com/go-martini/martini"
	log "github.com/inconshreveable/log15/v3"
)

// A structure with details about an error that occurred while handling a request.
// Passing this structure to an ErrEncoder's Abort() method gives the caller complete control
// over the error response shape and status code.
type Error struct {
	ErrorCode  int    `json:"error_code,omitempty" xml:",omitempty"` // unique error code
	StatusCode int    `json:"status_code"`                           // http status code
	Message    string `json:"msg"`                                   // user-facing error message
	Details    M      `json:"details" xml:"-"`                       // extra error context for client, XXX should work in XML
}

func (e *Error) Error() string {
	return e.Message
}

// a Map of extra error details
type M map[string]interface{}

// errEncoderMiddleware injects an ErrEncoder into the martini context
// ErrEncoderMiddleware is automatically included in the middleware chain for
// all olive API endpoints.
func errEncoderMiddleware(debug bool) martini.Handler {
	return func(c martini.Context, w http.ResponseWriter, enc Encoder, l log.Logger) {
		defer func() {
			if p := recover(); p != nil {
				if _, ok := p.(abort); ok {
					return
				}
				panic(p)
			}
		}()
		c.Map(&errEncoder{enc: enc, l: l, w: w.(martini.ResponseWriter), debug: debug})
		c.Next()
	}
}

// abort is raised by the error functions
type abort struct{}

type errEncoder struct {
	enc   Encoder
	l     log.Logger
	w     martini.ResponseWriter
	debug bool
}

func (e *errEncoder) abort(err error) {
	apiErr, ok := err.(*Error)
	if !ok {
		apiErr = internalServerError(err)
	}

	logDetails := log.Ctx(apiErr.Details)
	if logDetails == nil {
		logDetails = log.Ctx{}
	}

	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(apiErr.StatusCode)
	}

	// log the error
	logFn := e.l.Warn
	if apiErr.StatusCode == http.StatusInternalServerError {
		logFn = e.l.Error
		if !ok && !e.debug {
			apiErr.Details = nil
		}
	}
	logFn(apiErr.Message, logDetails)

	// write to response only if nothing else has been written
	if !e.w.Written() {
		e.w.WriteHeader(apiErr.StatusCode)
		e.enc.Encode(e.w, apiErr)
	}
}

func (e *errEncoder) Abort(err error) {
	e.abort(err)
	panic(abort{})
}

func internalServerError(err error) *Error {
	return &Error{
		StatusCode: http.StatusInternalServerError,
		Details:    M{"err": err.Error()},
	}
}
