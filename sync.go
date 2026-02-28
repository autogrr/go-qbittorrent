package qbittorrent

//go:generate go run ./cmd/gen

import (
	"context"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// SyncOptions configures the SyncManager behaviour.
type SyncOptions struct {
	// Interval is the base polling interval (default 1s).
	Interval time.Duration

	// MinInterval is the floor for dynamic intervals (default 500ms).
	MinInterval time.Duration

	// MaxInterval is the ceiling for dynamic intervals (default 10s).
	MaxInterval time.Duration

	// DynamicInterval enables automatic interval adaptation based on
	// whether the most recent sync returned new data (default true).
	DynamicInterval bool

	// OnUpdate is called after every successful sync that changes state.
	OnUpdate func(*SyncState)

	// OnError is called on sync errors. Return true to stop syncing.
	OnError func(error) bool
}

func (o *SyncOptions) setDefaults() {
	if o.Interval == 0 {
		o.Interval = time.Second
	}
	if o.MinInterval == 0 {
		o.MinInterval = 500 * time.Millisecond
	}
	if o.MaxInterval == 0 {
		o.MaxInterval = 10 * time.Second
	}
}

// SyncState is the aggregated maindata state kept current by SyncManager.
type SyncState struct {
	mu          sync.RWMutex
	rid         int
	Torrents    map[string]Torrent
	Categories  map[string]Category
	Tags        []string
	Trackers    map[string][]string
	ServerState ServerState
}

func newSyncState() *SyncState {
	return &SyncState{
		Torrents:   make(map[string]Torrent),
		Categories: make(map[string]Category),
		Trackers:   make(map[string][]string),
	}
}

// GetTorrent returns the torrent with the given hash, and whether it was found.
// The returned Torrent is a fully independent copy: pointer fields are
// deep-copied so the caller cannot mutate internal state through them.
func (s *SyncState) GetTorrent(hash string) (Torrent, bool) {
	s.mu.RLock()
	t, ok := s.Torrents[hash]
	s.mu.RUnlock()
	if ok {
		t = *deepCopyTorrent(&t)
	}
	return t, ok
}

// GetTorrents returns a deep copy of the torrent map.
// Pointer fields within each Torrent are independently copied.
func (s *SyncState) GetTorrents() map[string]Torrent {
	s.mu.RLock()
	out := make(map[string]Torrent, len(s.Torrents))
	for k, v := range s.Torrents {
		out[k] = *deepCopyTorrent(&v)
	}
	s.mu.RUnlock()
	return out
}

// GetTorrentSlice returns all torrents as a slice sorted by hash.
// The order is deterministic across calls regardless of map iteration order.
// Pointer fields within each Torrent are independently copied.
func (s *SyncState) GetTorrentSlice() []Torrent {
	s.mu.RLock()
	out := make([]Torrent, 0, len(s.Torrents))
	for _, t := range s.Torrents {
		out = append(out, *deepCopyTorrent(&t))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return Deref(out[i].Hash) < Deref(out[j].Hash) })
	return out
}

// GetServerState returns the most recent server state.
// Pointer fields are deep-copied so the caller cannot mutate internal state.
func (s *SyncState) GetServerState() ServerState {
	s.mu.RLock()
	ss := *deepCopyServerState(&s.ServerState)
	s.mu.RUnlock()
	return ss
}

// GetCategories returns a copy of the current category map.
func (s *SyncState) GetCategories() map[string]Category {
	s.mu.RLock()
	out := make(map[string]Category, len(s.Categories))
	for k, v := range s.Categories {
		out[k] = v
	}
	s.mu.RUnlock()
	return out
}

// GetTags returns the current tag list.
func (s *SyncState) GetTags() []string {
	s.mu.RLock()
	out := make([]string, len(s.Tags))
	copy(out, s.Tags)
	s.mu.RUnlock()
	return out
}

// Len returns the number of tracked torrents without allocating.
func (s *SyncState) Len() int {
	s.mu.RLock()
	n := len(s.Torrents)
	s.mu.RUnlock()
	return n
}

// VisitTorrents calls fn for every torrent while holding the read lock.
// Each Torrent passed to fn is a deep copy: pointer fields are independently
// copied so fn cannot mutate internal state through them.
// Return false from fn to stop iteration early.
//
// Prefer this over GetTorrents when iterating large torrent sets (e.g. 200 k)
// because it avoids copying the entire map into a new data structure.
func (s *SyncState) VisitTorrents(fn func(hash string, t Torrent) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, v := range s.Torrents {
		if !fn(k, *deepCopyTorrent(&v)) {
			return
		}
	}
}

// VisitCategories calls fn for every category while holding the read lock.
// Return false from fn to stop iteration early.
func (s *SyncState) VisitCategories(fn func(name string, cat Category) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, v := range s.Categories {
		if !fn(k, v) {
			return
		}
	}
}

// apply merges a MainData update into the state.
func (s *SyncState) apply(d *MainData) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rid = d.Rid

	if d.FullUpdate {
		// Replace maps entirely on full updates.
		// The decoded maps are taken directly (O(1)) to avoid an O(n) copy.
		if d.Torrents != nil {
			s.Torrents = d.Torrents
		}
		if d.Categories != nil {
			s.Categories = d.Categories
		}
		if d.Tags != nil {
			s.Tags = d.Tags
		}
		if d.Trackers != nil {
			s.Trackers = d.Trackers
		}
	} else {
		// Partial update: merge diffs.
		for hash, torrent := range d.Torrents {
			if existing, ok := s.Torrents[hash]; ok {
				mergePartialTorrent(&existing, &torrent)
				s.Torrents[hash] = existing
			} else {
				s.Torrents[hash] = torrent
			}
		}
		for _, removed := range d.TorrentsRemoved {
			delete(s.Torrents, removed)
		}
		for name, cat := range d.Categories {
			s.Categories[name] = cat
		}
		for _, removed := range d.CategoriesRemoved {
			delete(s.Categories, removed)
		}
		if len(d.Tags) > 0 {
			s.Tags = append(s.Tags, d.Tags...)
		}
		for _, removed := range d.TagsRemoved {
			s.removeTag(removed)
		}
		for tracker, torrents := range d.Trackers {
			s.Trackers[tracker] = torrents
		}
		for _, removed := range d.TrackersRemoved {
			delete(s.Trackers, removed)
		}
	}

	// ServerState fields in partial updates only contain changed keys.
	mergePartialServerState(&s.ServerState, &d.ServerState)
}

func (s *SyncState) removeTag(tag string) {
	for i, t := range s.Tags {
		if t == tag {
			s.Tags = append(s.Tags[:i], s.Tags[i+1:]...)
			return
		}
	}
}

// SyncManager maintains a continuously-updated local view of qBittorrent state.
// It uses singleflight to coalesce concurrent sync requests and adapts its
// polling interval dynamically.
type SyncManager struct {
	client *Client
	opts   SyncOptions
	state  *SyncState
	sfg    singleflight.Group

	startOnce  sync.Once // ensures Start() is a no-op if called more than once
	cancelOnce sync.Once
	cancel     context.CancelFunc
	done       chan struct{}
}

// NewSyncManager creates a new SyncManager. Call Start() to begin polling.
func (c *Client) NewSyncManager(opts ...SyncOptions) *SyncManager {
	o := SyncOptions{}
	if len(opts) > 0 {
		o = opts[0]
	}
	o.setDefaults()
	return &SyncManager{
		client: c,
		opts:   o,
		state:  newSyncState(),
		done:   make(chan struct{}),
	}
}

// State returns read-only access to the live SyncState.
func (m *SyncManager) State() *SyncState { return m.state }

// Sync performs a single synchronisation cycle (thread-safe, deduplicated).
// Returns the updated state.
func (m *SyncManager) Sync(ctx context.Context) (*SyncState, error) {
	res, err, _ := m.sfg.Do("sync", func() (any, error) {
		return m.state, m.doSync(ctx)
	})
	if err != nil {
		return nil, err
	}
	return res.(*SyncState), nil
}

// doSync is the un-deduplicated implementation called directly by the background
// loop. Re-using this avoids allocating a singleflight closure on every tick.
func (m *SyncManager) doSync(ctx context.Context) error {
	m.state.mu.RLock()
	rid := m.state.rid
	m.state.mu.RUnlock()

	data, err := m.client.SyncMainData(ctx, rid)
	if err != nil {
		return err
	}
	m.state.apply(data)
	return nil
}

// Start begins background polling. Cancel the supplied context or call Stop to halt.
// Start is idempotent: subsequent calls are no-ops and the second context is ignored.
func (m *SyncManager) Start(ctx context.Context) {
	m.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(ctx)
		m.cancelOnce.Do(func() { m.cancel = cancel })
		go m.loop(ctx)
	})
}

// Stop halts background polling and blocks until the loop exits.
func (m *SyncManager) Stop() {
	m.cancelOnce.Do(func() {})
	if m.cancel != nil {
		m.cancel()
	}
	<-m.done
}

func (m *SyncManager) loop(ctx context.Context) {
	defer close(m.done)

	interval := m.opts.Interval
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Call doSync directly to avoid allocating a singleflight closure
			// on each tick. Concurrent calls from user code still go through
			// the deduplicated Sync() path.
			if err := m.doSync(ctx); err != nil {
				if m.opts.OnError != nil && m.opts.OnError(err) {
					return
				}
				// Back off on error.
				if m.opts.DynamicInterval {
					interval = clampDuration(interval*2, m.opts.MinInterval, m.opts.MaxInterval)
					ticker.Reset(interval)
				}
				continue
			}
			if m.opts.OnUpdate != nil {
				m.opts.OnUpdate(m.state)
			}
			// Shrink interval when activity is detected.
			if m.opts.DynamicInterval {
				interval = clampDuration(interval/2, m.opts.MinInterval, m.opts.MaxInterval)
				ticker.Reset(interval)
			}
		}
	}
}

func clampDuration(d, min, max time.Duration) time.Duration {
	if d < min {
		return min
	}
	if d > max {
		return max
	}
	return d
}
