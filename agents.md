# go-qbittorrent — Agent Reference

Complete technical reference for AI agents working in this repository.
Keep this file up to date whenever architectural decisions change.

---

## 1. Project Identity

| Item | Value |
|---|---|
| Module | `github.com/autogrr/go-qbittorrent` |
| Go version | 1.22+ |
| Purpose | Thread-safe Go client for the qBittorrent WebUI API v2 |
| License | MIT |

---

## 2. Repository Layout

```
.
├── .github/workflows/test.yml   # CI pipeline (Go 1.22, 1.23, stable)
├── cmd/gen/main.go              # Code generator (go/ast → merge_gen.go + deepcopy_gen.go)
├── examples/basic/main.go       # Minimal usage example
│
├── client.go                    # Config, NewClient, HTTP transport, retry, auth, compression
├── types.go                     # All API value types (ALL fields are pointer *T)
├── options.go                   # Query/body option structs with Encode() methods
├── errors.go                    # Sentinel errors + APIError
├── ptr.go                       # Ptr[T] / Deref[T] generics
├── json.go                      # bytedance/sonic JSON decoder wrapper
├── log.go                       # Leveled logger (no external dep)
│
├── app.go                       # /api/v2/app/* methods
├── torrents.go                  # /api/v2/torrents/* methods
├── transfer.go                  # /api/v2/transfer/* methods
├── sync.go                      # /api/v2/sync/* methods + SyncState
├── syncapi.go                   # SyncAPI poller for long-polling maindata
├── rss_search.go                # /api/v2/rss/* + /api/v2/search/* methods
│
├── merge_gen.go                 # GENERATED — do not edit by hand
├── deepcopy_gen.go              # GENERATED — do not edit by hand
│
├── *_test.go                    # All tests are offline (no live qBittorrent)
└── testserver_test.go           # Shared fakeServer + JSON/form helpers
```

---

## 3. Core Architecture Decisions

### 3.1 Pointer fields everywhere

Every field in every API response struct in `types.go` is a pointer (`*T`).
This lets callers distinguish "field absent in JSON" (nil) from "field present and zero" (`Ptr(0)`).

```go
// Correct — pointer field
ETA *int64 `json:"eta"`

// Wrong — never add plain value fields to API structs in types.go
ETA int64 `json:"eta"`
```

Use the generic helpers from `ptr.go`:

```go
Ptr(42)        // *int  pointing to 42
Deref(t.ETA)   // int64, 0 if t.ETA == nil
```

### 3.2 Buffer pool

All HTTP response and multipart request bodies flow through a `sync.Pool` of pre-allocated
`*bytes.Buffer` (64 KiB each). Never allocate your own buffer for HTTP I/O — use the pool:

```go
buf := getBuf()
defer putBuf(buf)
```

### 3.3 Compression auto-tune

`autoTuneCompression` is called on every successful login (unless `Config.DisableCompression`
is already set). It dials the host directly and measures TCP round-trip time.
If RTT < 5 ms (LAN/localhost), gzip is disabled on the transport — qBittorrent compresses
synchronously at gzip level 6 on its event-loop thread, wasting CPU on both ends.

Set `Config.DisableCompression = true` to skip auto-tune and always skip compression.

### 3.4 HTTP/1.1 only

HTTP/2 is explicitly disabled. qBittorrent's WebUI uses a single-threaded event loop;
multiplexing over HTTP/2 introduces head-of-line blocking and race conditions worse than
pipelining over a single HTTP/1.1 connection.

### 3.5 JSON decoder

`bytedance/sonic` is used rather than `encoding/json` for significantly faster unmarshalling
of large torrent arrays. The wrapper is in `json.go`. Use the local `decodeJSON` / `unmarshal`
wrappers, not `json.Unmarshal` directly.

### 3.6 Retry on 403

The client automatically re-logs in and retries on HTTP 403. The retry count is controlled
by `Config.RetryAttempts` (default 3). The delay between attempts is `Config.RetryDelay`
(default 500 ms).

---

## 4. Types and Options

### 4.1 Adding a new API type

1. Add the struct to `types.go`. All fields must be pointer types.
2. Run the generator (see §5) to regenerate `merge_gen.go` and `deepcopy_gen.go`.
3. If the new struct should participate in `mergePartial`, add its name to `mergeTargets`
   in `cmd/gen/main.go`.

### 4.2 Adding new filter/option fields

`TorrentFilterOptions`, `TorrentAddOptions`, and similar structs live in `options.go`.
Each has an `Encode() url.Values` method. When adding a field:

- Non-pointer bools: emit as `"true"` only when the field is `true` (zero value is omitted).
- `*bool`: emit when non-nil (`"true"` or `"false"`).
- Strings/ints: emit only when non-zero.

```go
// Example pattern
if o.IncludeFiles {
    v.Set("includeFiles", "true")
}
if o.IsPrivate != nil {
    v.Set("private", boolStr(*o.IsPrivate))
}
```

---

## 5. Code Generator

### 5.1 What it generates

The generator at `cmd/gen/main.go` parses `types.go` using `go/ast` and emits two files:

| File | Contents |
|---|---|
| `merge_gen.go` | `mergePartialTorrent` and `mergePartialServerState` — presence-aware field merging for the sync/maindata pipeline |
| `deepcopy_gen.go` | `deepCopy<Type>` functions for every struct that has at least one pointer field |

**`mergePartial<T>`** semantics:
- `*T` pointer fields → copied only when `src` field is non-nil
- plain `bool` fields → always overwritten (false is a meaningful value)
- value fields (`int`, `string`, `float64`) → copied only when `src` value is non-zero

**`deepCopy<T>`** semantics:
- Shallow-copies the struct, then allocates a new pointee for each `*T` field
- Returns nil when input is nil
- Prevents callers from mutating internal `SyncState` through returned pointers

### 5.2 Running the generator

The `//go:generate` directive lives in `sync.go`:

```go
//go:generate go run ./cmd/gen
```

Run from the **module root**:

```bash
go generate ./...
```

Or run the generator directly (also from the module root, because the generator opens
`types.go` by a relative path):

```bash
go run ./cmd/gen
```

Both commands write `merge_gen.go` and `deepcopy_gen.go` in-place and print:

```
wrote merge_gen.go
wrote deepcopy_gen.go
```

**Always re-run the generator after any change to `types.go`** that adds, removes, or
renames a field in `Torrent`, `ServerState`, or any other struct with pointer fields.

### 5.3 Maintaining the generator

The generator (`cmd/gen/main.go`) does two AST passes over `types.go`:

1. **Pass 1** — collects named-type aliases (e.g. `type TorrentState string`) so that
   fields using those aliases can be classified by their underlying kind.
2. **Pass 2** — walks all struct declarations, classifies each field as pointer / bool /
   string / numeric / other, and assembles `[]fieldDesc`.

**`mergeTargets`** — the slice at the top of `cmd/gen/main.go` controls which structs
get a `mergePartial` function. Add a struct name there before running `go generate` if
the sync pipeline needs to merge it.

**Adding a new field type**: if you add a field that uses a type whose underlying kind
is not `bool`, `string`, or a numeric type (e.g. a named `int` type), verify that
`resolveUnderlying` in the generator correctly resolves it by checking the emitted code
in `merge_gen.go`. If it falls through to `AlwaysWrite: true` when it should be zero-skipped,
add or adjust the alias in `types.go` (e.g. `type MyInt = int`).

**Output format**: the generator passes all output through `go/format.Source` before
writing. If the generator crashes with a formatting error, the template or a field
descriptor is producing invalid Go syntax — check `cmd/gen/main.go` templates.

---

## 6. Testing

### 6.1 Golden rule: all tests must be offline

**Never write a test that requires a live qBittorrent instance.**
Every test must work without any external process running.
Use the `fakeServer` helper (see §6.2) to simulate the qBittorrent WebUI.

This is enforced in CI — there is no qBittorrent instance in the GitHub Actions environment.

### 6.2 fakeServer

`testserver_test.go` provides a shared `fakeServer` that wraps `httptest.NewServer`.
It ships with a default login handler that always returns `Ok.` (mimicking qBittorrent's
successful login response).

```go
func TestMyFeature(t *testing.T) {
    fs := newFakeServer(t)  // automatically cleaned up when test ends

    fs.handle("GET", "/api/v2/some/endpoint", func(w http.ResponseWriter, r *http.Request) {
        writeJSON(w, MyResponseType{...})
    })

    c := fs.loggedInClient(t)  // creates + logs in a Client pointing at the fake server
    result, err := c.MyMethod(testCtx(t), ...)
    // assert ...
}
```

`fakeServer` helpers available in tests:

| Helper | Purpose |
|---|---|
| `newFakeServer(t)` | Create server; registers `/api/v2/auth/login` success handler |
| `fs.handle(method, path, handler)` | Register or replace a route |
| `fs.client(t)` | Create an unauthenticated `*Client` |
| `fs.loggedInClient(t)` | Create a `*Client` and call `Login` |
| `fs.callCount(path)` | How many times a path was hit (for retry/count assertions) |
| `writeJSON(w, v)` | Write `v` as JSON with 200 |
| `writeString(w, s)` | Write plain text response |
| `formValue(r, key)` | Read a POST form field |
| `hasFormValue(r, key)` | Check presence of a POST form field |
| `queryValue(r, key)` | Read a URL query parameter |
| `decodeJSON(r, dst)` | Decode request body JSON |
| `makeHashes(hh...)` | Join hashes with `\|` (qBittorrent multi-hash format) |
| `encodePrefs(map)` | JSON-encode preferences for the prefs endpoint |
| `testCtx(t)` | Context with 10s deadline, cancelled on test cleanup |

### 6.3 Test file conventions

- One `_test.go` file per source file: `torrents_test.go` ↔ `torrents.go`.
- All tests are in `package qbittorrent` (white-box tests, same package).
- Use `github.com/stretchr/testify/assert` for assertions.
- Use `testCtx(t)` for every context passed to client methods; never use `context.Background()` directly in tests.
- Use `Ptr(value)` for pointer literals in test data.

### 6.4 Testing options encoding

`TorrentFilterOptions` and other option structs encode to `url.Values` independently of
HTTP. Test the encoding without any server:

```go
func TestMyOption_Encode(t *testing.T) {
    v := TorrentFilterOptions{IncludeFiles: true}.Encode()
    assert.Equal(t, "true", v.Get("includeFiles"))
    assert.Empty(t, v.Get("private"))  // nil IsPrivate must not appear
}
```

### 6.5 Running tests

```bash
# Run all tests with race detector
go test -race -count=1 ./...

# Run with timeout (matches CI)
go test -race -count=1 -timeout=2m ./...

# Run a single test
go test -race -run TestGetTorrents ./...

# Run benchmarks
go test -bench=. -benchmem ./...
```

### 6.6 Benchmarks

`bench_test.go` contains micro-benchmarks for the hot paths:

- `BenchmarkMergePartialTorrent` — merge cost per torrent
- `BenchmarkDeepCopyTorrent` — deep copy cost per torrent
- `BenchmarkGetTorrents_200k` — full map snapshot at 200 k entries
- `BenchmarkGetTorrentSlice_200k` — sorted slice at 200 k entries
- `BenchmarkGetTorrents_Parallel` — concurrent map access

Run before and after changes that touch `merge_gen.go`, `deepcopy_gen.go`, or `sync.go`
to catch regressions.

---

## 7. Error Handling

All sentinel errors are in `errors.go`. Every HTTP error path returns `*APIError`, which
wraps a sentinel and includes the HTTP status code and body snippet:

```go
errors.Is(err, ErrNotFound)     // check sentinel
var apiErr *APIError
errors.As(err, &apiErr)         // get status code + body
fmt.Println(apiErr.StatusCode)
```

Sentinel list:

| Sentinel | Trigger |
|---|---|
| `ErrUnauthorized` | HTTP 403 |
| `ErrNotFound` | HTTP 404 |
| `ErrConflict` | HTTP 409 |
| `ErrBadRequest` | HTTP 400 |
| `ErrInvalidTorrent` | HTTP 415 |
| `ErrLoginFailed` | Login response body is not `Ok.` |
| `ErrIPBanned` | Login response body is `Banned.` |
| `ErrAlreadyExists` | Add-torrent response body is `Exists` |
| `ErrBadResponse` | Any other unexpected server response |

---

## 8. CI Pipeline

File: `.github/workflows/test.yml`

- Triggers: push and pull_request to `main` / `master`
- Matrix: Go `1.22`, `1.23`, `stable` × `ubuntu-latest`
- Steps: checkout → setup-go (with module cache) → `go mod download` → `go mod verify` → `go vet ./...` → `go test -race -count=1 -timeout=2m ./...`

All tests must pass on every matrix cell before merging.

---

## 9. Dependencies

| Package | Purpose |
|---|---|
| `github.com/bytedance/sonic` | Fast JSON marshal/unmarshal (SIMD) |
| `golang.org/x/net` | `cookiejar/publicsuffix` for correct cookie scoping |
| `golang.org/x/sync` | (available; not currently used in hot paths) |
| `github.com/stretchr/testify` | Test assertions (`assert`, `require`) |

Do not add heavy dependencies. The library is intended to be a lightweight client — no
logging frameworks, no config libraries, no HTTP middleware packages.

---

## 10. qBittorrent API Version Notes

| Feature | Min qBittorrent version | Implementation |
|---|---|---|
| `includeTrackers` param on `torrents/info` | 5.1 | `TorrentFilterOptions.IncludeTrackers`; populates `Torrent.Trackers` |
| `includeFiles` param on `torrents/info` | 5.2 | `TorrentFilterOptions.IncludeFiles`; populates `Torrent.Files` |
| `private` filter param on `torrents/info` | 5.1 | `TorrentFilterOptions.IsPrivate` (`*bool`; nil = no filter) |
| `TrackerEndpoint` nested in tracker | 5.x | `TorrentTracker.Endpoints []TrackerEndpoint` |
| `tracker.updating` / `next_announce` / `min_announce` | 5.x | Fields on `TorrentTracker` |

When qBittorrent adds response fields that should be included in maindata merge operations,
add them to `types.go` and re-run `go generate ./...`. If the new field belongs to
`Torrent` or `ServerState`, the generator will automatically include it in the
`mergePartial` function.

---

## 11. Common Workflows

### Add a new API endpoint

1. Identify the endpoint group (torrents, app, transfer, rss, etc.).
2. Add the method to the appropriate `.go` file (e.g. `torrents.go`).
3. Add a response type to `types.go` if needed (all pointer fields).
4. Run `go generate ./...` if `types.go` changed.
5. Add a test in the corresponding `_test.go` using `newFakeServer` — no live server.
6. Run `go test -race -count=1 ./...` to confirm all tests pass.

### Add a field to an existing response type

1. Add the field to the struct in `types.go` — must be a pointer type.
2. Run `go generate ./...` to update `merge_gen.go` and `deepcopy_gen.go`.
3. Verify the generated code looks correct (especially for new named types).
4. Add or update tests.

### Add a new query parameter to TorrentFilterOptions

1. Add the field to `TorrentFilterOptions` in `options.go` with a comment citing the
   minimum qBittorrent version.
2. Add the encoding logic to `Encode()` following the existing patterns.
3. Add unit tests in `options_test.go` (no fakeServer needed — just call `.Encode()`).
4. If the parameter affects the response shape (e.g. embeds nested arrays), add the
   corresponding fields to the `Torrent` struct and re-run `go generate ./...`.

### Update or debug the code generator

1. Edit `cmd/gen/main.go`.
2. Run `go run ./cmd/gen` from the module root.
3. Inspect `merge_gen.go` and `deepcopy_gen.go` for correctness.
4. Run `go build ./...` to confirm the generated code compiles.
5. Run `go test -race -count=1 ./...` to confirm nothing broke.
