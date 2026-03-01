package qbittorrent

import "unique"

// internStr returns a canonicalised copy of s using the stdlib unique package
// (available since Go 1.23).  Repeated calls with equal strings return a value
// backed by the same underlying memory, reducing allocations when many Torrent
// copies share low-cardinality fields such as Category, SavePath, Name, State
// and Tracker.
//
// The unique package uses a runtime-managed weak-reference table so entries
// are automatically collected when no Handle remains live — there is no
// unbounded growth from torrents that are later removed.
func internStr(s string) string {
	return unique.Make(s).Value()
}
