package servertiming

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
	handler := ServerTimingMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stop := MeasureServerTiming(r.Context(), "db")
		stop()
		w.WriteHeader(http.StatusOK)
	}))

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
