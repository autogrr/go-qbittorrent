package qbittorrent

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sampleTorrents = []Torrent{
	{
		Hash:     Ptr("aabbccddeeff00112233445566778899aabbccdd"),
		Name:     Ptr("Arch Linux 2024.01.01"),
		State:    Ptr(StateDownloading),
		Size:     Ptr[int64](800 * 1024 * 1024),
		Progress: Ptr(0.42),
		DlSpeed:  Ptr[int64](2 * 1024 * 1024),
		Category: Ptr("linux"),
		Tags:     Ptr("open-source,distro"),
	},
	{
		Hash:     Ptr("1122334455667788990011223344556677889900"),
		Name:     Ptr("Ubuntu 24.04 LTS"),
		State:    Ptr(StateUploading),
		Size:     Ptr[int64](1200 * 1024 * 1024),
		Progress: Ptr(1.0),
		UpSpeed:  Ptr[int64](512 * 1024),
		Category: Ptr("linux"),
	},
}

func TestGetTorrents(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "downloading", queryValue(r, "filter"))
		assert.Equal(t, "linux", queryValue(r, "category"))
		writeJSON(w, []Torrent{sampleTorrents[0]})
	})
	c := fs.loggedInClient(t)
	torrents, err := c.GetTorrents(testCtx(t), TorrentFilterOptions{
		Filter:   FilterDownloading,
		Category: "linux",
	})
	require.NoError(t, err)
	require.Len(t, torrents, 1)
	assert.Equal(t, "Arch Linux 2024.01.01", Deref(torrents[0].Name))
	assert.Equal(t, StateDownloading, Deref(torrents[0].State))
}

func TestGetTorrents_Empty(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/torrents/info", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []Torrent{})
	})
	c := fs.loggedInClient(t)
	torrents, err := c.GetTorrents(testCtx(t), TorrentFilterOptions{})
	require.NoError(t, err)
	assert.Empty(t, torrents)
}

func TestGetTorrentsRaw(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/torrents/info", func(w http.ResponseWriter, _ *http.Request) {
		writeString(w, `[{"hash":"abc","name":"test"}]`)
	})
	c := fs.loggedInClient(t)
	raw, err := c.GetTorrentsRaw(testCtx(t), TorrentFilterOptions{})
	require.NoError(t, err)
	assert.Contains(t, string(raw), "abc")
}

func TestGetTorrentProperties(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/torrents/properties", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, Deref(sampleTorrents[0].Hash), queryValue(r, "hash"))
		writeJSON(w, TorrentProperties{
			Name:       Ptr("Arch Linux 2024.01.01"),
			TotalSize:  Ptr[int64](800 * 1024 * 1024),
			IsPrivate:  Ptr(false),
			ShareRatio: Ptr(1.23),
		})
	})
	c := fs.loggedInClient(t)
	props, err := c.GetTorrentProperties(testCtx(t), Deref(sampleTorrents[0].Hash))
	require.NoError(t, err)
	assert.Equal(t, "Arch Linux 2024.01.01", Deref(props.Name))
	assert.InDelta(t, 1.23, Deref(props.ShareRatio), 0.001)
}

func TestGetTorrentTrackers(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/torrents/trackers", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "abc123", queryValue(r, "hash"))
		writeJSON(w, []TorrentTracker{
			{URL: Ptr("udp://tracker.example.com:6969"), Status: Ptr(TrackerWorking), NumSeeds: Ptr(100)},
			{URL: Ptr("udp://backup.example.com:80"), Status: Ptr(TrackerNotWorking)},
		})
	})
	c := fs.loggedInClient(t)
	trackers, err := c.GetTorrentTrackers(testCtx(t), "abc123")
	require.NoError(t, err)
	require.Len(t, trackers, 2)
	assert.Equal(t, TrackerWorking, Deref(trackers[0].Status))
	assert.Equal(t, 100, Deref(trackers[0].NumSeeds))
}

func TestGetTorrentFiles(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/torrents/files", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "abc123", queryValue(r, "hash"))
		writeJSON(w, []TorrentFile{
			{Index: Ptr(0), Name: Ptr("arch.iso"), Size: Ptr[int64](800 * 1024 * 1024), Priority: Ptr(FilePriorityNormal), Progress: Ptr(0.42)},
		})
	})
	c := fs.loggedInClient(t)
	files, err := c.GetTorrentFiles(testCtx(t), "abc123", nil)
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "arch.iso", Deref(files[0].Name))
	assert.Equal(t, FilePriorityNormal, Deref(files[0].Priority))
}

func TestGetTorrentFiles_WithIndexes(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/torrents/files", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "0|2", queryValue(r, "indexes"))
		writeJSON(w, []TorrentFile{})
	})
	c := fs.loggedInClient(t)
	_, err := c.GetTorrentFiles(testCtx(t), "abc123", []int{0, 2})
	require.NoError(t, err)
}

func TestGetTorrentWebSeeds(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/torrents/webseeds", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []WebSeed{{URL: "https://seed.example.com/file.iso"}})
	})
	c := fs.loggedInClient(t)
	seeds, err := c.GetTorrentWebSeeds(testCtx(t), "abc")
	require.NoError(t, err)
	require.Len(t, seeds, 1)
	assert.Equal(t, "https://seed.example.com/file.iso", seeds[0].URL)
}

func TestGetTorrentPieceStates(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/torrents/pieceStates", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []int{0, 1, 2, 2, 0, 1})
	})
	c := fs.loggedInClient(t)
	states, err := c.GetTorrentPieceStates(testCtx(t), "abc")
	require.NoError(t, err)
	assert.Equal(t, []int{0, 1, 2, 2, 0, 1}, states)
}

func TestPause(t *testing.T) {
	fs := newFakeServer(t)
	var gotHashes string
	// Simulate newer API (stop).
	fs.handle("POST", "/api/v2/torrents/stop", func(w http.ResponseWriter, r *http.Request) {
		gotHashes = formValue(r, "hashes")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.Pause(testCtx(t), []string{"abc123", "def456"}))
	assert.Equal(t, "abc123|def456", gotHashes)
}

func TestPause_FallbackToLegacy(t *testing.T) {
	fs := newFakeServer(t)
	// /stop returns 404 → should fall back to /pause.
	fs.handle("POST", "/api/v2/torrents/stop", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	var gotHashes string
	fs.handle("POST", "/api/v2/torrents/pause", func(w http.ResponseWriter, r *http.Request) {
		gotHashes = formValue(r, "hashes")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.Pause(testCtx(t), []string{"abc123"}))
	assert.Equal(t, "abc123", gotHashes)
}

func TestResume_FallbackToLegacy(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("POST", "/api/v2/torrents/start", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	var gotHashes string
	fs.handle("POST", "/api/v2/torrents/resume", func(w http.ResponseWriter, r *http.Request) {
		gotHashes = formValue(r, "hashes")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.Resume(testCtx(t), []string{"abc123"}))
	assert.Equal(t, "abc123", gotHashes)
}

func TestDelete_WithoutFiles(t *testing.T) {
	fs := newFakeServer(t)
	var gotDelete, gotFiles string
	fs.handle("POST", "/api/v2/torrents/delete", func(w http.ResponseWriter, r *http.Request) {
		gotDelete = formValue(r, "hashes")
		gotFiles = formValue(r, "deleteFiles")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.Delete(testCtx(t), []string{"abc", "def"}, false))
	assert.Equal(t, "abc|def", gotDelete)
	assert.Equal(t, "false", gotFiles)
}

func TestDelete_WithFiles(t *testing.T) {
	fs := newFakeServer(t)
	var gotFiles string
	fs.handle("POST", "/api/v2/torrents/delete", func(w http.ResponseWriter, r *http.Request) {
		gotFiles = formValue(r, "deleteFiles")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.Delete(testCtx(t), []string{"abc"}, true))
	assert.Equal(t, "true", gotFiles)
}

func TestDeleteAll(t *testing.T) {
	fs := newFakeServer(t)
	var gotHashes string
	fs.handle("POST", "/api/v2/torrents/delete", func(w http.ResponseWriter, r *http.Request) {
		gotHashes = formValue(r, "hashes")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.Delete(testCtx(t), nil, false))
	assert.Equal(t, "all", gotHashes)
}

func TestRecheck(t *testing.T) {
	fs := newFakeServer(t)
	var called bool
	fs.handle("POST", "/api/v2/torrents/recheck", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.Recheck(testCtx(t), []string{"abc"}))
	assert.True(t, called)
}

func TestReannounce(t *testing.T) {
	fs := newFakeServer(t)
	var gotHashes string
	fs.handle("POST", "/api/v2/torrents/reannounce", func(w http.ResponseWriter, r *http.Request) {
		gotHashes = formValue(r, "hashes")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.Reannounce(testCtx(t), []string{"abc123", "def456"}))
	assert.Equal(t, "abc123|def456", gotHashes)
}

func TestAddTorrentFromURL_Success(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("POST", "/api/v2/torrents/add", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "magnet:?xt=urn:btih:abc", formValue(r, "urls"))
		assert.Equal(t, "linux", formValue(r, "category"))
		writeString(w, "Ok.")
	})
	c := fs.loggedInClient(t)
	err := c.AddTorrentFromURL(testCtx(t), "magnet:?xt=urn:btih:abc", TorrentAddOptions{
		Category: "linux",
	})
	require.NoError(t, err)
}

func TestAddTorrentFromURL_AlreadyExists(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("POST", "/api/v2/torrents/add", func(w http.ResponseWriter, _ *http.Request) {
		writeString(w, "Fails.")
	})
	c := fs.loggedInClient(t)
	err := c.AddTorrentFromURL(testCtx(t), "magnet:", TorrentAddOptions{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAlreadyExists)
}

func TestAddTorrentFromURLs_Multiple(t *testing.T) {
	fs := newFakeServer(t)
	var gotURLs string
	fs.handle("POST", "/api/v2/torrents/add", func(w http.ResponseWriter, r *http.Request) {
		gotURLs = formValue(r, "urls")
		writeString(w, "Ok.")
	})
	c := fs.loggedInClient(t)
	err := c.AddTorrentFromURLs(testCtx(t), []string{"magnet:?a", "magnet:?b"}, TorrentAddOptions{})
	require.NoError(t, err)
	assert.Equal(t, "magnet:?a\nmagnet:?b", gotURLs)
}

func TestAddTorrentFromReader(t *testing.T) {
	fs := newFakeServer(t)
	var gotBody bytes.Buffer
	fs.handle("POST", "/api/v2/torrents/add", func(w http.ResponseWriter, r *http.Request) {
		_, _ = gotBody.ReadFrom(r.Body)
		writeString(w, "Ok.")
	})
	c := fs.loggedInClient(t)
	data := bytes.NewReader([]byte("fake torrent data"))
	err := c.AddTorrentFromReader(testCtx(t), "test.torrent", data, TorrentAddOptions{
		SavePath: "/data",
	})
	require.NoError(t, err)
	// The multipart body should contain the file data and extra fields.
	body := gotBody.String()
	assert.Contains(t, body, "fake torrent data")
	assert.Contains(t, body, "/data")
}

func TestAddTorrentFromReader_AlreadyExists(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("POST", "/api/v2/torrents/add", func(w http.ResponseWriter, _ *http.Request) {
		writeString(w, "Fails.")
	})
	c := fs.loggedInClient(t)
	err := c.AddTorrentFromReader(testCtx(t), "t.torrent", bytes.NewReader(nil), TorrentAddOptions{})
	require.ErrorIs(t, err, ErrAlreadyExists)
}

func TestAddTrackers(t *testing.T) {
	fs := newFakeServer(t)
	var gotHash, gotURLs string
	fs.handle("POST", "/api/v2/torrents/addTrackers", func(w http.ResponseWriter, r *http.Request) {
		gotHash = formValue(r, "hash")
		gotURLs = formValue(r, "urls")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.AddTrackers(testCtx(t), "abc123", []string{
		"udp://tracker1.example.com:6969",
		"udp://tracker2.example.com:80",
	}))
	assert.Equal(t, "abc123", gotHash)
	assert.Equal(t, "udp://tracker1.example.com:6969\nudp://tracker2.example.com:80", gotURLs)
}

func TestEditTracker(t *testing.T) {
	fs := newFakeServer(t)
	var gotOrig, gotNew string
	fs.handle("POST", "/api/v2/torrents/editTracker", func(w http.ResponseWriter, r *http.Request) {
		gotOrig = formValue(r, "origUrl")
		gotNew = formValue(r, "newUrl")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.EditTracker(testCtx(t), "abc", "udp://old.example.com:80", "udp://new.example.com:80"))
	assert.Equal(t, "udp://old.example.com:80", gotOrig)
	assert.Equal(t, "udp://new.example.com:80", gotNew)
}

func TestRemoveTrackers(t *testing.T) {
	fs := newFakeServer(t)
	var gotURLs string
	fs.handle("POST", "/api/v2/torrents/removeTrackers", func(w http.ResponseWriter, r *http.Request) {
		gotURLs = formValue(r, "urls")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.RemoveTrackers(testCtx(t), "abc", []string{"udp://a.com:80", "udp://b.com:80"}))
	assert.Equal(t, "udp://a.com:80|udp://b.com:80", gotURLs)
}

func TestSetForceStart(t *testing.T) {
	fs := newFakeServer(t)
	var gotValue string
	fs.handle("POST", "/api/v2/torrents/setForceStart", func(w http.ResponseWriter, r *http.Request) {
		gotValue = formValue(r, "value")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.SetForceStart(testCtx(t), []string{"abc"}, true))
	assert.Equal(t, "true", gotValue)
}

func TestSetAutoManagement(t *testing.T) {
	fs := newFakeServer(t)
	var gotEnable string
	fs.handle("POST", "/api/v2/torrents/setAutoManagement", func(w http.ResponseWriter, r *http.Request) {
		gotEnable = formValue(r, "enable")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.SetAutoManagement(testCtx(t), []string{"abc"}, true))
	assert.Equal(t, "true", gotEnable)
}

func TestGetCategories(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/torrents/categories", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]Category{
			"linux":  {Name: "linux", SavePath: "/data/linux"},
			"movies": {Name: "movies", SavePath: "/data/movies"},
		})
	})
	c := fs.loggedInClient(t)
	cats, err := c.GetCategories(testCtx(t))
	require.NoError(t, err)
	require.Len(t, cats, 2)
	assert.Equal(t, "/data/linux", cats["linux"].SavePath)
}

func TestCreateCategory(t *testing.T) {
	fs := newFakeServer(t)
	var gotName, gotPath string
	fs.handle("POST", "/api/v2/torrents/createCategory", func(w http.ResponseWriter, r *http.Request) {
		gotName = formValue(r, "category")
		gotPath = formValue(r, "savePath")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.CreateCategory(testCtx(t), "anime", "/data/anime"))
	assert.Equal(t, "anime", gotName)
	assert.Equal(t, "/data/anime", gotPath)
}

func TestEditCategory(t *testing.T) {
	fs := newFakeServer(t)
	var gotName, gotPath string
	fs.handle("POST", "/api/v2/torrents/editCategory", func(w http.ResponseWriter, r *http.Request) {
		gotName = formValue(r, "category")
		gotPath = formValue(r, "savePath")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.EditCategory(testCtx(t), "linux", "/mnt/linux"))
	assert.Equal(t, "linux", gotName)
	assert.Equal(t, "/mnt/linux", gotPath)
}

func TestRemoveCategories(t *testing.T) {
	fs := newFakeServer(t)
	var gotCategories string
	fs.handle("POST", "/api/v2/torrents/removeCategories", func(w http.ResponseWriter, r *http.Request) {
		gotCategories = formValue(r, "categories")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.RemoveCategories(testCtx(t), []string{"linux", "movies"}))
	assert.Equal(t, "linux\nmovies", gotCategories)
}

func TestGetTags(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/torrents/tags", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []string{"hdr", "remux", "web-dl"})
	})
	c := fs.loggedInClient(t)
	tags, err := c.GetTags(testCtx(t))
	require.NoError(t, err)
	assert.Equal(t, []string{"hdr", "remux", "web-dl"}, tags)
}

func TestCreateTags(t *testing.T) {
	fs := newFakeServer(t)
	var gotTags string
	fs.handle("POST", "/api/v2/torrents/createTags", func(w http.ResponseWriter, r *http.Request) {
		gotTags = formValue(r, "tags")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.CreateTags(testCtx(t), []string{"hdr", "remux"}))
	assert.Equal(t, "hdr,remux", gotTags)
}

func TestAddTags(t *testing.T) {
	fs := newFakeServer(t)
	var gotHashes, gotTags string
	fs.handle("POST", "/api/v2/torrents/addTags", func(w http.ResponseWriter, r *http.Request) {
		gotHashes = formValue(r, "hashes")
		gotTags = formValue(r, "tags")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.AddTags(testCtx(t), []string{"abc", "def"}, []string{"hdr", "bluray"}))
	assert.Equal(t, "abc|def", gotHashes)
	assert.Equal(t, "hdr,bluray", gotTags)
}

func TestRemoveTags(t *testing.T) {
	fs := newFakeServer(t)
	var gotTags string
	fs.handle("POST", "/api/v2/torrents/removeTags", func(w http.ResponseWriter, r *http.Request) {
		gotTags = formValue(r, "tags")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.RemoveTags(testCtx(t), []string{"abc"}, []string{"hdr"}))
	assert.Equal(t, "hdr", gotTags)
}

func TestSetTorrentCategory(t *testing.T) {
	fs := newFakeServer(t)
	var gotHashes, gotCat string
	fs.handle("POST", "/api/v2/torrents/setCategory", func(w http.ResponseWriter, r *http.Request) {
		gotHashes = formValue(r, "hashes")
		gotCat = formValue(r, "category")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.SetTorrentCategory(testCtx(t), []string{"abc"}, "movies"))
	assert.Equal(t, "abc", gotHashes)
	assert.Equal(t, "movies", gotCat)
}

func TestSetTorrentLocation(t *testing.T) {
	fs := newFakeServer(t)
	var gotLoc string
	fs.handle("POST", "/api/v2/torrents/setLocation", func(w http.ResponseWriter, r *http.Request) {
		gotLoc = formValue(r, "location")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.SetTorrentLocation(testCtx(t), []string{"abc"}, "/mnt/new"))
	assert.Equal(t, "/mnt/new", gotLoc)
}

func TestRenameTorrent(t *testing.T) {
	fs := newFakeServer(t)
	var gotName string
	fs.handle("POST", "/api/v2/torrents/rename", func(w http.ResponseWriter, r *http.Request) {
		gotName = formValue(r, "name")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.RenameTorrent(testCtx(t), "abc", "New Name"))
	assert.Equal(t, "New Name", gotName)
}

func TestRenameFile(t *testing.T) {
	fs := newFakeServer(t)
	var gotOld, gotNew string
	fs.handle("POST", "/api/v2/torrents/renameFile", func(w http.ResponseWriter, r *http.Request) {
		gotOld = formValue(r, "oldPath")
		gotNew = formValue(r, "newPath")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.RenameFile(testCtx(t), "abc", "old.mkv", "new.mkv"))
	assert.Equal(t, "old.mkv", gotOld)
	assert.Equal(t, "new.mkv", gotNew)
}

func TestSetTorrentDownloadLimit(t *testing.T) {
	fs := newFakeServer(t)
	var gotLimit string
	fs.handle("POST", "/api/v2/torrents/setDownloadLimit", func(w http.ResponseWriter, r *http.Request) {
		gotLimit = formValue(r, "limit")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.SetTorrentDownloadLimit(testCtx(t), []string{"abc"}, 512*1024))
	assert.Equal(t, "524288", gotLimit)
}

func TestGetTorrentDownloadLimits(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("POST", "/api/v2/torrents/downloadLimit", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]int64{"abc": 512 * 1024, "def": 0})
	})
	c := fs.loggedInClient(t)
	limits, err := c.GetTorrentDownloadLimits(testCtx(t), []string{"abc", "def"})
	require.NoError(t, err)
	assert.Equal(t, int64(512*1024), limits["abc"])
	assert.Equal(t, int64(0), limits["def"])
}

func TestSetTorrentShareLimits(t *testing.T) {
	fs := newFakeServer(t)
	var gotRatio, gotSeed, gotInactive string
	fs.handle("POST", "/api/v2/torrents/setShareLimits", func(w http.ResponseWriter, r *http.Request) {
		gotRatio = formValue(r, "ratioLimit")
		gotSeed = formValue(r, "seedingTimeLimit")
		gotInactive = formValue(r, "inactiveSeedingTimeLimit")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.SetTorrentShareLimits(testCtx(t), []string{"abc"}, 1.5, 3600, 7200))
	assert.Equal(t, "1.5", gotRatio)
	assert.Equal(t, "3600", gotSeed)
	assert.Equal(t, "7200", gotInactive)
}

func TestSetFilePriority(t *testing.T) {
	fs := newFakeServer(t)
	var gotID, gotPrio string
	fs.handle("POST", "/api/v2/torrents/filePrio", func(w http.ResponseWriter, r *http.Request) {
		gotID = formValue(r, "id")
		gotPrio = formValue(r, "priority")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.SetFilePriority(testCtx(t), "abc", []int{0, 2, 5}, FilePriorityHigh))
	assert.Equal(t, "0|2|5", gotID)
	assert.Equal(t, "6", gotPrio)
}

func TestExportTorrent(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/torrents/export", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "abc123", queryValue(r, "hash"))
		w.Header().Set("Content-Type", "application/x-bittorrent")
		_, _ = w.Write([]byte("d4:infoe"))
	})
	c := fs.loggedInClient(t)
	data, err := c.ExportTorrent(testCtx(t), "abc123")
	require.NoError(t, err)
	assert.Equal(t, "d4:infoe", string(data))
}

func TestSetMaxPriority(t *testing.T) {
	fs := newFakeServer(t)
	var called bool
	fs.handle("POST", "/api/v2/torrents/topPrio", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.SetMaxPriority(testCtx(t), []string{"abc"}))
	assert.True(t, called)
}

func TestSetMinPriority(t *testing.T) {
	fs := newFakeServer(t)
	var called bool
	fs.handle("POST", "/api/v2/torrents/bottomPrio", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.SetMinPriority(testCtx(t), []string{"abc"}))
	assert.True(t, called)
}

func TestToggleSequentialDownload(t *testing.T) {
	fs := newFakeServer(t)
	var called bool
	fs.handle("POST", "/api/v2/torrents/toggleSequentialDownload", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.ToggleSequentialDownload(testCtx(t), []string{"abc"}))
	assert.True(t, called)
}

func TestToggleFirstLastPiecePrio(t *testing.T) {
	fs := newFakeServer(t)
	var called bool
	fs.handle("POST", "/api/v2/torrents/toggleFirstLastPiecePrio", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.ToggleFirstLastPiecePrio(testCtx(t), []string{"abc"}))
	assert.True(t, called)
}

func TestSetSuperSeeding(t *testing.T) {
	fs := newFakeServer(t)
	var gotValue string
	fs.handle("POST", "/api/v2/torrents/setSuperSeeding", func(w http.ResponseWriter, r *http.Request) {
		gotValue = formValue(r, "value")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.SetSuperSeeding(testCtx(t), []string{"abc"}, true))
	assert.Equal(t, "true", gotValue)
}

func TestAddPeers(t *testing.T) {
	fs := newFakeServer(t)
	var gotPeers string
	fs.handle("POST", "/api/v2/torrents/addPeers", func(w http.ResponseWriter, r *http.Request) {
		gotPeers = formValue(r, "peers")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.AddPeers(testCtx(t), []string{"abc"}, []string{"1.2.3.4:6881", "5.6.7.8:6882"}))
	assert.Equal(t, "1.2.3.4:6881|5.6.7.8:6882", gotPeers)
}
