# go-qbittorrent

High-performance Go API bindings for the [qBittorrent WebUI API](https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)) (v2 / qBit 4.1+).

## Performance design

| Technique | Benefit |
|-----------|---------|
| HTTP/2 with tuned transport (64 KiB R/W buffers, 32 idle conns/host) | Multiplexed requests, lower latency |
| `sync.Pool` for request/response byte buffers | Amortises allocations across repeated calls |
| `json.NewDecoder` streaming on the happy path | Network bytes flow directly into the JSON parser — no intermediate copy |
| Multipart uploads buffered synchronously into pooled buffer | No goroutine, no `io.Pipe`, no closure heap escape |
| Pre-computed login form | `url.Values.Encode()` runs once at construction, not on every re-login |
| `singleflight` in `SyncManager.Sync()` | Coalesces concurrent sync polls into one request |
| `time.NewTimer` + `Stop()` for retry delay | No timer goroutine leak on context cancellation |
| `github.com/bytedance/sonic` JSON | 2–4× faster marshal/unmarshal via SIMD on amd64 |
| Context-first API throughout | No implicit goroutine leaks; full cancellation support |
| Automatic session re-login on 403 | Transparent reconnection without caller changes |

## Full API coverage

- `auth` — Login / Logout
- `app` — Version, build info, preferences, cookies, shutdown
- `log` — Main log, peer log
- `sync` — `maindata`, `torrentPeers` (raw and typed)
- `transfer` — Speed stats, global limits, ban peers
- `torrents` — Full CRUD: list, properties, trackers, files, piece states, add (URL/reader/bytes), pause, resume, delete, recheck, reannounce, priorities, limits, categories, tags, renaming, torrent creator (qBit ≥5.0)
- `rss` — Feeds, folders, rules, matching articles
- `search` — Start/stop/status/results/delete, plugin management
- `SyncManager` — Background polling with dynamic interval, partial-update merging, and thread-safe state reads

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    qbt "github.com/autogrr/go-qbittorrent"
)

func main() {
    client, err := qbt.NewClient(qbt.Config{
        Host:     "http://localhost:8080",
        Username: "admin",
        Password: "adminadmin",
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    if err := client.Login(ctx); err != nil {
        log.Fatal(err)
    }

    torrents, err := client.GetTorrents(ctx, qbt.TorrentFilterOptions{
        Filter: qbt.FilterDownloading,
    })
    if err != nil {
        log.Fatal(err)
    }
    for _, t := range torrents {
        fmt.Printf("%s  %.1f%%  ↓%d KiB/s\n", t.Name, t.Progress*100, t.DlSpeed/1024)
    }
}
```

## SyncManager (live state)

```go
sm := client.NewSyncManager(qbt.SyncOptions{
    Interval:        500 * time.Millisecond,
    DynamicInterval: true,
    OnUpdate: func(state *qbt.SyncState) {
        fmt.Printf("free disk: %d MiB\n", state.GetServerState().FreeSpaceOnDisk/1<<20)
    },
})
sm.Start(ctx)
defer sm.Stop()

// Thread-safe reads at any time:
torrent, ok := sm.State().GetTorrent("abc123...")
```

## Adding a torrent

```go
// From URL / magnet:
err = client.AddTorrentFromURL(ctx, "magnet:?xt=urn:btih:...", qbt.TorrentAddOptions{
    Category:       "movies",
    SavePath:       "/data/movies",
    RatioLimit:     2.0,
})

// From an io.Reader (streaming — no full-file buffering):
f, _ := os.Open("my.torrent")
defer f.Close()
err = client.AddTorrentFromReader(ctx, "my.torrent", f, qbt.TorrentAddOptions{})
```

## Error handling

```go
if errors.Is(err, qbt.ErrNotFound) { ... }
if errors.Is(err, qbt.ErrConflict) { ... }

var apiErr *qbt.APIError
if errors.As(err, &apiErr) {
    fmt.Println(apiErr.StatusCode, apiErr.Body)
}
```

## Installation

```bash
go get github.com/autogrr/go-qbittorrent
```

## Requirements

- Go 1.22+
- qBittorrent 4.1+ (WebUI API v2)

---

## Code generation

The package uses `cmd/gen` to generate two files from `types.go` at build time.
Run it whenever a struct in `types.go` is added or modified:

```bash
go generate ./...
```

This reads both `//go:generate` directives (located in `sync.go`) and
produces:

### `merge_gen.go`

Contains a `mergePartial<Type>` function for each struct listed in
`cmd/gen/main.go:mergeTargets` (currently `Torrent` and `ServerState`).

The generated merge semantics are:

| Field kind | Behaviour |
|---|---|
| `string` | Only written if `src != ""` |
| numeric (`int*`, `uint*`, `float*`) | Only written if `src != 0` |
| `bool` | **Always written** — `false` is a valid partial-update value |
| `*T` (pointer) | **Always written** — `nil` is a valid partial-update value |
| slice / map / other | **Always written** |

This prevents partial sync updates from silently zeroing fields that were
omitted from the server response.

### `deepcopy_gen.go`

Contains a `deepCopy<Type>` function for every struct in `types.go` that has
at least one pointer field.

Each pointer field is independently allocated so callers that receive a value
through `GetTorrent`, `GetTorrents`, `GetTorrentSlice`, or `VisitTorrents`
cannot mutate internal `SyncState` through a shared pointer.

Currently the only pointer field across all types is `Torrent.Private *bool`.

### Adding a new struct or field

1. Edit `types.go` — add the struct or field.
2. If the struct needs partial-merge semantics, add its name to `mergeTargets`
   in `cmd/gen/main.go`.
3. Run `go generate ./...`.
4. The generator re-parses `types.go` via `go/ast`, classifies fields by
   underlying kind (following named-type aliases), and regenerates both files.
   `gofmt`-formatted output is written atomically.
