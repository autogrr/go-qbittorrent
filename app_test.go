package qbittorrent

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAppVersion(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/app/version", func(w http.ResponseWriter, _ *http.Request) {
		writeString(w, "5.0.3")
	})
	c := fs.loggedInClient(t)
	ver, err := c.GetAppVersion(testCtx(t))
	require.NoError(t, err)
	assert.Equal(t, "5.0.3", ver)
}

func TestGetWebAPIVersion(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/app/webapiVersion", func(w http.ResponseWriter, _ *http.Request) {
		writeString(w, "2.11.4")
	})
	c := fs.loggedInClient(t)
	ver, err := c.GetWebAPIVersion(testCtx(t))
	require.NoError(t, err)
	assert.Equal(t, "2.11.4", ver)
}

func TestGetBuildInfo(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/app/buildInfo", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, BuildInfo{
			Bitness:    Ptr(64),
			Boost:      Ptr("1.84.0"),
			LibTorrent: Ptr("2.0.10"),
			OpenSSL:    Ptr("3.2.1"),
			Qt:         Ptr("6.7.0"),
			Zlib:       Ptr("1.3"),
		})
	})
	c := fs.loggedInClient(t)
	info, err := c.GetBuildInfo(testCtx(t))
	require.NoError(t, err)
	assert.Equal(t, 64, Deref(info.Bitness))
	assert.Equal(t, "2.0.10", Deref(info.LibTorrent))
	assert.Equal(t, "6.7.0", Deref(info.Qt))
}

func TestGetDefaultSavePath(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/app/defaultSavePath", func(w http.ResponseWriter, _ *http.Request) {
		writeString(w, "/data/downloads")
	})
	c := fs.loggedInClient(t)
	path, err := c.GetDefaultSavePath(testCtx(t))
	require.NoError(t, err)
	assert.Equal(t, "/data/downloads", path)
}

func TestGetPreferences(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/app/preferences", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, AppPreferences{
			SavePath:                  "/data/downloads",
			MaxActiveDownloads:        3,
			MaxActiveTorrents:         5,
			QueueingEnabled:           true,
			RSSAutoDownloadingEnabled: true,
		})
	})
	c := fs.loggedInClient(t)
	prefs, err := c.GetPreferences(testCtx(t))
	require.NoError(t, err)
	assert.Equal(t, "/data/downloads", prefs.SavePath)
	assert.Equal(t, 3, prefs.MaxActiveDownloads)
	assert.True(t, prefs.QueueingEnabled)
}

func TestSetPreferences(t *testing.T) {
	fs := newFakeServer(t)
	var gotJSON string
	fs.handle("POST", "/api/v2/app/setPreferences", func(w http.ResponseWriter, r *http.Request) {
		gotJSON = formValue(r, "json")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	err := c.SetPreferences(testCtx(t), map[string]any{
		"queueing_enabled":     true,
		"max_active_downloads": 4,
	})
	require.NoError(t, err)
	assert.Contains(t, gotJSON, "queueing_enabled")
	assert.Contains(t, gotJSON, "max_active_downloads")
}

func TestSetQueueingEnabled(t *testing.T) {
	fs := newFakeServer(t)
	var gotJSON string
	fs.handle("POST", "/api/v2/app/setPreferences", func(w http.ResponseWriter, r *http.Request) {
		gotJSON = formValue(r, "json")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.SetQueueingEnabled(testCtx(t), true))
	assert.Contains(t, gotJSON, "queueing_enabled")
	assert.Contains(t, gotJSON, "true")
}

func TestGetCookies(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/app/cookies", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []Cookie{
			{Name: "uid", Domain: "tracker.example.com", Value: "secret123", Path: "/"},
		})
	})
	c := fs.loggedInClient(t)
	cookies, err := c.GetCookies(testCtx(t))
	require.NoError(t, err)
	require.Len(t, cookies, 1)
	assert.Equal(t, "uid", cookies[0].Name)
	assert.Equal(t, "secret123", cookies[0].Value)
}

func TestSetCookies(t *testing.T) {
	fs := newFakeServer(t)
	var gotCookies string
	fs.handle("POST", "/api/v2/app/setCookies", func(w http.ResponseWriter, r *http.Request) {
		gotCookies = formValue(r, "cookies")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	err := c.SetCookies(testCtx(t), []Cookie{
		{Name: "auth", Domain: ".tracker.example.com", Value: "token999"},
	})
	require.NoError(t, err)
	assert.Contains(t, gotCookies, "auth")
	assert.Contains(t, gotCookies, "token999")
}

func TestShutdown(t *testing.T) {
	fs := newFakeServer(t)
	called := false
	fs.handle("POST", "/api/v2/app/shutdown", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.Shutdown(testCtx(t)))
	assert.True(t, called)
}
