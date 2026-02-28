package qbittorrent

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLogs(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/log/main", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "true", queryValue(r, "normal"))
		assert.Equal(t, "true", queryValue(r, "warning"))
		writeJSON(w, []LogEntry{
			{ID: Ptr(1), Message: Ptr("qBittorrent started"), Timestamp: Ptr[int64](1700000000), Type: Ptr(1)},
			{ID: Ptr(2), Message: Ptr("Torrent added"), Timestamp: Ptr[int64](1700000010), Type: Ptr(2)},
		})
	})
	c := fs.loggedInClient(t)
	logs, err := c.GetLogs(testCtx(t), LogOptions{Normal: true, Warning: true})
	require.NoError(t, err)
	require.Len(t, logs, 2)
	assert.Equal(t, 1, Deref(logs[0].ID))
	assert.Equal(t, "qBittorrent started", Deref(logs[0].Message))
}

func TestGetLogs_LastKnownID(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/log/main", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "99", queryValue(r, "last_known_id"))
		writeJSON(w, []LogEntry{})
	})
	c := fs.loggedInClient(t)
	logs, err := c.GetLogs(testCtx(t), LogOptions{Normal: true, LastKnownID: 99})
	require.NoError(t, err)
	assert.Empty(t, logs)
}

func TestGetPeerLogs(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/log/peers", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "5", queryValue(r, "last_known_id"))
		writeJSON(w, []PeerLogEntry{
			{ID: Ptr(6), IP: Ptr("1.2.3.4"), Blocked: Ptr(true), Reason: Ptr("spam filter")},
		})
	})
	c := fs.loggedInClient(t)
	logs, err := c.GetPeerLogs(testCtx(t), 5)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, "1.2.3.4", Deref(logs[0].IP))
	assert.True(t, Deref(logs[0].Blocked))
}

func TestGetPeerLogs_AllEntries(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/log/peers", func(w http.ResponseWriter, r *http.Request) {
		// last_known_id should not be set when -1 is passed.
		assert.Empty(t, queryValue(r, "last_known_id"))
		writeJSON(w, []PeerLogEntry{})
	})
	c := fs.loggedInClient(t)
	_, err := c.GetPeerLogs(testCtx(t), -1)
	require.NoError(t, err)
}
