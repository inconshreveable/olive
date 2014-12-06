package olive

import (
	"net/http"
	"time"

	"github.com/go-martini/martini"
	log "gopkg.in/inconshreveable/log15.v2"
	logext "gopkg.in/inconshreveable/log15.v2/ext"
)

func loggerMiddleware(c martini.Context, req *http.Request, w http.ResponseWriter) {
	start := time.Now()
	l := log.New("pg", req.URL.Path, "id", logext.RandId(8))
	c.MapTo(l, (*log.Logger)(nil))
	l.Info("start")
	c.Next()
	rw := w.(martini.ResponseWriter)
	l.Info("end", "status", rw.Status(), "dur", time.Since(start))
}
