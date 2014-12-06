package olive

import (
	"net/http"

	"github.com/go-martini/martini"
	log "gopkg.in/inconshreveable/log15.v2"
)

// Response is a composition of the most common interfaces needed when handling a request.
type Response interface {
	martini.ResponseWriter
	log.Logger

	// Encode uses the negotiated codec to serialize and write the value to the response.
	Encode(v interface{}) error

	// Abort terminates a handler immediately with an error and no further processing is done.
	//
	// If the error is of type *olive.Error, the properties of the *olive.Error will be used to
	// determine the status code and shape of the error response. Otherwise, the response will
	// be a 500 internal server error which includes the error argument as one of its details.
	Abort(error)
}

type response struct {
	martini.ResponseWriter
	enc Encoder
	log.Logger
	*errEncoder
}

// The ResponseMiddleware injects an olive.Response into the martini context
func responseMiddleware() martini.Handler {
	return func(w http.ResponseWriter, enc Encoder, l log.Logger, e *errEncoder, c martini.Context) {
		c.MapTo(&response{w.(martini.ResponseWriter), enc, l, e}, (*Response)(nil))
	}
}

func (r *response) Encode(v interface{}) error {
	return r.enc.Encode(r.ResponseWriter, v)
}
