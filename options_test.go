package qbittorrent

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---- TorrentFilterOptions ----

func TestTorrentFilterOptions_Empty(t *testing.T) {
	v := TorrentFilterOptions{}.Encode()
	assert.Empty(t, v.Get("filter"))
	assert.Empty(t, v.Get("category"))
	assert.Empty(t, v.Get("sort"))
	assert.Empty(t, v.Get("reverse"))
	assert.Empty(t, v.Get("limit"))
	assert.Empty(t, v.Get("offset"))
	assert.Empty(t, v.Get("hashes"))
}

func TestTorrentFilterOptions_Full(t *testing.T) {
	priv := true
	opts := TorrentFilterOptions{
		Filter:          FilterDownloading,
		Category:        "linux",
		Tag:             "open-source",
		Sort:            "name",
		Reverse:         true,
		Limit:           50,
		Offset:          10,
		Hashes:          []string{"abc123", "def456"},
		IncludeTrackers: true,
		IncludeFiles:    true,
		IsPrivate:       &priv,
	}
	v := opts.Encode()
	assert.Equal(t, "downloading", v.Get("filter"))
	assert.Equal(t, "linux", v.Get("category"))
	assert.Equal(t, "open-source", v.Get("tag"))
	assert.Equal(t, "name", v.Get("sort"))
	assert.Equal(t, "true", v.Get("reverse"))
	assert.Equal(t, "50", v.Get("limit"))
	assert.Equal(t, "10", v.Get("offset"))
	assert.Equal(t, "abc123|def456", v.Get("hashes"))
	assert.Equal(t, "true", v.Get("includeTrackers"))
	assert.Equal(t, "true", v.Get("includeFiles"))
	assert.Equal(t, "true", v.Get("private"))
}

func TestTorrentFilterOptions_IsPrivateFalse(t *testing.T) {
	priv := false
	v := TorrentFilterOptions{IsPrivate: &priv}.Encode()
	assert.Equal(t, "false", v.Get("private"))
}

func TestTorrentFilterOptions_IsPrivateNilOmitted(t *testing.T) {
	v := TorrentFilterOptions{}.Encode()
	assert.Empty(t, v.Get("private"))
}

func TestTorrentFilterOptions_IncludeFilesOnly(t *testing.T) {
	v := TorrentFilterOptions{IncludeFiles: true}.Encode()
	assert.Equal(t, "true", v.Get("includeFiles"))
	assert.Empty(t, v.Get("includeTrackers"))
}

func TestTorrentFilterOptions_NegativeLimitIgnored(t *testing.T) {
	v := TorrentFilterOptions{Limit: -1, Offset: -1}.Encode()
	assert.Empty(t, v.Get("limit"))
	assert.Empty(t, v.Get("offset"))
}

// ---- TorrentAddOptions ----

func TestTorrentAddOptions_Empty(t *testing.T) {
	v := TorrentAddOptions{}.Encode()
	assert.Empty(t, v.Get("savepath"))
	assert.Empty(t, v.Get("category"))
	assert.Empty(t, v.Get("stopped"))
	assert.Empty(t, v.Get("paused"))
}

func TestTorrentAddOptions_Full(t *testing.T) {
	autoTMM := true
	opts := TorrentAddOptions{
		SavePath:                 "/data/downloads",
		DownloadPath:             "/data/incomplete",
		Category:                 "movies",
		Tags:                     []string{"hdr", "remux"},
		Rename:                   "final.mkv",
		SkipHashCheck:            true,
		Stopped:                  true,
		ContentLayout:            ContentLayoutSubfolderNone,
		AutoTMM:                  &autoTMM,
		RatioLimit:               1.5,
		SeedingTimeLimit:         3600,
		InactiveSeedingTimeLimit: 7200,
		UploadLimit:              1024 * 1024,
		DownloadLimit:            2 * 1024 * 1024,
		FirstLastPiecePrio:       true,
		SequentialDownload:       true,
	}
	v := opts.Encode()
	assert.Equal(t, "/data/downloads", v.Get("savepath"))
	assert.Equal(t, "/data/incomplete", v.Get("downloadPath"))
	assert.Equal(t, "movies", v.Get("category"))
	assert.Equal(t, "hdr,remux", v.Get("tags"))
	assert.Equal(t, "final.mkv", v.Get("rename"))
	assert.Equal(t, "true", v.Get("skip_checking"))
	assert.Equal(t, "true", v.Get("stopped"))
	assert.Equal(t, "true", v.Get("paused"))
	assert.Equal(t, "NoSubfolder", v.Get("contentLayout"))
	assert.Equal(t, "true", v.Get("autoTMM"))
	assert.Equal(t, "1.5", v.Get("ratioLimit"))
	assert.Equal(t, "3600", v.Get("seedingTimeLimit"))
	assert.Equal(t, "7200", v.Get("inactiveSeedingTimeLimit"))
	assert.Equal(t, "1048576", v.Get("uploadLimit"))
	assert.Equal(t, "2097152", v.Get("downloadLimit"))
	assert.Equal(t, "true", v.Get("firstLastPiecePrio"))
	assert.Equal(t, "true", v.Get("sequentialDownload"))
}

func TestTorrentAddOptions_AutoTMMFalse(t *testing.T) {
	f := false
	v := TorrentAddOptions{AutoTMM: &f}.Encode()
	assert.Equal(t, "false", v.Get("autoTMM"))
}

func TestTorrentAddOptions_AutoTMMNil(t *testing.T) {
	v := TorrentAddOptions{}.Encode()
	assert.Empty(t, v.Get("autoTMM"))
}

// ---- LogOptions ----

func TestLogOptions_Encode(t *testing.T) {
	v := LogOptions{Normal: true, Info: true, Warning: false, Critical: true, LastKnownID: 42}.Encode()
	assert.Equal(t, "true", v.Get("normal"))
	assert.Equal(t, "true", v.Get("info"))
	assert.Equal(t, "false", v.Get("warning"))
	assert.Equal(t, "true", v.Get("critical"))
	assert.Equal(t, "42", v.Get("last_known_id"))
}

func TestLogOptions_NoLastKnownID(t *testing.T) {
	v := LogOptions{Normal: true}.Encode()
	_, hasID := v[string("last_known_id")]
	_ = hasID
	// LastKnownID=0 should NOT add the param.
	assert.Empty(t, v.Get("last_known_id"))
}

// ---- joinHashes ----

func TestJoinHashes_Empty(t *testing.T) {
	assert.Equal(t, "all", joinHashes(nil))
	assert.Equal(t, "all", joinHashes([]string{}))
}

func TestJoinHashes_Single(t *testing.T) {
	assert.Equal(t, "abc", joinHashes([]string{"abc"}))
}

func TestJoinHashes_Multiple(t *testing.T) {
	assert.Equal(t, "a|b|c", joinHashes([]string{"a", "b", "c"}))
}

// ---- boolStr ----

func TestBoolStr(t *testing.T) {
	assert.Equal(t, "true", boolStr(true))
	assert.Equal(t, "false", boolStr(false))
}

// ---- url.Values round-trip ----

func TestFilterOptions_URLEncoding(t *testing.T) {
	opts := TorrentFilterOptions{Category: "my category", Tag: "a b"}
	v := opts.Encode()
	encoded := v.Encode()
	parsed, err := url.ParseQuery(encoded)
	assert.NoError(t, err)
	assert.Equal(t, "my category", parsed.Get("category"))
	assert.Equal(t, "a b", parsed.Get("tag"))
}

// ---- join helpers ----

func TestJoinPipe(t *testing.T) {
	assert.Equal(t, "", joinPipe(nil))
	assert.Equal(t, "a", joinPipe([]string{"a"}))
	assert.Equal(t, "a|b|c", joinPipe([]string{"a", "b", "c"}))
}

func TestJoinNewline(t *testing.T) {
	assert.Equal(t, "", joinNewline(nil))
	assert.Equal(t, "a", joinNewline([]string{"a"}))
	assert.Equal(t, "a\nb\nc", joinNewline([]string{"a", "b", "c"}))
}

func TestJoinComma(t *testing.T) {
	assert.Equal(t, "", joinComma(nil))
	assert.Equal(t, "a", joinComma([]string{"a"}))
	assert.Equal(t, "a,b,c", joinComma([]string{"a", "b", "c"}))
}

func TestJoinSep_ExactAlloc(t *testing.T) {
	// joinSep pre-computes length exactly — output must equal naive join.
	ss := []string{"hello", "world", "foo"}
	assert.Equal(t, "hello|world|foo", joinSep(ss, '|'))
	assert.Equal(t, "hello world foo", joinSep(ss, ' '))
}
