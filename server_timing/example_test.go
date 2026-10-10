package servertiming

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

func ExampleNewServerTiming() {
	st := NewServerTiming()
	st.Record("db", "Database", 53*time.Millisecond)
	st.Record("cache", "", 2*time.Millisecond)

	fmt.Println(st.HeaderValue())

	// Output: db;desc="Database";dur=53, cache;dur=2
}

func ExampleServerTimingMiddleware() {
	handler := ServerTimingMiddleware()(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			stop := MeasureServerTiming(r.Context(), "db")
			stop()
			w.WriteHeader(http.StatusOK)
		}),
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	fmt.Println(rec.Code)
	fmt.Println(rec.Header().Get(HeaderServerTiming) != "")

	// Output:
	// 200
	// true
}

func ExampleWrapServerTiming() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RecordServerTiming(r.Context(), "db", "", 0)
		_, _ = w.Write([]byte("ok"))
	})

	rec := httptest.NewRecorder()
	wrapped, r := WrapServerTiming(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	handler.ServeHTTP(wrapped, r)

	fmt.Println(rec.Code, rec.Body.String())
	fmt.Println(rec.Header().Get(HeaderServerTiming) != "")

	// Output:
	// 200 ok
	// true
}

func ExampleServerTiming_MeasureWithDesc() {
	st := NewServerTiming()

	stop := st.MeasureWithDesc("db", "Main query")
	stop()

	// The duration is wall-clock, so only the stable prefix is asserted.
	fmt.Println(strings.HasPrefix(st.HeaderValue(), `db;desc="Main query";dur=`))

	// Output: true
}

func ExampleServerTimingFromContext() {
	// Outside the middleware no collector is present: the lookup returns nil,
	// and every method on the nil *ServerTiming is a no-op, so handlers can
	// record without nil checks.
	fmt.Println(ServerTimingFromContext(context.Background()) == nil)
	RecordServerTiming(context.Background(), "db", "", 0)

	// Output: true
}
