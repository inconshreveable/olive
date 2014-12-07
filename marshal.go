package olive

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-martini/martini"
	"github.com/goji/param"
	log "gopkg.in/inconshreveable/log15.v2"
)

func unmarshalMiddleware(decoders map[string]Decoder, inputParam interface{}) martini.Handler {
	return func(r *http.Request, c martini.Context, e *errEncoder) {
		// skip if there's no input
		if !reflect.ValueOf(inputParam).IsValid() {
			return
		}

		// copy param
		paramPtr := reflect.New(reflect.ValueOf(inputParam).Type()).Interface()

		// GET handlers always pull their parameters from the URL
		if r.Method == "GET" {
			err := param.Parse(r.URL.Query(), paramPtr)
			if err != nil {
				e.Abort(decodeFailure(err))
			}
		} else {
			ce := getCE(r.Method, r.Header.Get("Content-Type"))
			switch ce {
			// XXX: allow this to be pluggable
			case "utf8", "utf-8", "":
			default:
				e.Abort(unsupportedMediaTypeCE(ce))
			}
			ct, _ := split(r.Header.Get("Content-Type"), ";")
			dec, ok := decoders[ct]
			if !ok {
				e.Abort(unsupportedMediaType(ct, decoders))
			}
			err := dec.Decode(r.Body, paramPtr)
			if err != nil {
				e.Abort(decodeFailure(err))
				return
			}
		}
		c.Map(paramPtr)
	}
}

func decodeFailure(err error) *Error {
	return &Error{
		StatusCode: http.StatusBadRequest,
		Message:    "failed to deserialize request parameter",
		Details:    M{"err": err},
	}
}

func unsupportedMediaType(contentType string, decoderMap map[string]Decoder) *Error {
	available := make([]string, 0)
	for k, _ := range decoderMap {
		available = append(available, k)
	}
	return &Error{
		StatusCode: http.StatusUnsupportedMediaType,
		Message:    "unsupported request Content-Type",
		Details:    M{"content-type": contentType, "available": available},
	}
}

func unsupportedMediaTypeCE(encoding string) *Error {
	return &Error{
		StatusCode: http.StatusUnsupportedMediaType,
		Message:    "unsupported request content encoding charset",
		Details:    M{"charset": encoding, "available": []string{"UTF-8"}},
	}
}

func marshalMiddleware(encoders []ContentEncoder) martini.Handler {
	return func(w http.ResponseWriter, r *http.Request, c martini.Context, l log.Logger) {
		accept := r.Header.Get("Accept")
		if accept == "" {
			accept = "*/*"
		}
		var (
			bestQ       float64
			bestEncoder ContentEncoder
		)
		for _, enc := range encoders {
			q := accepts(accept, enc.ContentType)
			if q > bestQ {
				bestQ = q
				bestEncoder = enc
			}
		}
		if bestQ == 0 {
			// error reporter injecting middleware comes after the Marshaller,
			// so construct our own with JSON
			w.Header().Set("Content-Type", "application/json")
			e := errEncoder{jsonEncoder, l, w.(martini.ResponseWriter), false}
			e.abort(notAcceptable(accept, encoders))
		}
		c.MapTo(safeEncoder(bestEncoder, l), (*Encoder)(nil))
		w.Header().Set("Content-Type", bestEncoder.ContentType)
	}
}

func notAcceptable(acceptHeader string, encoders []ContentEncoder) *Error {
	available := make([]string, 0)
	for _, e := range encoders {
		available = append(available, e.ContentType)
	}
	return &Error{
		StatusCode: http.StatusNotAcceptable,
		Message:    "unsupported request Accept header",
		Details:    M{"accept-header": acceptHeader, "supported": available},
	}
}

// RFC2616 header parser (simple version).
func accepts(a, ctype string) (q float64) {
	if a == ctype || a == "*/*" || a == "" {
		// bail out in some common cases
		return 1
	}
	cGroup, cType := split(ctype, "/")
	for _, field := range strings.Split(a, ",") {
		found, match := false, false
		for i, token := range strings.Split(field, ";") {
			if i == 0 {
				// token is "type/subtype", "type/*" or "*/*"
				aGroup, aType := split(token, "/")
				if cType == aType || aType == "*" {
					if (aGroup == "*" && aType == "*") || cGroup == aGroup {
						// token matches, continue to look for a q value
						found = true
						continue
					}
				}
				break
			}
			// token is "key=value"
			k, v := split(token, "=")
			if k != "q" {
				continue
			}
			// k is "q"
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				break
			}
			if q < f {
				q = f
			}
			match = true
			break
		}
		if found && !match {
			q = 1
			break
		}
	}
	return
}

// Check the content encoding against a list of acceptable values.
func getCE(meth string, ce string) string {
	if !(meth == "POST" || meth == "PATCH" || meth == "PUT") {
		return ""
	}
	_, ce = split(strings.ToLower(ce), ";")
	_, ce = split(ce, "charset=")
	ce, _ = split(ce, ";")
	return ce
}

// Split a string in two parts, cleaning any whitespace.
func split(str, sep string) (a, b string) {
	parts := strings.SplitN(str, sep, 2)
	a = strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		b = strings.TrimSpace(parts[1])
	}
	return
}
