# olive - REST APIs with Martini [![godoc reference](https://godoc.org/gopkg.in/inconshreveable/olive.v0?status.png)](https://godoc.org/gopkg.in/inconshreveable/olive.v0)

olive is tiny, opinonated scaffolding to rapidly build standards-compliant REST APIs on top of Martini.
olive handles content negotiation, parameter deserialization, request-scoped logging, panic recovery,
and RFC-appropriate error responses so that you can focus on writing your business logic.

Because it's just Martini, you can easily plug in any middleware from the martini community for
additional functionality like setting CORS headers.

olive is currently experimental.

Here's a simple API which calculates factorials that handles input and output in JSON or XML.

```go
package main
import (
    "net/http"
    "github.com/inconshreveable/olive"
)

func main() {
    o := olive.Martini()
    o.Debug = true
    o.Post("/fact", o.Endpoint(factorial).Param(Input{}))
    http.ListenAndServe(":8080", o)
}

type Input struct {
    Num     int `json:"num" xml:"Num"`
    Timeout int `json:"timeout" xml:"Timeout"`
}

type Output struct {
    Factorial int `json:"answer" xml:"Answer"`
}

func factorial(r olive.Response, in *Input) {
    r.Info("computing factorial", "num", in.Num, "timeout", in.Timeout)
    ans, err := computeFactorial(in.Num, time.Duration(in.Timeout) * time.Second)
    if err != nil {
        r.Abort(err)
    }
    r.Encode(Output{Factorial: ans})
}
```

### olive.Response
All olive endpoint handlers are injected with an `olive.Response` object with
methods that makes it easy to respond to the caller.

The most common way you will do this is with `Encode` method. This serializes
a structure by automatically choosing an appropriate `ContentEncoder` based on the
requests `Accept` header. By default, olive only understands how to serialize JSON
and XML, but it can be extended to handle arbitrary content types.

```go
func helloWorldEndpoint(r olive.Response) {
    r.Encode([]string{"hello", "world"})
}
```

`olive.Response` also includes an `Abort()` function for terminating a handler when
an error has been encountered:

```go
func handler(r olive.Response) {
    if err := doSomething(); err != nil {
        r.Abort(err)
    }
}
```

The `olive.Response` interface also embeds a log15.Logger for easy logging and the http.ResponseWriter
interface for writing custom status codes:

```go
func handler(r olive.Response) {
    r.Warn("about to write a failing response", "code", 404)
    r.WriteHeader(404)
}
```

### Param injection
When you create an olive endpoint, you can specify a Param that it expects. If you do,
olive will attempt to deserialize the request body by examing the Content-Type 
and then map it into the structure you passed as the Param. It then takes the mapped structure
and injects it into the `martini.Context` for easy use in your handler.

```go
func main() {
    o := olive.Martini()
    o.Post("/table", o.Endpoint(createTable).Param(NewTable{}))
    // etc
}

type NewTable struct {
    Width int
    Height int
    Depth int
}
func createTable(r olive.Response, nt *NewTable) {
    // create the table and respond
}
```
