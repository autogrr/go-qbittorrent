package qbittorrent

import (
	"net/url"
	"strconv"
)

// TorrentFilterOptions controls the torrents/info query.
type TorrentFilterOptions struct {
	Filter          TorrentFilter
	Category        string
	Tag             string
	Sort            string
	Reverse         bool
	Limit           int
	Offset          int
	Hashes          []string
	IncludeTrackers bool
}

// Encode returns the options as url.Values for GET requests.
func (o TorrentFilterOptions) Encode() url.Values {
	v := make(url.Values, 10)
	if o.Filter != "" {
		v.Set("filter", string(o.Filter))
	}
	if o.Category != "" {
		v.Set("category", o.Category)
	}
	if o.Tag != "" {
		v.Set("tag", o.Tag)
	}
	if o.Sort != "" {
		v.Set("sort", o.Sort)
	}
	if o.Reverse {
		v.Set("reverse", "true")
	}
	if o.Limit > 0 {
		v.Set("limit", strconv.Itoa(o.Limit))
	}
	if o.Offset > 0 {
		v.Set("offset", strconv.Itoa(o.Offset))
	}
	if len(o.Hashes) > 0 {
		v.Set("hashes", joinPipe(o.Hashes))
	}
	if o.IncludeTrackers {
		v.Set("includeTrackers", "true")
	}
	return v
}

// TorrentAddOptions controls how a torrent is added.
type TorrentAddOptions struct {
	// Destination paths
	SavePath     string
	DownloadPath string

	// Metadata
	Category string
	Tags     []string
	Rename   string

	// Behaviour
	SkipHashCheck bool
	Stopped       bool // prevent auto-start
	ContentLayout ContentLayout
	AutoTMM       *bool

	// Limits
	RatioLimit               float64
	SeedingTimeLimit         int64
	InactiveSeedingTimeLimit int64
	UploadLimit              int64
	DownloadLimit            int64

	// Piece / streaming
	FirstLastPiecePrio bool
	SequentialDownload bool
}

// Encode returns the options as query-string params for the add form.
func (o TorrentAddOptions) Encode() url.Values {
	v := make(url.Values, 16)
	if o.SavePath != "" {
		v.Set("savepath", o.SavePath)
	}
	if o.DownloadPath != "" {
		v.Set("downloadPath", o.DownloadPath)
	}
	if o.Category != "" {
		v.Set("category", o.Category)
	}
	if len(o.Tags) > 0 {
		v.Set("tags", joinComma(o.Tags))
	}
	if o.Rename != "" {
		v.Set("rename", o.Rename)
	}
	if o.SkipHashCheck {
		v.Set("skip_checking", "true")
	}
	if o.Stopped {
		v.Set("stopped", "true")
		v.Set("paused", "true") // compat with older API
	}
	if o.ContentLayout != "" {
		v.Set("contentLayout", string(o.ContentLayout))
	}
	if o.AutoTMM != nil {
		if *o.AutoTMM {
			v.Set("autoTMM", "true")
		} else {
			v.Set("autoTMM", "false")
		}
	}
	if o.RatioLimit != 0 {
		v.Set("ratioLimit", strconv.FormatFloat(o.RatioLimit, 'f', -1, 64))
	}
	if o.SeedingTimeLimit != 0 {
		v.Set("seedingTimeLimit", strconv.FormatInt(o.SeedingTimeLimit, 10))
	}
	if o.InactiveSeedingTimeLimit != 0 {
		v.Set("inactiveSeedingTimeLimit", strconv.FormatInt(o.InactiveSeedingTimeLimit, 10))
	}
	if o.UploadLimit > 0 {
		v.Set("uploadLimit", strconv.FormatInt(o.UploadLimit, 10))
	}
	if o.DownloadLimit > 0 {
		v.Set("downloadLimit", strconv.FormatInt(o.DownloadLimit, 10))
	}
	if o.FirstLastPiecePrio {
		v.Set("firstLastPiecePrio", "true")
	}
	if o.SequentialDownload {
		v.Set("sequentialDownload", "true")
	}
	return v
}

// LogOptions controls which log levels are fetched.
type LogOptions struct {
	Normal      bool
	Info        bool
	Warning     bool
	Critical    bool
	LastKnownID int
}

func (o LogOptions) Encode() url.Values {
	v := make(url.Values, 5)
	v.Set("normal", boolStr(o.Normal))
	v.Set("info", boolStr(o.Info))
	v.Set("warning", boolStr(o.Warning))
	v.Set("critical", boolStr(o.Critical))
	if o.LastKnownID > 0 {
		v.Set("last_known_id", strconv.Itoa(o.LastKnownID))
	}
	return v
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
