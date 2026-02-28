package qbittorrent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- APIError ----

func TestAPIError_WithBody(t *testing.T) {
	err := apiErr(ErrNotFound, 404, "torrent not found")
	assert.Contains(t, err.Error(), "404")
	assert.Contains(t, err.Error(), "torrent not found")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestAPIError_WithoutBody(t *testing.T) {
	err := apiErr(ErrConflict, 409, "")
	assert.Contains(t, err.Error(), "409")
	assert.NotContains(t, err.Error(), "body")
	assert.ErrorIs(t, err, ErrConflict)
}

func TestAPIError_Unwrap(t *testing.T) {
	err := apiErr(ErrBadRequest, 400, "oops")
	assert.ErrorIs(t, err, ErrBadRequest)
}

func TestAPIError_As(t *testing.T) {
	err := apiErr(ErrConflict, 409, "clash")
	var ae *APIError
	require.ErrorAs(t, err, &ae)
	assert.Equal(t, 409, ae.StatusCode)
	assert.Equal(t, "clash", ae.Body)
	assert.Equal(t, ErrConflict, ae.Sentinel)
}

// ---- Sentinel errors are distinct ----

func TestSentinels_AreDistinct(t *testing.T) {
	sentinels := []error{
		ErrBadResponse, ErrUnauthorized, ErrNotFound, ErrConflict,
		ErrBadRequest, ErrLoginFailed, ErrIPBanned, ErrInvalidTorrent, ErrAlreadyExists,
	}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i != j {
				assert.NotEqual(t, a, b, "sentinels %d and %d are equal", i, j)
			}
		}
	}
}

// ---- TorrentState helpers ----

func TestTorrentState_IsDownloading(t *testing.T) {
	downloading := []TorrentState{
		StateDownloading, StateMetaDL, StateStalledDL, StateQueuedDL,
		StateForcedDL, StateCheckingDL, StateAllocating,
	}
	for _, s := range downloading {
		assert.True(t, s.IsDownloading(), "expected %q to be downloading", s)
	}
	assert.False(t, StateUploading.IsDownloading())
	assert.False(t, StateError.IsDownloading())
}

func TestTorrentState_IsUploading(t *testing.T) {
	uploading := []TorrentState{StateUploading, StateStalledUP, StateQueuedUP, StateForcedUP}
	for _, s := range uploading {
		assert.True(t, s.IsUploading(), "expected %q to be uploading", s)
	}
	assert.False(t, StateDownloading.IsUploading())
}

func TestTorrentState_IsPaused(t *testing.T) {
	paused := []TorrentState{StatePausedDL, StatePausedUP, StateStoppedDL, StateStoppedUP}
	for _, s := range paused {
		assert.True(t, s.IsPaused(), "expected %q to be paused", s)
	}
	assert.False(t, StateDownloading.IsPaused())
}

func TestTorrentState_IsComplete(t *testing.T) {
	complete := []TorrentState{StateUploading, StateStalledUP, StateForcedUP, StateQueuedUP, StateCheckingUP}
	for _, s := range complete {
		assert.True(t, s.IsComplete(), "expected %q to be complete", s)
	}
	assert.False(t, StateDownloading.IsComplete())
}
