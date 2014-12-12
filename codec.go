package olive

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/goji/param"
	log "gopkg.in/inconshreveable/log15.v2"
)

type Decoder interface {
	Decode(rd io.Reader, v interface{}) error
}

type decoderFunc func(io.Reader, interface{}) error

func (f decoderFunc) Decode(rd io.Reader, v interface{}) error {
	return f(rd, v)
}

var (
	jsonDecoder = decoderFunc(func(rd io.Reader, v interface{}) error {
		return json.NewDecoder(rd).Decode(v)
	})
	xmlDecoder = decoderFunc(func(rd io.Reader, v interface{}) error {
		return xml.NewDecoder(rd).Decode(v)
	})
	formDecoder = decoderFunc(func(rd io.Reader, v interface{}) error {
		buf, err := ioutil.ReadAll(rd)
		if err != nil {
			return err
		}
		vals, err := url.ParseQuery(string(buf))
		if err != nil {
			return err
		}
		return param.Parse(vals, v)
	})
)

type Encoder interface {
	Encode(wr io.Writer, v interface{}) error
}

type encoderFunc func(io.Writer, interface{}) error

func (f encoderFunc) Encode(wr io.Writer, v interface{}) error {
	return f(wr, v)
}

var (
	jsonEncoder = encoderFunc(func(wr io.Writer, v interface{}) error {
		return json.NewEncoder(wr).Encode(v)
	})
	prettyJSONEncoder = encoderFunc(func(wr io.Writer, v interface{}) error {
		buf, err := json.MarshalIndent(v, "", "    ")
		if err != nil {
			return err
		}
		_, err = wr.Write(buf)
		return err
	})
	xmlEncoder = encoderFunc(func(wr io.Writer, v interface{}) error {
		return xml.NewEncoder(wr).Encode(v)
	})
)

// safeEncoder wraps an encoder to write out an error
// response if the wrapped encoder fails for any reason
// (e.g. failed xml serialization)
func safeEncoder(e Encoder, l log.Logger) Encoder {
	return encoderFunc(func(wr io.Writer, v interface{}) error {
		err := e.Encode(wr, v)
		if err == nil {
			return nil
		}
		l.Error("failed to encode response", "err", err)
		// XXX: the status code isn't set appropriately here
		e.Encode(wr, &Error{
			StatusCode: http.StatusInternalServerError,
			Message:    "failed to encode response",
			Details:    M{"err": err},
		})
		return err
	})
}
