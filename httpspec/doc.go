// Package httpspec provides a reusable suite of HTTP behavior specifications.
//
// Point it at any [http.Handler] to validate that the implementation follows
// standard HTTP conventions: the index page is reachable, unknown paths
// return 404, responses include Content-Type, HEAD and OPTIONS are handled,
// and error responses do not leak internal details.
//
// Each spec runs as a parallel subtest with a human-readable name, producing
// output that reads like a behavior specification document:
//
//	--- PASS: TestHTTPBehavior/the_index_page_should_not_return_404_Not_Found
//	--- PASS: TestHTTPBehavior/unknown_paths_should_return_404_or_a_redirect
//	--- PASS: TestHTTPBehavior/responses_with_a_body_should_include_Content-Type
//
// # Quick start
//
//	func TestHTTPBehavior(t *testing.T) {
//		t.Parallel()
//		handler := myApp.NewHandler()
//		httpspec.Run(t, handler)
//	}
//
// # Skipping inapplicable specs
//
// SPA servers that serve index.html for every path should skip the
// "unknown paths return 404" spec:
//
//	httpspec.Run(t, handler,
//		httpspec.SkipSpec(httpspec.SpecNameUnknownPathReturns404))
//
// # Adding custom specs
//
//	httpspec.Run(t, handler,
//		httpspec.WithExtraSpecs(httpspec.Spec{
//			Name:     "GET /health returns 200",
//			Category: httpspec.CategoryRouting,
//			Check:    httpspec.ExpectStatus(http.MethodGet, "/health", http.StatusOK),
//		}))
package httpspec
