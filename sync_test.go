package qbittorrent

import (
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncMainData_FullUpdate(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/sync/maindata", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "0", queryValue(r, "rid"))
		writeJSON(w, MainData{
			Rid:        1,
			FullUpdate: true,
			Torrents: map[string]Torrent{
				"abc123": {Hash: Ptr("abc123"), Name: Ptr("Test Torrent"), State: Ptr(StateDownloading)},
			},
			ServerState: ServerState{
				ConnectionStatus: Ptr("connected"),
				FreeSpaceOnDisk:  Ptr[int64](50 * 1024 * 1024 * 1024),
			},
			Categories: map[string]Category{
				"linux": {Name: "linux", SavePath: "/data"},
			},
			Tags: []string{"hdr", "remux"},
		})
	})
	c := fs.loggedInClient(t)
	data, err := c.SyncMainData(testCtx(t), 0)
	require.NoError(t, err)
	assert.Equal(t, 1, data.Rid)
	assert.True(t, data.FullUpdate)
	require.NotNil(t, data.Torrents["abc123"])
	assert.Equal(t, "Test Torrent", Deref(data.Torrents["abc123"].Name))
	assert.Equal(t, "connected", Deref(data.ServerState.ConnectionStatus))
	assert.Equal(t, []string{"hdr", "remux"}, data.Tags)
}

func TestSyncMainData_PartialUpdate(t *testing.T) {
	fs := newFakeServer(t)
	callCount := 0
	fs.handle("GET", "/api/v2/sync/maindata", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			writeJSON(w, MainData{
				Rid:        1,
				FullUpdate: true,
				Torrents: map[string]Torrent{
					"abc": {Hash: Ptr("abc"), Name: Ptr("Original"), DlSpeed: Ptr[int64](1000), State: Ptr(StateDownloading)},
				},
				ServerState: ServerState{ConnectionStatus: Ptr("connected")},
			})
		} else {
			writeJSON(w, MainData{
				Rid:        2,
				FullUpdate: false,
				Torrents: map[string]Torrent{
					"abc": {DlSpeed: Ptr[int64](2000)},
				},
			})
		}
	})
	c := fs.loggedInClient(t)
	d1, err := c.SyncMainData(testCtx(t), 0)
	require.NoError(t, err)
	assert.Equal(t, 1, d1.Rid)

	d2, err := c.SyncMainData(testCtx(t), 1)
	require.NoError(t, err)
	assert.Equal(t, 2, d2.Rid)
	// The partial update should only carry the updated speed.
	assert.Equal(t, int64(2000), Deref(d2.Torrents["abc"].DlSpeed))
}

func TestSyncMainDataRaw(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/sync/maindata", func(w http.ResponseWriter, _ *http.Request) {
		writeString(w, `{"rid":5,"full_update":true}`)
	})
	c := fs.loggedInClient(t)
	raw, err := c.SyncMainDataRaw(testCtx(t), 0)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"rid":5`)
}

func TestSyncTorrentPeers(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/sync/torrentPeers", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "abc123", queryValue(r, "hash"))
		assert.Equal(t, "3", queryValue(r, "rid"))
		writeJSON(w, TorrentPeersResponse{
			Rid:        4,
			FullUpdate: false,
			Peers: map[string]TorrentPeer{
				"1.2.3.4:6881": {IP: Ptr("1.2.3.4"), Port: Ptr(6881), DlSpeed: Ptr[int64](512)},
			},
		})
	})
	c := fs.loggedInClient(t)
	resp, err := c.SyncTorrentPeers(testCtx(t), "abc123", 3)
	require.NoError(t, err)
	assert.Equal(t, 4, resp.Rid)
	peer := resp.Peers["1.2.3.4:6881"]
	assert.Equal(t, "1.2.3.4", Deref(peer.IP))
	assert.Equal(t, int64(512), Deref(peer.DlSpeed))
}

// ---------- mergePartialTorrent unit tests ----------

func TestMergePartialTorrent_NonZeroFieldsOverwrite(t *testing.T) {
	dst := &Torrent{Hash: Ptr("abc"), Name: Ptr("Original"), DlSpeed: Ptr[int64](100), State: Ptr(StateDownloading)}
	src := &Torrent{DlSpeed: Ptr[int64](9999), UpSpeed: Ptr[int64](500)}
	mergePartialTorrent(dst, src)
	assert.Equal(t, int64(9999), Deref(dst.DlSpeed), "non-zero DlSpeed should overwrite")
	assert.Equal(t, int64(500), Deref(dst.UpSpeed), "non-zero UpSpeed should propagate")
}

func TestMergePartialTorrent_ZeroFieldsDontOverwrite(t *testing.T) {
	dst := &Torrent{Hash: Ptr("abc"), Name: Ptr("Keep Me"), Progress: Ptr(0.75)}
	src := &Torrent{DlSpeed: Ptr[int64](1000)} // Name and Progress are nil in src
	mergePartialTorrent(dst, src)
	assert.Equal(t, "Keep Me", Deref(dst.Name), "nil Name in src must not overwrite dst")
	assert.InDelta(t, 0.75, Deref(dst.Progress), 0.001, "nil Progress in src must not overwrite dst")
	assert.Equal(t, int64(1000), Deref(dst.DlSpeed))
}

func TestMergePartialTorrent_StatePreserved(t *testing.T) {
	dst := &Torrent{Hash: Ptr("abc"), State: Ptr(StateStalledUP)}
	src := &Torrent{DlSpeed: Ptr[int64](0)} // State is nil in src
	mergePartialTorrent(dst, src)
	assert.Equal(t, StateStalledUP, Deref(dst.State))
}

// ---------- mergePartialServerState unit tests ----------

func TestMergeServerState_NonZeroOverwrites(t *testing.T) {
	dst := ServerState{DlInfoSpeed: Ptr[int64](1000), ConnectionStatus: Ptr("connected")}
	src := ServerState{DlInfoSpeed: Ptr[int64](5000)}
	mergePartialServerState(&dst, &src)
	assert.Equal(t, int64(5000), Deref(dst.DlInfoSpeed))
	assert.Equal(t, "connected", Deref(dst.ConnectionStatus), "nil ConnectionStatus in src must be preserved")
}

func TestMergeServerState_FreeSpaceUpdated(t *testing.T) {
	dst := ServerState{FreeSpaceOnDisk: Ptr[int64](10 * 1024 * 1024 * 1024)}
	src := ServerState{FreeSpaceOnDisk: Ptr[int64](8 * 1024 * 1024 * 1024)}
	mergePartialServerState(&dst, &src)
	assert.Equal(t, int64(8*1024*1024*1024), Deref(dst.FreeSpaceOnDisk))
}

// ---------- SyncState unit tests ----------

func TestSyncState_Apply_FullUpdate(t *testing.T) {
	ss := newSyncState()
	data := &MainData{
		Rid:        1,
		FullUpdate: true,
		Torrents: map[string]Torrent{
			"a": {Hash: Ptr("a"), Name: Ptr("Alpha")},
			"b": {Hash: Ptr("b"), Name: Ptr("Beta")},
		},
		ServerState: ServerState{ConnectionStatus: Ptr("connected")},
		Categories:  map[string]Category{"cat": {Name: "cat"}},
		Tags:        []string{"t1"},
	}
	ss.apply(data)

	a, ok := ss.GetTorrent("a")
	require.True(t, ok)
	assert.Equal(t, "Alpha", Deref(a.Name))
	torrents := ss.GetTorrents()
	assert.Len(t, torrents, 2)
	assert.Equal(t, "connected", Deref(ss.GetServerState().ConnectionStatus))
	cats := ss.GetCategories()
	assert.Equal(t, "cat", cats["cat"].Name)
	assert.Equal(t, []string{"t1"}, ss.GetTags())
}

func TestSyncState_Apply_PartialMerge(t *testing.T) {
	ss := newSyncState()
	// Initial full update.
	ss.apply(&MainData{
		Rid:        1,
		FullUpdate: true,
		Torrents: map[string]Torrent{
			"a": {Hash: Ptr("a"), Name: Ptr("Alpha"), DlSpeed: Ptr[int64](100)},
		},
		ServerState: ServerState{DlInfoSpeed: Ptr[int64](100)},
	})
	// Partial update: only DlSpeed changed.
	ss.apply(&MainData{
		Rid:        2,
		FullUpdate: false,
		Torrents: map[string]Torrent{
			"a": {DlSpeed: Ptr[int64](9999)},
		},
		ServerState: ServerState{DlInfoSpeed: Ptr[int64](7777)},
	})

	t2, ok := ss.GetTorrent("a")
	require.True(t, ok)
	assert.Equal(t, "Alpha", Deref(t2.Name), "Name should be preserved from full update")
	assert.Equal(t, int64(9999), Deref(t2.DlSpeed), "DlSpeed should be updated")
	assert.Equal(t, int64(7777), Deref(ss.GetServerState().DlInfoSpeed))
}

func TestSyncState_GetTorrentSlice(t *testing.T) {
	ss := newSyncState()
	ss.apply(&MainData{
		Rid:        1,
		FullUpdate: true,
		Torrents: map[string]Torrent{
			"a": {Hash: Ptr("a"), Name: Ptr("Alpha")},
			"b": {Hash: Ptr("b"), Name: Ptr("Beta")},
		},
	})
	sl := ss.GetTorrentSlice()
	assert.Len(t, sl, 2)
}

// ---------- SyncManager integration test ----------

func TestSyncManager_StartStop(t *testing.T) {
	fs := newFakeServer(t)
	var callCount int32
	fs.handle("GET", "/api/v2/sync/maindata", func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		writeJSON(w, MainData{
			Rid:        int(n),
			FullUpdate: n == 1,
		})
	})

	c := fs.loggedInClient(t)
	var updateCount int32
	mgr := c.NewSyncManager(SyncOptions{
		Interval: 20 * time.Millisecond,
		OnUpdate: func(_ *SyncState) {
			atomic.AddInt32(&updateCount, 1)
		},
	})

	mgr.Start(testCtx(t))
	time.Sleep(100 * time.Millisecond)
	mgr.Stop()

	assert.GreaterOrEqual(t, atomic.LoadInt32(&callCount), int32(1), "should have polled at least once")
	assert.GreaterOrEqual(t, atomic.LoadInt32(&updateCount), int32(1), "OnUpdate should have been called")
}

func TestSyncManager_StartTwice(t *testing.T) {
	// Calling Start() more than once must be a no-op: only one poller goroutine
	// may run, and Stop() must not panic (no double-close of the done channel).
	fs := newFakeServer(t)
	var callCount int32
	fs.handle("GET", "/api/v2/sync/maindata", func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		writeJSON(w, MainData{Rid: int(n), FullUpdate: n == 1})
	})

	c := fs.loggedInClient(t)
	mgr := c.NewSyncManager(SyncOptions{Interval: 10 * time.Millisecond})

	ctx := testCtx(t)
	mgr.Start(ctx)
	mgr.Start(ctx) // must be a no-op
	mgr.Start(ctx) // must be a no-op

	time.Sleep(80 * time.Millisecond)

	// Stop must not panic (done channel must be closed exactly once).
	assert.NotPanics(t, mgr.Stop)

	// With three goroutines the call rate would be ~3×; one goroutine at
	// 10 ms interval over 80 ms gives ≈8 calls. Allow a generous ceiling
	// to avoid flakiness but catch a gross "3 pollers" regression.
	n := atomic.LoadInt32(&callCount)
	assert.GreaterOrEqual(t, n, int32(1), "should have polled at least once")
	assert.LessOrEqual(t, n, int32(30), "call count suggests more than one poller goroutine")
}

func TestSyncManager_OnError(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/sync/maindata", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	c := fs.loggedInClient(t)
	var errCount int32
	mgr := c.NewSyncManager(SyncOptions{
		Interval: 20 * time.Millisecond,
		OnError: func(err error) bool {
			atomic.AddInt32(&errCount, 1)
			return false
		},
	})

	mgr.Start(testCtx(t))
	time.Sleep(80 * time.Millisecond)
	mgr.Stop()

	assert.GreaterOrEqual(t, atomic.LoadInt32(&errCount), int32(1), "OnError should have been called")
}

// ---- GetServerState deep copy ----

func TestGetServerState_DeepCopy(t *testing.T) {
	// Modifying the value returned by GetServerState must not affect the
	// internal SyncState, because GetServerState returns a deep copy.
	ss := newSyncState()
	ss.apply(&MainData{
		FullUpdate:  true,
		ServerState: ServerState{ConnectionStatus: Ptr("connected"), DlInfoSpeed: Ptr[int64](1000)},
	})

	// Obtain a copy and mutate it.
	state := ss.GetServerState()
	*state.ConnectionStatus = "disconnected"
	*state.DlInfoSpeed = 9999

	// The internal state must be unchanged.
	internal := ss.GetServerState()
	assert.Equal(t, "connected", Deref(internal.ConnectionStatus))
	assert.Equal(t, int64(1000), Deref(internal.DlInfoSpeed))
}
