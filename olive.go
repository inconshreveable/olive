package olive

import (
	"net/http"
	"strings"

	"github.com/go-martini/martini"
)

// A ContentEncoder is an Encoder that can encode a resource to the
// representation described by the ContentType mimetype. Requests with an Accept
// header that match the ContentType will use that ContentEncoder.
type ContentEncoder struct {
	ContentType string
	Encoder
}

// Olive creates API Endpoints. Customizing the properties of the Olive
// changes the defaults of the created Endpoints.
type Olive struct {
	rt       martini.Router
	Encoders []ContentEncoder   // default set of ContentEncoders used by a new Endpoint
	Decoders map[string]Decoder // default map of Decoders used by a new Endpoint
	Debug    bool               // default debug flag of a new Endpoint
}

func (o *Olive) fwd(rtfn func(string, ...martini.Handler) martini.Route, pattern string, e Endpoint) martini.Route {
	return rtfn(pattern, e.Handlers()...)
}

func (o *Olive) Get(pattern string, e Endpoint) martini.Route {
	return o.fwd(o.rt.Get, pattern, e)
}

func (o *Olive) Post(pattern string, e Endpoint) martini.Route {
	return o.fwd(o.rt.Post, pattern, e)
}

func (o *Olive) Put(pattern string, e Endpoint) martini.Route {
	return o.fwd(o.rt.Put, pattern, e)
}

func (o *Olive) Patch(pattern string, e Endpoint) martini.Route {
	return o.fwd(o.rt.Patch, pattern, e)
}

func (o *Olive) Delete(pattern string, e Endpoint) martini.Route {
	return o.fwd(o.rt.Delete, pattern, e)
}

func (o *Olive) Options(pattern string, e Endpoint) martini.Route {
	return o.fwd(o.rt.Options, pattern, e)
}

func (o *Olive) Head(pattern string, e Endpoint) martini.Route {
	return o.fwd(o.rt.Head, pattern, e)
}

func (o *Olive) Any(pattern string, e Endpoint) martini.Route {
	return o.fwd(o.rt.Any, pattern, e)
}

// Returns a new Olive API creating endpoints that can be mapped onto
// the given martini.Router.
//
//     rt := martini.NewRouter()
//     o := olive.New(rt)
//     e := o.Endpoint(showTables)
//     rt.Get(e.Handlers()...)
func New(rt martini.Router) *Olive {
	o := &Olive{
		rt: rt,
		Encoders: []ContentEncoder{
			{"text/html", prettyJSONEncoder},
			{"application/json", jsonEncoder},
			{"text/xml", xmlEncoder},
			{"application/xml", xmlEncoder},
		},
		Decoders: map[string]Decoder{
			"application/json":                  jsonDecoder,
			"text/xml":                          xmlDecoder,
			"application/xml":                   xmlDecoder,
			"application/x-www-form-urlencoded": formDecoder,
		},
	}
	rt.NotFound(o.Endpoint(noRouteHandler).Handlers()...)
	return o
}

func (o *Olive) Endpoint(hs ...martini.Handler) Endpoint {
	return &endpoint{
		rt:       o.rt,
		decs:     o.Decoders,
		encs:     o.Encoders,
		debug:    o.Debug,
		handlers: hs,
	}
}

// An Endpoint describes an Endpoint in an olive REST API. Callers may
// customize an endpoint's behavior by chaining calls that manipulate its state.
// After the Endpoint is built, the caller can use the Handlers() function to get
// the set of martini.Handlers that implement the API endpoint.
//
//     o := olive.Martini()
//     e := o.Endpoint(listTables).Param(TableFilter{}).Debug(true)
//     o.Get("/tables", e.Handlers()...)
type Endpoint interface {
	// stucture of the request input, deserialized either from the request body or query string
	// if set, a pointer to a value of this type will be dependency-injected into the handler
	Param(interface{}) Endpoint

	// overload the allowed decoders
	Decoders(map[string]Decoder) Endpoint

	// customize the allowed encoders for this endpoint
	Encoders([]ContentEncoder) Endpoint

	// debug determines if error stack traces are printed to the client
	Debug(bool) Endpoint

	// returns the handlers that make up the endpoint
	Handlers() []martini.Handler
}

type endpoint struct {
	rt       martini.Router
	param    interface{}
	decs     map[string]Decoder
	encs     []ContentEncoder
	debug    bool
	handlers []martini.Handler
}

func (e *endpoint) Decoders(decoders map[string]Decoder) Endpoint { e.decs = decoders; return e }
func (e *endpoint) Encoders(encoders []ContentEncoder) Endpoint   { e.encs = encoders; return e }
func (e *endpoint) Param(p interface{}) Endpoint                  { e.param = p; return e }
func (e *endpoint) Debug(debug bool) Endpoint                     { e.debug = debug; return e }
func (e *endpoint) Handlers() []martini.Handler {
	return append([]martini.Handler{
		mapRoutes(e.rt),
		loggerMiddleware,
		defaultRecoveryMiddleware(e.debug),
		marshalMiddleware(e.encs),
		errEncoderMiddleware(e.debug),
		unmarshalMiddleware(e.decs, e.param),
		responseMiddleware(),
	}, e.handlers...)
}

func noRouteHandler(r Response, req *http.Request, rts martini.Routes) {
	if methods := rts.MethodsFor(req.URL.Path); len(methods) > 0 {
		allowed := strings.Join(methods, ", ")
		r.Header().Set("Allow", allowed)
		r.Abort(&Error{
			StatusCode: http.StatusMethodNotAllowed,
			Details:    M{"method": req.Method, "allowed": allowed},
		})
	} else {
		r.Abort(&Error{
			StatusCode: http.StatusNotFound,
			Details:    M{"path": req.URL.Path},
		})
	}
}

func mapRoutes(rt martini.Routes) martini.Handler {
	return func(c martini.Context) {
		c.MapTo(rt, (*martini.Routes)(nil))
	}
}

// A convenient pairing of an Olive and Martini which
// can be used to define and customize an Olive API.
type OliveMartini struct {
	*martini.Martini
	*Olive
	martini.Router
}

// Returns an *OliveMartini that has both an Olive router and *martini.Martini
// appropriately wired together and ready for use.
func Martini() *OliveMartini {
	m := martini.New()
	rt := martini.NewRouter()
	o := New(rt)
	m.Action(rt.Handle)
	return &OliveMartini{m, o, rt}
}
