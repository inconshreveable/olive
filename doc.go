/*
olive is a a tiny framework built on top of martini for rapid
development of robust REST APIs.

olive handles content type negotation, serialization, deserialization,
unique request-id logging and panic recovery leaving you free to write
just your application's business logic.

Simple example:

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

The above API will appropriately deserialize a POST body of XML, JSON,
or x-www-form-urlencoded depending on the Content-Type header. Based
on the client's Accept header, the result will be serialized in either
XML or JSON. Appropriate failures are returned for invalid client requests.
The logger assigns a unique id to each request for easy tracing purposes:

	INFO[11-21|15:33:58] start                                    pg=/fact id=e416b6cc83f386bc
	INFO[11-21|15:33:58] computing factorial                      pg=/fact id=e416b6cc83f386bc num=4 timeout=5
	INFO[11-21|15:33:58] end                                      pg=/fact id=e416b6cc83f386bc status=200 dur=371.98us

A more advanced example explaining features in detail:

	package main

	import (
		"net/http"

		"github.com/go-martini/martini"
		"github.com/inconshreveable/olive"
	)

	func main() {
		o := olive.Martini()
		o.Post("/accounts", o.Endpoint(createAccount).Param(CreateAccountParam{}))
		o.Get("/accounts", o.Endpoint(getAccounts).Param(GetAccountsParam{}))
		o.Get("/accounts/:id", o.Endpoint(getAccount)).Name("accountInstance")

		// serve the API
		http.ListenAndServe(":8080", o)
	}

	// This is the expected request payload for the createAccount endpoint
	// It will automatically be deserialized appropriately depending on
	// the Content-Type header sent by the client
	type CreateAccountParam struct {
		Name  string `json:"name" xml:"Name"`
		Email string `json:"email" xml:"Email"`
	}

	// If a struct is specified with Endpoint's Param() function, the request body
	// is deserialized and a pointer to the result is dependency injected
	func createAccount(r olive.Response, param *CreateAccountParam) {
		// this is all business logic
		ac, err := account.Create(param.Name, param.Email)
		if err != nil {
			// Abort fails the request immediately, there is no need to return.
			// If the standard 'error' interface is passed in, we return a
			// 500 internal server error. see below for fine-grained control
			r.Abort(err)
		}

		// custom status codes need a call to WriteHeader first
		r.WriteHeader(201)

		// serialize output
		r.Encode(ac)
	}

	type GetAccountsParam struct {
		Email string `param:"email"`
	}

	// Unlike createAccount, GetAccountsParam is deserialized from the query URI instead
	// of from the request body because this is a GET request
	func getAccounts(r olive.Response, param *GetAccountsParam) {
		acs, err := account.GetAccountsForEmail(param.Email)
		if err != nil {
			r.Abort(err)
		}

		// the Response interface embeds a log15.Logger that you can use for easy logging
		// every request has a unique ID included in the log line
		r.Debug("fetched accounts", "email", param.Email, "count", len(acs))

		r.Encode(acs)
	}

	func getAccount(r olive.Response, p martini.Params) {
		// access to URL parameters is the same as Martini
		accountId := p["id"]
		s, err := account.GetById(accountId)
		switch {
		case err == account.NotFoundError:
			// if you pass an olive.Error to Abort(), you can exert more sophisticated
			// control over the returned response
			r.Abort(&olive.Error{
				StatusCode: 404, // http status code
				ErrorCode:  102, // unique error code for this failure ("account not found")
				Message:    "account not found",
				Details:    olive.M{"id": accountId},
			})
		case err != nil:
			r.Abort(err)
		}
		r.Encode(s)
	}
*/
package olive
