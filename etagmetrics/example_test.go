package etagmetrics_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	etag "github.com/larsartmann/go-etag/server"
	"github.com/larsartmann/httputil/etagmetrics"
)

func ExampleAttach() {
	cfg, counters := etagmetrics.Attach(etag.DefaultETagConfig())
	handler := etag.New(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	fmt.Println(rec.Code)

	snap := counters.Snapshot()
	fmt.Println(snap.Generated, snap.NotModified, snap.BufferOverflows)

	// Output:
	// 200
	// 1 0 0
}
