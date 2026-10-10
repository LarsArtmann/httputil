# branching-flow baseline — 2026-10-10, branching-flow 0.6.4 (60a9108)

Regenerate: `branching-flow all . --format markdown`. Every row is line-verified accepted design; per-class verdicts live in docs/architecture-reference.md ("Branching-flow residual baseline"). Anything NEW vs these sections/counts is a finding.

```markdown
# Duplicate Type Analysis

6 rows

| Group | Type            | Kind   | File                           | Line | Package      | Verdict        | Shared         | Action                                            |
|------|----------------|-------|-------------------------------|-----|-------------|---------------|---------------|--------------------------------------------------|
| 1     | clientIPKey     | struct | context.go                     | 8    | httputil     | false_positive | (empty struct) | Empty structs used as marker types or brand types |
| 1     | csrfKey         | struct | csrf.go                        | 457  | httputil     | false_positive | (empty struct) | Empty structs used as marker types or brand types |
| 1     | languageKey     | struct | language.go                    | 437  | httputil     | false_positive | (empty struct) | Empty structs used as marker types or brand types |
| 1     | nonceKey        | struct | nonce.go                       | 132  | httputil     | false_positive | (empty struct) | Empty structs used as marker types or brand types |
| 1     | requestIDKey    | struct | requestid.go                   | 8    | httputil     | false_positive | (empty struct) | Empty structs used as marker types or brand types |
| 1     | serverTimingKey | struct | server_timing/server_timing.go | 219  | servertiming | false_positive | (empty struct) | Empty structs used as marker types or brand types |

# Phantom Type Analysis

21 rows

| Name                                | Rule      | Type   | Underlying | Kind     | Conf.  | Origin     | Slot   | Callee  | File                           | Line | Suggestion                                      |
|------------------------------------|----------|-------|-----------|---------|-------|-----------|-------|--------|-------------------------------|-----|------------------------------------------------|
| method, path                        | transpose | string | string     | fn param | high   |            |        |         | httpspec/httpspec.go           | 139  | create type Method, Path                        |
| method, path                        | transpose | string | string     | fn param | high   |            |        |         | httpspec/httpspec.go           | 154  | create type Method, Path                        |
| method, path, header, expectedValue | transpose | string | string     | fn param | medium |            |        |         | httpspec/httpspec.go           | 168  | create type Method, Path, Header, ExpectedValue |
| method, path, header                | transpose | string | string     | fn param | medium |            |        |         | httpspec/httpspec.go           | 185  | create type Method, Path, Header                |
| method, path, substring             | transpose | string | string     | fn param | medium |            |        |         | httpspec/httpspec.go           | 199  | create type Method, Path, Substring             |
| method, path, contentType           | transpose | string | string     | fn param | medium |            |        |         | httpspec/httpspec.go           | 273  | create type Method, Path, ContentType           |
| method, path, contentType           | transpose | string | string     | fn param | medium |            |        |         | httpspec/httpspec.go           | 298  | create type Method, Path, ContentType           |
| method, path, field                 | transpose | string | string     | fn param | medium |            |        |         | httpspec/httpspec.go           | 367  | create type Method, Path, Field                 |
| method, path, etag                  | transpose | string | string     | fn param | medium |            |        |         | httpspec/httpspec.go           | 398  | create type Method, Path, Etag                  |
| certFile, keyFile                   | transpose | string | string     | fn param | high   |            |        |         | server.go                      | 251  | create type CertFile, KeyFile                   |
| name, desc                          | transpose | string | string     | fn param | high   |            |        |         | server_timing/server_timing.go | 69   | create type Name, Desc                          |
| name, desc                          | transpose | string | string     | fn param | high   |            |        |         | server_timing/server_timing.go | 105  | create type Name, Desc                          |
| name, desc                          | transpose | string | string     | fn param | high   |            |        |         | server_timing/server_timing.go | 237  | create type Name, Desc                          |
| name -> target                      | collision | string | string     | fn param | medium | name       | target | indexOf | compression_negotiator.go      | 181  | create type Target                              |
| name -> input                       | collision | string | string     | fn param | medium | name       | input  | trim    | compression_qvalue.go          | 11   | create type InputString                         |
| tag -> input                        | collision | string | string     | fn param | medium | tag        | input  | trim    | language.go                    | 221  | create type InputString                         |
| tag -> input                        | collision | string | string     | fn param | medium | tag        | input  | trim    | language.go                    | 250  | create type InputString                         |
| header -> input                     | collision | string | string     | fn param | medium | header     | input  | trim    | language.go                    | 307  | create type InputString                         |
| tag -> input                        | collision | string | string     | fn param | medium | tag        | input  | trim    | language.go                    | 347  | create type InputString                         |
| tag -> input                        | collision | string | string     | fn param | medium | tag        | input  | trim    | language.go                    | 427  | create type InputString                         |
| DefaultTag -> input                 | collision | string | string     | fn param | medium | DefaultTag | input  | trim    | language.go                    | 459  | create type InputString                         |

# Panic Conditions Analysis

20 rows

| Type               | Severity | Function | File                      | Line | Description                                           | Suggestion                                                                                                 |
|-------------------|---------|---------|--------------------------|-----|------------------------------------------------------|-----------------------------------------------------------------------------------------------------------|
| Index Out of Range | medium   | httputil | compression_negotiator.go | 62   | Slice index 'j' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | compression_negotiator.go | 63   | Slice index 'j' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | compression_negotiator.go | 63   | Slice index 'j' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | compression_negotiator.go | 64   | Slice index 'j' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | compression_negotiator.go | 64   | Slice index 'j' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | compression_qvalue.go     | 135  | Slice index 'start' access may panic if out of bounds | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | language.go               | 254  | Slice index 'i' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | language.go               | 260  | Slice index 'idx' access may panic if out of bounds   | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | language.go               | 328  | Slice index 'j' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | language.go               | 329  | Slice index 'j' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | language.go               | 329  | Slice index 'j' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | ratelimit_keyed.go        | 422  | Slice index 'i' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | ratelimit_keyed.go        | 422  | Slice index 'j' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | ratelimit_keyed.go        | 424  | Slice index 'i' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | ratelimit_keyed.go        | 424  | Slice index 'j' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | ratelimit_keyed.go        | 424  | Slice index 'j' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | ratelimit_keyed.go        | 424  | Slice index 'i' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | ratelimit_keyed.go        | 425  | Slice index 'i' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Index Out of Range | medium   | httputil | ratelimit_keyed.go        | 426  | Slice index 'j' access may panic if out of bounds     | Add bounds checking before access: if i < len(arr) { ... } or use slices.Index(slice, val) for safe lookup |
| Explicit Panic     | high     | httpspec | httpspec/httpspec.go      | 349  | Explicit panic() call detected                        | Consider returning an error instead of panicking                                                           |

# Strong ID Analysis

0 rows

| Name | Current Type | Kind | File | Line | Suggested Type |
|-----|-------------|-----|-----|-----|---------------|

# Boolean Blindness Analysis

0 rows

| Struct | Bool Fields | Before | After | File | Line | Suggestion |
|-------|------------|-------|------|-----|-----|-----------|

# Composition Analysis: Anti-Patterns

0 rows

| Struct | Pattern | Confidence | Category | File | Line | Suggestion |
|-------|--------|-----------|---------|-----|-----|-----------|

# Composition Analysis: Mixins

0 rows

| Struct | Pattern | Confidence | Category | File | Line | Suggestion |
|-------|--------|-----------|---------|-----|-----|-----------|

# Split-Brain Interface Analysis

0 rows

| Concretion | Generic | Substitutions | File | Line |
|-----------|--------|--------------|-----|-----|

# Context Propagation Analysis

0 rows

| Call | Function | File | Line |
|-----|---------|-----|-----|

# Naked Return Guard Analysis

0 rows

| Function | Length | File | Line |
|---------|-------|-----|-----|

# Flag Parameter Analysis

1 rows

| Function | File                 | Line | Param    | Index | Bools | Severity | Reason                                                                                              |
|---------|---------------------|-----|---------|------|------|---------|----------------------------------------------------------------------------------------------------|
| runSpecs | httpspec/httpspec.go | 235  | parallel | 2     | 1     | low      | bool parameter is not last, positional confusion for callers. Move to end or use an options struct. |

# Interface Completion Analysis

0 rows

| Interface | Implementer | File | Line | Methods | Severity |
|----------|------------|-----|-----|--------|---------|

# samber/do Anti-Pattern Analysis

0 rows

| Call | Pattern | Message | Function | File | Line |
|-----|--------|--------|---------|-----|-----|

# samber/ro Anti-Pattern Analysis

0 rows

| Call | Pattern | Message | Function | File | Line |
|-----|--------|--------|---------|-----|-----|

```
