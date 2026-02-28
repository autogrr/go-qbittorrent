package qbittorrent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortTorrents_ByName(t *testing.T) {
	torrents := []Torrent{
		{Hash: Ptr("c"), Name: Ptr("Charlie")},
		{Hash: Ptr("a"), Name: Ptr("Alpha")},
		{Hash: Ptr("b"), Name: Ptr("Bravo")},
	}
	SortTorrents(torrents, TorrentSort{Field: "name"})
	assert.Equal(t, "Alpha", Deref(torrents[0].Name))
	assert.Equal(t, "Bravo", Deref(torrents[1].Name))
	assert.Equal(t, "Charlie", Deref(torrents[2].Name))
}

func TestSortTorrents_ByNameReverse(t *testing.T) {
	torrents := []Torrent{
		{Hash: Ptr("a"), Name: Ptr("Alpha")},
		{Hash: Ptr("b"), Name: Ptr("Bravo")},
		{Hash: Ptr("c"), Name: Ptr("Charlie")},
	}
	SortTorrents(torrents, TorrentSort{Field: "name", Reverse: true})
	assert.Equal(t, "Charlie", Deref(torrents[0].Name))
	assert.Equal(t, "Bravo", Deref(torrents[1].Name))
	assert.Equal(t, "Alpha", Deref(torrents[2].Name))
}

func TestSortTorrents_ByAddedOn(t *testing.T) {
	torrents := []Torrent{
		{Hash: Ptr("b"), AddedOn: Ptr(int64(200))},
		{Hash: Ptr("a"), AddedOn: Ptr(int64(100))},
		{Hash: Ptr("c"), AddedOn: Ptr(int64(300))},
	}
	SortTorrents(torrents, TorrentSort{Field: "added_on"})
	assert.Equal(t, int64(100), Deref(torrents[0].AddedOn))
	assert.Equal(t, int64(200), Deref(torrents[1].AddedOn))
	assert.Equal(t, int64(300), Deref(torrents[2].AddedOn))
}

func TestSortTorrents_BySize(t *testing.T) {
	torrents := []Torrent{
		{Hash: Ptr("a"), Size: Ptr(int64(500))},
		{Hash: Ptr("b"), Size: Ptr(int64(100))},
		{Hash: Ptr("c"), Size: Ptr(int64(300))},
	}
	SortTorrents(torrents, TorrentSort{Field: "size"})
	assert.Equal(t, int64(100), Deref(torrents[0].Size))
	assert.Equal(t, int64(300), Deref(torrents[1].Size))
	assert.Equal(t, int64(500), Deref(torrents[2].Size))
}

func TestSortTorrents_ByProgress(t *testing.T) {
	torrents := []Torrent{
		{Hash: Ptr("a"), Progress: Ptr(0.75)},
		{Hash: Ptr("b"), Progress: Ptr(0.25)},
		{Hash: Ptr("c"), Progress: Ptr(1.0)},
	}
	SortTorrents(torrents, TorrentSort{Field: "progress", Reverse: true})
	assert.Equal(t, 1.0, Deref(torrents[0].Progress))
	assert.Equal(t, 0.75, Deref(torrents[1].Progress))
	assert.Equal(t, 0.25, Deref(torrents[2].Progress))
}

func TestSortTorrents_NilFieldsFirst(t *testing.T) {
	torrents := []Torrent{
		{Hash: Ptr("a"), Name: Ptr("Alpha")},
		{Hash: Ptr("b"), Name: nil},
		{Hash: Ptr("c"), Name: Ptr("Charlie")},
	}
	SortTorrents(torrents, TorrentSort{Field: "name"})
	assert.Nil(t, torrents[0].Name, "nil should sort first")
	assert.Equal(t, "Alpha", Deref(torrents[1].Name))
	assert.Equal(t, "Charlie", Deref(torrents[2].Name))
}

func TestSortTorrents_NilFieldsLastWhenReversed(t *testing.T) {
	torrents := []Torrent{
		{Hash: Ptr("a"), Name: Ptr("Alpha")},
		{Hash: Ptr("b"), Name: nil},
		{Hash: Ptr("c"), Name: Ptr("Charlie")},
	}
	SortTorrents(torrents, TorrentSort{Field: "name", Reverse: true})
	assert.Equal(t, "Charlie", Deref(torrents[0].Name))
	assert.Equal(t, "Alpha", Deref(torrents[1].Name))
	assert.Nil(t, torrents[2].Name, "nil should sort last when reversed")
}

func TestSortTorrents_UnknownFieldFallsBackToHash(t *testing.T) {
	torrents := []Torrent{
		{Hash: Ptr("c")},
		{Hash: Ptr("a")},
		{Hash: Ptr("b")},
	}
	SortTorrents(torrents, TorrentSort{Field: "nonexistent"})
	assert.Equal(t, "a", Deref(torrents[0].Hash))
	assert.Equal(t, "b", Deref(torrents[1].Hash))
	assert.Equal(t, "c", Deref(torrents[2].Hash))
}

func TestSortTorrents_EmptyFieldFallsBackToHash(t *testing.T) {
	torrents := []Torrent{
		{Hash: Ptr("b")},
		{Hash: Ptr("a")},
	}
	SortTorrents(torrents, TorrentSort{})
	assert.Equal(t, "a", Deref(torrents[0].Hash))
	assert.Equal(t, "b", Deref(torrents[1].Hash))
}

func TestSortTorrents_ByState(t *testing.T) {
	torrents := []Torrent{
		{Hash: Ptr("a"), State: Ptr(StatePausedDL)},
		{Hash: Ptr("b"), State: Ptr(StateDownloading)},
		{Hash: Ptr("c"), State: Ptr(StateUploading)},
	}
	SortTorrents(torrents, TorrentSort{Field: "state"})
	// Alphabetical: downloading < pausedDL < uploading
	assert.Equal(t, StateDownloading, Deref(torrents[0].State))
	assert.Equal(t, StatePausedDL, Deref(torrents[1].State))
	assert.Equal(t, StateUploading, Deref(torrents[2].State))
}

func TestSortTorrents_ByBoolField(t *testing.T) {
	torrents := []Torrent{
		{Hash: Ptr("a"), Private: Ptr(true)},
		{Hash: Ptr("b"), Private: Ptr(false)},
		{Hash: Ptr("c"), Private: nil},
	}
	SortTorrents(torrents, TorrentSort{Field: "private"})
	// nil < false < true
	assert.Nil(t, torrents[0].Private)
	assert.Equal(t, false, Deref(torrents[1].Private))
	assert.Equal(t, true, Deref(torrents[2].Private))
}

func TestSortTorrents_EmptySlice(t *testing.T) {
	var torrents []Torrent
	SortTorrents(torrents, TorrentSort{Field: "name"})
	assert.Empty(t, torrents)
}

func TestSortTorrents_SingleElement(t *testing.T) {
	torrents := []Torrent{{Hash: Ptr("a"), Name: Ptr("Only")}}
	SortTorrents(torrents, TorrentSort{Field: "name"})
	assert.Equal(t, "Only", Deref(torrents[0].Name))
}

func TestGetTorrentSlice_Sorted(t *testing.T) {
	ss := newSyncState()
	ss.apply(&MainData{
		Rid:        1,
		FullUpdate: true,
		Torrents: map[string]Torrent{
			"hash_c": {Hash: Ptr("hash_c"), Name: Ptr("Charlie"), AddedOn: Ptr(int64(300))},
			"hash_a": {Hash: Ptr("hash_a"), Name: Ptr("Alpha"), AddedOn: Ptr(int64(100))},
			"hash_b": {Hash: Ptr("hash_b"), Name: Ptr("Bravo"), AddedOn: Ptr(int64(200))},
		},
	})

	// Default: sorted by hash
	sl := ss.GetTorrentSlice()
	assert.Equal(t, "hash_a", Deref(sl[0].Hash))
	assert.Equal(t, "hash_b", Deref(sl[1].Hash))
	assert.Equal(t, "hash_c", Deref(sl[2].Hash))

	// Sort by name
	sl = ss.GetTorrentSlice(TorrentSort{Field: "name"})
	assert.Equal(t, "Alpha", Deref(sl[0].Name))
	assert.Equal(t, "Bravo", Deref(sl[1].Name))
	assert.Equal(t, "Charlie", Deref(sl[2].Name))

	// Sort by added_on descending (newest first)
	sl = ss.GetTorrentSlice(TorrentSort{Field: "added_on", Reverse: true})
	assert.Equal(t, int64(300), Deref(sl[0].AddedOn))
	assert.Equal(t, int64(200), Deref(sl[1].AddedOn))
	assert.Equal(t, int64(100), Deref(sl[2].AddedOn))
}

func TestCmpTorrentPrivate(t *testing.T) {
	mkT := func(p *bool) *Torrent { return &Torrent{Private: p} }
	assert.Equal(t, 0, cmpTorrentPrivate(mkT(nil), mkT(nil)))
	assert.Equal(t, -1, cmpTorrentPrivate(mkT(nil), mkT(Ptr(false))))
	assert.Equal(t, 1, cmpTorrentPrivate(mkT(Ptr(false)), mkT(nil)))
	assert.Equal(t, 0, cmpTorrentPrivate(mkT(Ptr(true)), mkT(Ptr(true))))
	assert.Equal(t, 0, cmpTorrentPrivate(mkT(Ptr(false)), mkT(Ptr(false))))
	assert.Equal(t, -1, cmpTorrentPrivate(mkT(Ptr(false)), mkT(Ptr(true))))
	assert.Equal(t, 1, cmpTorrentPrivate(mkT(Ptr(true)), mkT(Ptr(false))))
}
