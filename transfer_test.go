package qbittorrent

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTransferInfo(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/transfer/info", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, TransferInfo{
			DlInfoSpeed:      Ptr[int64](1024 * 100),
			UpInfoSpeed:      Ptr[int64](1024 * 50),
			DlRateLimit:      Ptr[int64](0),
			UpRateLimit:      Ptr[int64](0),
			DHTNodes:         Ptr[int64](42),
			ConnectionStatus: Ptr("connected"),
			FreeSpace:        Ptr[int64](10 * 1024 * 1024 * 1024),
		})
	})
	c := fs.loggedInClient(t)
	info, err := c.GetTransferInfo(testCtx(t))
	require.NoError(t, err)
	assert.Equal(t, int64(1024*100), Deref(info.DlInfoSpeed))
	assert.Equal(t, "connected", Deref(info.ConnectionStatus))
	assert.Equal(t, int64(42), Deref(info.DHTNodes))
}

func TestGetAlternativeSpeedLimitsMode(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/transfer/speedLimitsMode", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 1)
	})
	c := fs.loggedInClient(t)
	mode, err := c.GetAlternativeSpeedLimitsMode(testCtx(t))
	require.NoError(t, err)
	assert.Equal(t, 1, mode)
}

func TestToggleAlternativeSpeedLimits(t *testing.T) {
	fs := newFakeServer(t)
	called := false
	fs.handle("POST", "/api/v2/transfer/toggleSpeedLimitsMode", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.ToggleAlternativeSpeedLimits(testCtx(t)))
	assert.True(t, called)
}

func TestGetGlobalDownloadLimit(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/transfer/downloadLimit", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, int64(5*1024*1024))
	})
	c := fs.loggedInClient(t)
	limit, err := c.GetGlobalDownloadLimit(testCtx(t))
	require.NoError(t, err)
	assert.Equal(t, int64(5*1024*1024), limit)
}

func TestSetGlobalDownloadLimit(t *testing.T) {
	fs := newFakeServer(t)
	var gotLimit string
	fs.handle("POST", "/api/v2/transfer/setDownloadLimit", func(w http.ResponseWriter, r *http.Request) {
		gotLimit = formValue(r, "limit")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.SetGlobalDownloadLimit(testCtx(t), 10*1024*1024))
	assert.Equal(t, "10485760", gotLimit)
}

func TestGetGlobalUploadLimit(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/transfer/uploadLimit", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, int64(2*1024*1024))
	})
	c := fs.loggedInClient(t)
	limit, err := c.GetGlobalUploadLimit(testCtx(t))
	require.NoError(t, err)
	assert.Equal(t, int64(2*1024*1024), limit)
}

func TestSetGlobalUploadLimit(t *testing.T) {
	fs := newFakeServer(t)
	var gotLimit string
	fs.handle("POST", "/api/v2/transfer/setUploadLimit", func(w http.ResponseWriter, r *http.Request) {
		gotLimit = formValue(r, "limit")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.SetGlobalUploadLimit(testCtx(t), 1024*1024))
	assert.Equal(t, "1048576", gotLimit)
}

func TestBanPeers(t *testing.T) {
	fs := newFakeServer(t)
	var gotPeers string
	fs.handle("POST", "/api/v2/transfer/banPeers", func(w http.ResponseWriter, r *http.Request) {
		gotPeers = formValue(r, "peers")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.BanPeers(testCtx(t), []string{"1.2.3.4:6881", "5.6.7.8:6881"}))
	assert.Equal(t, "1.2.3.4:6881|5.6.7.8:6881", gotPeers)
}
