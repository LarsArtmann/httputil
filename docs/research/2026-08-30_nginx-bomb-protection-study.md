# Study — nginx vs `httputil.Decompression` Bomb Protection

**Date:** 2026-08-30
**Source item:** `05-45:f48` ("nginx bomb-protection study — compare nginx behavior with Decompression bomb protection")
**Sources:** nginx.org official docs (fetched and quoted 2026-08-30): [ngx_http_gunzip_module](https://nginx.org/en/docs/http/ngx_http_gunzip_module.html), [ngx_http_core_module §client_max_body_size](https://nginx.org/en/docs/http/ngx_http_core_module.html#client_max_body_size), §large_client_header_buffers.

## Question

If a Go service sits behind nginx, does the proxy already protect against gzip decompression bombs in **request bodies**, or is `Decompression`'s `MaxDecompressionSize` doing work the proxy does not?

## What nginx actually does (verified against docs)

1. **nginx does not decompress request bodies at all.** The only gunzip filter, `ngx_http_gunzip_module`, is described as "a filter that decompresses **responses** with `Content-Encoding: gzip` for **clients** that do not support `gzip`" — i.e. response-side only, and "not built by default". There is no module that inflates a client's gzip-encoded request body inside nginx.
2. **`client_max_body_size` measures wire bytes, not inflated bytes.** Doc text: "Sets the maximum allowed size of the client request body. If the size in a request exceeds the configured value, the 413 (Request Entity Too Large) error is returned". For a chunked or `Content-Encoding: gzip` request, the check bounds the compressed bytes on the wire. A 10 MB bomb that inflates to 10 GB passes a 1 MB default `client_max_body_size` only if... it does not pass by size — but compression ratios of 1000:1 are trivially achievable, so any bomb small enough to satisfy the wire limit still inflates ~1000x past it on the backend side.
3. **nginx IS decompression-aware for HTTP/2 headers** — `large_client_header_buffers` notes that with HPACK Huffman encoding "the actual size of decompressed name and value strings may be larger. The maximum size of the entire request header is limited **after** decompression." So the nginx authors apply decompressed-size limiting exactly where they decompress (headers), and apply none where they don't (request bodies).

## Consequence: the bomb passes through to the app

A reverse-proxied deployment gets **no bomb protection from nginx** for gzipped request bodies. The compressed body is forwarded (`proxy_pass` does not transform it), the wire-size check applies to the small compressed form, and the first component to inflate the body is the Go application — i.e. `httputil.Decompression`.

## What `httputil.Decompression` does about it

- Wraps the inflated stream in `limitedReader` bounded by `MaxDecompressionSize` (default 16 MiB) — the limit is enforced on **decompressed** bytes, which is the resource actually under attack (memory/stream cost), not wire bytes.
- On trip, reads return `decompression.size_exceeded` (`Rejection` family, not retryable, client's fault) and the underlying decompressor is closed (`decompression.go`, `limitedReader.Read`). Handlers translate the error into whatever response policy they want (see the `Decompression`→`MaxBodySize` chain test, which aborts the read mid-stream).
- Note the config semantics precisely: `MaxDecompressionSize: 0` is documented as "no limit" on the field, but the constructor maps `0` to the 16 MiB default; a negative value is a config `Rejection` (`decompression.max_size_negative`). There is currently no way to configure "truly unlimited" — a deliberate safe-default, and the reason an operator cannot accidentally disable the protection.

## Comparison table

| Aspect                           | nginx (request body)                          | `httputil.Decompression`                           |
| -------------------------------- | --------------------------------------------- | -------------------------------------------------- |
| Decompresses request bodies?     | No (gunzip module is response-side only)      | Yes (gzip, deflate)                                |
| Size limit enforced on           | Wire bytes (`client_max_body_size`, 413)      | Decompressed bytes (`MaxDecompressionSize`)        |
| Bomb (small wire, huge inflated) | Passes the proxy sized check; backend exposed | Tripped at the inflated-byte limit; read aborted   |
| Error surfaced                   | 413 to client before proxying                 | Classified error to the handler (`Rejection`)      |
| Default                          | `client_max_body_size 1m` (wire)              | 16 MiB inflated (0 maps to default; no off-switch) |
| Header decompression awareness   | Yes (HPACK limits applied post-decompression) | N/A (Go net/http owns HTTP/2 header limits)        |

## Practical guidance for the README/deployment docs

- `Decompression` and `client_max_body_size` are **complementary, not redundant**: set the wire limit at the proxy to cap ingress bandwidth, and keep `MaxDecompressionSize` at the app to cap inflated bytes. Raising the proxy limit for large uploads must not be mirrored blindly into `MaxDecompressionSize` — the two numbers bound different resources.
- Deployments that terminate compression at the proxy with something like `proxy_set_header Content-Encoding ""` after gunzipping in a Lua/JS layer are the exception, not nginx stock behavior — stock nginx does not do this.
- If the app never configures `Decompression`, the Go service is the component exposed: any gzip-parsing code path (including `net/http` transparent handling only for **response** bodies — Go's client auto-decompresses, servers do not) needs its own limit.
