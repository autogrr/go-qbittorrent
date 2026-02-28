package qbittorrent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/bytedance/sonic"
)

// -----------------------------------------------------------------------
// Fixture generators
// -----------------------------------------------------------------------

// makeTorrents returns a map of n synthetic Torrent entries keyed by a
// hex-like hash string. The content is varied enough to exercise both the
// JSON parser and the merge logic.
func makeTorrents(n int) map[string]Torrent {
	m := make(map[string]Torrent, n)
	for i := 0; i < n; i++ {
		hash := fmt.Sprintf("%040x", i)
		m[hash] = Torrent{
			Hash:       Ptr(hash),
			Name:       Ptr(fmt.Sprintf("Torrent-%d", i)),
			State:      Ptr(StateDownloading),
			Size:       Ptr(int64(i) * 1024 * 1024),
			Downloaded: Ptr(int64(i) * 512 * 1024),
			DlSpeed:    Ptr(int64(i % 10000)),
			UpSpeed:    Ptr(int64(i % 5000)),
			Progress:   Ptr(float64(i%100) / 100.0),
			Ratio:      Ptr(float64(i%30) / 10.0),
			Category:   Ptr(fmt.Sprintf("cat-%d", i%1000)),
			Tags:       Ptr(fmt.Sprintf("tag%d,tag%d", i%50, (i+1)%50)),
			SavePath:   Ptr(fmt.Sprintf("/data/torrents/%d", i%500)),
			Tracker:    Ptr(fmt.Sprintf("udp://tracker%d.example.com:6969", i%20)),
			ETA:        Ptr(int64(i % 86400)),
			NumSeeds:   Ptr(i % 500),
			NumLeechs:  Ptr(i % 200),
			AddedOn:    Ptr(int64(1700000000 + i)),
		}
	}
	return m
}

// makeCategories returns n synthetic categories.
func makeCategories(n int) map[string]Category {
	m := make(map[string]Category, n)
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("cat-%d", i)
		m[name] = Category{
			Name:     name,
			SavePath: fmt.Sprintf("/data/%d", i),
		}
	}
	return m
}

// makeMainDataJSON serialises a MainData with n torrents to JSON bytes.
// Used to benchmark the JSON decode path.
func makeMainDataJSON(n int) []byte {
	md := MainData{
		Rid:        1,
		FullUpdate: true,
		Torrents:   makeTorrents(n),
		Categories: makeCategories(n / 2),
		Tags:       []string{"hdr", "remux", "web-dl", "bluray"},
		ServerState: ServerState{
			ConnectionStatus: Ptr("connected"),
			DlInfoSpeed:      Ptr[int64](5 * 1024 * 1024),
			FreeSpaceOnDisk:  Ptr[int64](500 * 1024 * 1024 * 1024),
		},
	}
	data, err := sonic.Marshal(md)
	if err != nil {
		panic(err)
	}
	return data
}

// populatedState returns a SyncState with n torrents and n/2 categories,
// ready for read benchmarks.
func populatedState(n int) *SyncState {
	s := newSyncState()
	s.apply(&MainData{
		Rid:         1,
		FullUpdate:  true,
		Torrents:    makeTorrents(n),
		Categories:  makeCategories(n / 2),
		Tags:        []string{"hdr", "remux"},
		ServerState: ServerState{ConnectionStatus: Ptr("connected")},
	})
	return s
}

// -----------------------------------------------------------------------
// SyncState apply
// -----------------------------------------------------------------------

func BenchmarkApplyFullUpdate_200k(b *testing.B) {
	torrents := makeTorrents(200_000)
	cats := makeCategories(100_000)
	d := &MainData{
		Rid:         1,
		FullUpdate:  true,
		Torrents:    torrents,
		Categories:  cats,
		Tags:        []string{"hdr", "remux"},
		ServerState: ServerState{ConnectionStatus: Ptr("connected")},
	}
	s := newSyncState()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d.Rid = i + 1
		s.apply(d)
	}
}

func BenchmarkApplyPartialUpdate_10k(b *testing.B) {
	// Start with 200k torrents already loaded.
	s := populatedState(200_000)
	// Build a partial diff touching 10k entries (5% churn).
	diff := make(map[string]Torrent, 10_000)
	for i := 0; i < 10_000; i++ {
		hash := fmt.Sprintf("%040x", i)
		diff[hash] = Torrent{DlSpeed: Ptr(int64(i * 2))}
	}
	d := &MainData{
		Rid:         2,
		FullUpdate:  false,
		Torrents:    diff,
		ServerState: ServerState{DlInfoSpeed: Ptr[int64](9 * 1024 * 1024)},
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d.Rid = 2 + i
		s.apply(d)
	}
}

func BenchmarkApplyPartialUpdate_1k(b *testing.B) {
	s := populatedState(200_000)
	diff := make(map[string]Torrent, 1_000)
	for i := 0; i < 1_000; i++ {
		hash := fmt.Sprintf("%040x", i)
		diff[hash] = Torrent{DlSpeed: Ptr(int64(i * 3))}
	}
	d := &MainData{Rid: 2, FullUpdate: false, Torrents: diff}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d.Rid = 2 + i
		s.apply(d)
	}
}

// -----------------------------------------------------------------------
// SyncState reads — copy vs visitor
// -----------------------------------------------------------------------

func BenchmarkGetTorrents_200k(b *testing.B) {
	s := populatedState(200_000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m := s.GetTorrents()
		_ = m
	}
}

func BenchmarkVisitTorrents_200k(b *testing.B) {
	s := populatedState(200_000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var count int
		s.VisitTorrents(func(_ string, _ Torrent) bool {
			count++
			return true
		})
		_ = count
	}
}

func BenchmarkGetTorrentSlice_200k(b *testing.B) {
	s := populatedState(200_000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sl := s.GetTorrentSlice()
		_ = sl
	}
}

func BenchmarkGetCategories_100k(b *testing.B) {
	s := populatedState(200_000) // has 100k categories
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m := s.GetCategories()
		_ = m
	}
}

func BenchmarkVisitCategories_100k(b *testing.B) {
	s := populatedState(200_000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var count int
		s.VisitCategories(func(_ string, _ Category) bool {
			count++
			return true
		})
		_ = count
	}
}

func BenchmarkGetTorrent_Lookup(b *testing.B) {
	s := populatedState(200_000)
	hash := fmt.Sprintf("%040x", 99_999)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		t, _ := s.GetTorrent(hash)
		_ = t
	}
}

func BenchmarkLen_200k(b *testing.B) {
	s := populatedState(200_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Len()
	}
}

// -----------------------------------------------------------------------
// Concurrent read contention
// -----------------------------------------------------------------------

func BenchmarkVisitTorrents_Parallel(b *testing.B) {
	s := populatedState(200_000)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var count int
			s.VisitTorrents(func(_ string, _ Torrent) bool {
				count++
				return true
			})
			_ = count
		}
	})
}

func BenchmarkGetTorrents_Parallel(b *testing.B) {
	s := populatedState(200_000)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m := s.GetTorrents()
			_ = m
		}
	})
}

// -----------------------------------------------------------------------
// mergePartialTorrent
// -----------------------------------------------------------------------

func BenchmarkMergePartialTorrent(b *testing.B) {
	dst := Torrent{
		Hash:     Ptr("aabbccddeeff00112233445566778899aabbccdd"),
		Name:     Ptr("Some Torrent"),
		State:    Ptr(StateDownloading),
		DlSpeed:  Ptr[int64](1024 * 1024),
		Progress: Ptr(0.42),
		Category: Ptr("linux"),
	}
	src := Torrent{
		DlSpeed: Ptr[int64](2 * 1024 * 1024),
		UpSpeed: Ptr[int64](512 * 1024),
		ETA:     Ptr[int64](3600),
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := dst
		mergePartialTorrent(&d, &src)
	}
}

// -----------------------------------------------------------------------
// JSON decode — sonic unmarshal vs streaming vs stdlib
// -----------------------------------------------------------------------

func BenchmarkJSONUnmarshal_Sonic_200k(b *testing.B) {
	data := makeMainDataJSON(200_000)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var md MainData
		if err := sonic.Unmarshal(data, &md); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONUnmarshal_SonicStream_200k(b *testing.B) {
	data := makeMainDataJSON(200_000)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var md MainData
		if err := sonic.ConfigDefault.NewDecoder(bytes.NewReader(data)).Decode(&md); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONUnmarshal_Stdlib_200k(b *testing.B) {
	data := makeMainDataJSON(200_000)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var md MainData
		if err := json.Unmarshal(data, &md); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONUnmarshal_Sonic_1k(b *testing.B) {
	data := makeMainDataJSON(1_000)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var md MainData
		if err := sonic.Unmarshal(data, &md); err != nil {
			b.Fatal(err)
		}
	}
}

// -----------------------------------------------------------------------
// End-to-end: readBody + unmarshal (simulates the full HTTP path)
// -----------------------------------------------------------------------

func BenchmarkReadAndUnmarshal_200k(b *testing.B) {
	data := makeMainDataJSON(200_000)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		body, err := io.ReadAll(bytes.NewReader(data))
		if err != nil {
			b.Fatal(err)
		}
		var md MainData
		if err := sonic.Unmarshal(body, &md); err != nil {
			b.Fatal(err)
		}
		_ = md
	}
}

func BenchmarkStreamDecode_200k(b *testing.B) {
	data := makeMainDataJSON(200_000)
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var md MainData
		if err := sonic.ConfigDefault.NewDecoder(bytes.NewReader(data)).Decode(&md); err != nil {
			b.Fatal(err)
		}
		_ = md
	}
}
