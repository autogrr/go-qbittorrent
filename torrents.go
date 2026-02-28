package qbittorrent

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/url"
	"strconv"
	"strings"
)

// ---- Torrent listing ----

// GetTorrents returns the list of torrents matching the given filter options.
func (c *Client) GetTorrents(ctx context.Context, opts TorrentFilterOptions) ([]Torrent, error) {
	var torrents []Torrent
	if err := c.get(ctx, "torrents", "info", opts.Encode(), &torrents); err != nil {
		return nil, err
	}
	return torrents, nil
}

// GetTorrentsRaw returns the raw JSON bytes from the torrents/info endpoint.
// Useful when you want to parse the response yourself (e.g. with a faster custom parser).
func (c *Client) GetTorrentsRaw(ctx context.Context, opts TorrentFilterOptions) ([]byte, error) {
	return c.getRaw(ctx, "torrents", "info?"+opts.Encode().Encode(), nil)
}

// GetTorrentProperties returns detailed properties for a single torrent.
func (c *Client) GetTorrentProperties(ctx context.Context, hash string) (*TorrentProperties, error) {
	var props TorrentProperties
	if err := c.get(ctx, "torrents", "properties", url.Values{"hash": {hash}}, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

// GetTorrentTrackers returns the trackers for a torrent.
func (c *Client) GetTorrentTrackers(ctx context.Context, hash string) ([]TorrentTracker, error) {
	var trackers []TorrentTracker
	if err := c.get(ctx, "torrents", "trackers", url.Values{"hash": {hash}}, &trackers); err != nil {
		return nil, err
	}
	return trackers, nil
}

// GetTorrentWebSeeds returns the web seeds for a torrent.
func (c *Client) GetTorrentWebSeeds(ctx context.Context, hash string) ([]WebSeed, error) {
	var seeds []WebSeed
	if err := c.get(ctx, "torrents", "webseeds", url.Values{"hash": {hash}}, &seeds); err != nil {
		return nil, err
	}
	return seeds, nil
}

// GetTorrentFiles returns the file list for a torrent.
// indexes is optional; when non-nil it restricts to those file indices.
func (c *Client) GetTorrentFiles(ctx context.Context, hash string, indexes []int) ([]TorrentFile, error) {
	params := url.Values{"hash": {hash}}
	if len(indexes) > 0 {
		parts := make([]string, len(indexes))
		for i, idx := range indexes {
			parts[i] = strconv.Itoa(idx)
		}
		params.Set("indexes", joinPipe(parts))
	}
	var files []TorrentFile
	if err := c.get(ctx, "torrents", "files", params, &files); err != nil {
		return nil, err
	}
	return files, nil
}

// GetTorrentPieceStates returns pieces states (0=not downloaded, 1=downloading, 2=downloaded).
func (c *Client) GetTorrentPieceStates(ctx context.Context, hash string) ([]int, error) {
	var states []int
	if err := c.get(ctx, "torrents", "pieceStates", url.Values{"hash": {hash}}, &states); err != nil {
		return nil, err
	}
	return states, nil
}

// GetTorrentPieceHashes returns the SHA1 hashes of each piece.
func (c *Client) GetTorrentPieceHashes(ctx context.Context, hash string) ([]string, error) {
	var hashes []string
	if err := c.get(ctx, "torrents", "pieceHashes", url.Values{"hash": {hash}}, &hashes); err != nil {
		return nil, err
	}
	return hashes, nil
}

// ExportTorrent downloads the .torrent file for the given hash.
// The caller is responsible for closing the returned io.ReadCloser.
func (c *Client) ExportTorrent(ctx context.Context, hash string) ([]byte, error) {
	return c.getRaw(ctx, "torrents", "export", url.Values{"hash": {hash}})
}

// ---- Torrent state control ----

// Pause pauses (stops) the given torrents. Use "all" or empty for all torrents.
func (c *Client) Pause(ctx context.Context, hashes []string) error {
	v := url.Values{"hashes": {joinHashes(hashes)}}
	// Try the newer API name first, fall back to legacy.
	err := c.post(ctx, "torrents", "stop", v, nil)
	if isNotFound(err) {
		err = c.post(ctx, "torrents", "pause", v, nil)
	}
	return err
}

// Resume resumes the given torrents.
func (c *Client) Resume(ctx context.Context, hashes []string) error {
	v := url.Values{"hashes": {joinHashes(hashes)}}
	err := c.post(ctx, "torrents", "start", v, nil)
	if isNotFound(err) {
		err = c.post(ctx, "torrents", "resume", v, nil)
	}
	return err
}

// Delete removes the given torrents. Set deleteFiles=true to also remove data.
func (c *Client) Delete(ctx context.Context, hashes []string, deleteFiles bool) error {
	v := url.Values{
		"hashes":      {joinHashes(hashes)},
		"deleteFiles": {boolStr(deleteFiles)},
	}
	return c.post(ctx, "torrents", "delete", v, nil)
}

// Recheck rechecks (force re-hash) the given torrents.
func (c *Client) Recheck(ctx context.Context, hashes []string) error {
	return c.post(ctx, "torrents", "recheck", url.Values{"hashes": {joinHashes(hashes)}}, nil)
}

// Reannounce forces immediate tracker re-announce for the given torrents.
func (c *Client) Reannounce(ctx context.Context, hashes []string) error {
	return c.post(ctx, "torrents", "reannounce", url.Values{"hashes": {joinHashes(hashes)}}, nil)
}

// ---- Adding torrents ----

// AddTorrentFromURLs adds one or more torrents by magnet link or HTTP URL.
func (c *Client) AddTorrentFromURLs(ctx context.Context, urls []string, opts TorrentAddOptions) error {
	v := opts.Encode()
	v.Set("urls", joinNewline(urls))
	res, err := c.postGetString(ctx, "torrents", "add", v)
	if err != nil {
		return err
	}
	if strings.Contains(res, "Fails") {
		return ErrAlreadyExists
	}
	return nil
}

// AddTorrentFromURL adds a single torrent by URL or magnet link.
func (c *Client) AddTorrentFromURL(ctx context.Context, rawURL string, opts TorrentAddOptions) error {
	v := opts.Encode()
	v.Set("urls", rawURL)
	res, err := c.postGetString(ctx, "torrents", "add", v)
	if err != nil {
		return err
	}
	if strings.Contains(res, "Fails") {
		return ErrAlreadyExists
	}
	return nil
}

// AddTorrentFromReader adds a torrent from an io.Reader (e.g. a file handle).
// filename is used as the multipart filename; it can be any value ending in ".torrent".
func (c *Client) AddTorrentFromReader(ctx context.Context, filename string, r io.Reader, opts TorrentAddOptions) error {
	extra := opts.Encode()
	res, err := c.postMultipartGetString(ctx, "torrents", "add", extra, "torrents", filename, r)
	if err != nil {
		return err
	}
	if strings.Contains(res, "Fails") {
		return ErrAlreadyExists
	}
	return nil
}

// AddTorrentFromBytes adds a torrent from a raw .torrent byte slice.
func (c *Client) AddTorrentFromBytes(ctx context.Context, filename string, data []byte, opts TorrentAddOptions) error {
	return c.AddTorrentFromReader(ctx, filename, strings.NewReader(string(data)), opts)
}

// ---- Tracker management ----

// AddTrackers adds new tracker URLs (newline-separated) to a torrent.
func (c *Client) AddTrackers(ctx context.Context, hash string, urls []string) error {
	v := url.Values{
		"hash": {hash},
		"urls": {joinNewline(urls)},
	}
	return c.post(ctx, "torrents", "addTrackers", v, nil)
}

// EditTracker replaces the old tracker URL with a new one.
func (c *Client) EditTracker(ctx context.Context, hash, origURL, newURL string) error {
	v := url.Values{
		"hash":    {hash},
		"origUrl": {origURL},
		"newUrl":  {newURL},
	}
	return c.post(ctx, "torrents", "editTracker", v, nil)
}

// RemoveTrackers removes tracker URLs from a torrent.
func (c *Client) RemoveTrackers(ctx context.Context, hash string, urls []string) error {
	v := url.Values{
		"hash": {hash},
		"urls": {joinPipe(urls)},
	}
	return c.post(ctx, "torrents", "removeTrackers", v, nil)
}

// ---- Peer management ----

// AddPeers adds peers ("host:port") to torrents.
func (c *Client) AddPeers(ctx context.Context, hashes []string, peers []string) error {
	v := url.Values{
		"hashes": {joinPipe(hashes)},
		"peers":  {joinPipe(peers)},
	}
	return c.post(ctx, "torrents", "addPeers", v, nil)
}

// ---- Priority ----

// SetMaxPriority sets the max download priority for the given torrents.
func (c *Client) SetMaxPriority(ctx context.Context, hashes []string) error {
	return c.post(ctx, "torrents", "topPrio", url.Values{"hashes": {joinHashes(hashes)}}, nil)
}

// SetMinPriority sets the minimum download priority for the given torrents.
func (c *Client) SetMinPriority(ctx context.Context, hashes []string) error {
	return c.post(ctx, "torrents", "bottomPrio", url.Values{"hashes": {joinHashes(hashes)}}, nil)
}

// IncreasePriority increases the priority of the given torrents.
func (c *Client) IncreasePriority(ctx context.Context, hashes []string) error {
	return c.post(ctx, "torrents", "increasePrio", url.Values{"hashes": {joinHashes(hashes)}}, nil)
}

// DecreasePriority decreases the priority of the given torrents.
func (c *Client) DecreasePriority(ctx context.Context, hashes []string) error {
	return c.post(ctx, "torrents", "decreasePrio", url.Values{"hashes": {joinHashes(hashes)}}, nil)
}

// SetFilePriority sets the download priority for specific files within a torrent.
func (c *Client) SetFilePriority(ctx context.Context, hash string, fileIDs []int, priority FilePriority) error {
	ids := make([]string, len(fileIDs))
	for i, id := range fileIDs {
		ids[i] = strconv.Itoa(id)
	}
	v := url.Values{
		"hash":     {hash},
		"id":       {joinPipe(ids)},
		"priority": {strconv.Itoa(int(priority))},
	}
	return c.post(ctx, "torrents", "filePrio", v, nil)
}

// ---- Speed limits ----

// GetTorrentDownloadLimits returns per-torrent download limits (hash → bytes/s).
func (c *Client) GetTorrentDownloadLimits(ctx context.Context, hashes []string) (map[string]int64, error) {
	v := url.Values{"hashes": {joinHashes(hashes)}}
	var result map[string]int64
	if err := c.post(ctx, "torrents", "downloadLimit", v, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SetTorrentDownloadLimit sets the download speed limit for the given torrents.
func (c *Client) SetTorrentDownloadLimit(ctx context.Context, hashes []string, limit int64) error {
	v := url.Values{
		"hashes": {joinHashes(hashes)},
		"limit":  {strconv.FormatInt(limit, 10)},
	}
	return c.post(ctx, "torrents", "setDownloadLimit", v, nil)
}

// GetTorrentUploadLimits returns per-torrent upload limits (hash → bytes/s).
func (c *Client) GetTorrentUploadLimits(ctx context.Context, hashes []string) (map[string]int64, error) {
	v := url.Values{"hashes": {joinHashes(hashes)}}
	var result map[string]int64
	if err := c.post(ctx, "torrents", "uploadLimit", v, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SetTorrentUploadLimit sets the upload speed limit for the given torrents.
func (c *Client) SetTorrentUploadLimit(ctx context.Context, hashes []string, limit int64) error {
	v := url.Values{
		"hashes": {joinHashes(hashes)},
		"limit":  {strconv.FormatInt(limit, 10)},
	}
	return c.post(ctx, "torrents", "setUploadLimit", v, nil)
}

// SetTorrentShareLimits sets ratio/seeding-time limits for the given torrents.
// pass -1 for any limit to use global settings, -2 to disable.
func (c *Client) SetTorrentShareLimits(ctx context.Context, hashes []string, ratioLimit float64, seedingTimeLimit, inactiveSeedingTimeLimit int64) error {
	v := url.Values{
		"hashes":                   {joinHashes(hashes)},
		"ratioLimit":               {strconv.FormatFloat(ratioLimit, 'f', -1, 64)},
		"seedingTimeLimit":         {strconv.FormatInt(seedingTimeLimit, 10)},
		"inactiveSeedingTimeLimit": {strconv.FormatInt(inactiveSeedingTimeLimit, 10)},
	}
	return c.post(ctx, "torrents", "setShareLimits", v, nil)
}

// ---- Metadata ----

// SetTorrentLocation sets the save path for the given torrents.
func (c *Client) SetTorrentLocation(ctx context.Context, hashes []string, location string) error {
	v := url.Values{
		"hashes":   {joinHashes(hashes)},
		"location": {location},
	}
	return c.post(ctx, "torrents", "setLocation", v, nil)
}

// RenameTorrent renames a torrent.
func (c *Client) RenameTorrent(ctx context.Context, hash, name string) error {
	v := url.Values{"hash": {hash}, "name": {name}}
	return c.post(ctx, "torrents", "rename", v, nil)
}

// SetTorrentCategory assigns a category to the given torrents.
func (c *Client) SetTorrentCategory(ctx context.Context, hashes []string, category string) error {
	v := url.Values{"hashes": {joinHashes(hashes)}, "category": {category}}
	return c.post(ctx, "torrents", "setCategory", v, nil)
}

// ---- Categories ----

// GetCategories returns all categories (name → Category).
func (c *Client) GetCategories(ctx context.Context) (map[string]Category, error) {
	var categories map[string]Category
	if err := c.get(ctx, "torrents", "categories", nil, &categories); err != nil {
		return nil, err
	}
	return categories, nil
}

// CreateCategory creates a new category with an optional save path.
func (c *Client) CreateCategory(ctx context.Context, name, savePath string) error {
	v := url.Values{"category": {name}, "savePath": {savePath}}
	return c.post(ctx, "torrents", "createCategory", v, nil)
}

// EditCategory changes the save path for an existing category.
func (c *Client) EditCategory(ctx context.Context, name, savePath string) error {
	v := url.Values{"category": {name}, "savePath": {savePath}}
	return c.post(ctx, "torrents", "editCategory", v, nil)
}

// RemoveCategories deletes categories by name.
func (c *Client) RemoveCategories(ctx context.Context, names []string) error {
	v := url.Values{"categories": {joinNewline(names)}}
	return c.post(ctx, "torrents", "removeCategories", v, nil)
}

// ---- Tags ----

// GetTags returns all tags.
func (c *Client) GetTags(ctx context.Context) ([]string, error) {
	var tags []string
	if err := c.get(ctx, "torrents", "tags", nil, &tags); err != nil {
		return nil, err
	}
	return tags, nil
}

// CreateTags creates one or more tags.
func (c *Client) CreateTags(ctx context.Context, tags []string) error {
	v := url.Values{"tags": {joinComma(tags)}}
	return c.post(ctx, "torrents", "createTags", v, nil)
}

// AddTags adds tags to the given torrents.
func (c *Client) AddTags(ctx context.Context, hashes []string, tags []string) error {
	v := url.Values{
		"hashes": {joinHashes(hashes)},
		"tags":   {joinComma(tags)},
	}
	return c.post(ctx, "torrents", "addTags", v, nil)
}

// RemoveTags removes tags from the given torrents.
func (c *Client) RemoveTags(ctx context.Context, hashes []string, tags []string) error {
	v := url.Values{
		"hashes": {joinHashes(hashes)},
		"tags":   {joinComma(tags)},
	}
	return c.post(ctx, "torrents", "removeTags", v, nil)
}

// DeleteTags deletes tags globally.
func (c *Client) DeleteTags(ctx context.Context, tags []string) error {
	v := url.Values{"tags": {joinComma(tags)}}
	return c.post(ctx, "torrents", "deleteTags", v, nil)
}

// SetTags replaces all tags on the given torrents (qBit ≥ v2.11.4 / qBit ≥ 5.0.3).
func (c *Client) SetTags(ctx context.Context, hashes []string, tags []string) error {
	v := url.Values{
		"hashes": {joinHashes(hashes)},
		"tags":   {joinComma(tags)},
	}
	return c.post(ctx, "torrents", "setTags", v, nil)
}

// ---- Behaviour flags ----

// SetAutoManagement enables/disables automatic torrent management for torrents.
func (c *Client) SetAutoManagement(ctx context.Context, hashes []string, enable bool) error {
	v := url.Values{
		"hashes": {joinHashes(hashes)},
		"enable": {boolStr(enable)},
	}
	return c.post(ctx, "torrents", "setAutoManagement", v, nil)
}

// SetForceStart forces the torrents to start immediately.
func (c *Client) SetForceStart(ctx context.Context, hashes []string, value bool) error {
	v := url.Values{
		"hashes": {joinHashes(hashes)},
		"value":  {boolStr(value)},
	}
	return c.post(ctx, "torrents", "setForceStart", v, nil)
}

// SetSuperSeeding enables/disables super-seeding mode.
func (c *Client) SetSuperSeeding(ctx context.Context, hashes []string, value bool) error {
	v := url.Values{
		"hashes": {joinHashes(hashes)},
		"value":  {boolStr(value)},
	}
	return c.post(ctx, "torrents", "setSuperSeeding", v, nil)
}

// ToggleSequentialDownload toggles sequential downloading.
func (c *Client) ToggleSequentialDownload(ctx context.Context, hashes []string) error {
	v := url.Values{"hashes": {joinHashes(hashes)}}
	return c.post(ctx, "torrents", "toggleSequentialDownload", v, nil)
}

// ToggleFirstLastPiecePrio toggles first/last piece priority.
func (c *Client) ToggleFirstLastPiecePrio(ctx context.Context, hashes []string) error {
	v := url.Values{"hashes": {joinHashes(hashes)}}
	return c.post(ctx, "torrents", "toggleFirstLastPiecePrio", v, nil)
}

// ---- Renaming ----

// RenameFile renames a file within a torrent.
func (c *Client) RenameFile(ctx context.Context, hash, oldPath, newPath string) error {
	v := url.Values{
		"hash":    {hash},
		"oldPath": {oldPath},
		"newPath": {newPath},
	}
	return c.post(ctx, "torrents", "renameFile", v, nil)
}

// RenameFolder renames a folder within a torrent.
func (c *Client) RenameFolder(ctx context.Context, hash, oldPath, newPath string) error {
	v := url.Values{
		"hash":    {hash},
		"oldPath": {oldPath},
		"newPath": {newPath},
	}
	return c.post(ctx, "torrents", "renameFolder", v, nil)
}

// ---- Torrent Creator (qBit ≥ 5.0 / WebAPI ≥ 2.11.2) ----

// CreateTorrent starts a .torrent creation task.
func (c *Client) CreateTorrent(ctx context.Context, params TorrentCreationParams) (int, error) {
	b, err := jsonMarshal(params)
	if err != nil {
		return 0, err
	}
	v := url.Values{"taskParams": {string(b)}}
	var result struct {
		TaskID int `json:"taskID"`
	}
	if err := c.post(ctx, "torrentcreator", "addTask", v, &result); err != nil {
		return 0, err
	}
	return result.TaskID, nil
}

// GetTorrentCreationStatus returns the status of a torrent creation task.
func (c *Client) GetTorrentCreationStatus(ctx context.Context, taskID int) (*TorrentCreationStatus, error) {
	params := url.Values{"taskID": {strconv.Itoa(taskID)}}
	var status TorrentCreationStatus
	if err := c.get(ctx, "torrentcreator", "status", params, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// GetCreatedTorrentFile downloads the created .torrent file for a finished task.
func (c *Client) GetCreatedTorrentFile(ctx context.Context, taskID int) ([]byte, error) {
	params := url.Values{"taskID": {strconv.Itoa(taskID)}}
	return c.getRaw(ctx, "torrentcreator", "torrentFile", params)
}

// DeleteTorrentCreationTask deletes a torrent creation task.
func (c *Client) DeleteTorrentCreationTask(ctx context.Context, taskID int) error {
	v := url.Values{"taskID": {strconv.Itoa(taskID)}}
	return c.post(ctx, "torrentcreator", "deleteTask", v, nil)
}

// ---- Helpers ----

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if ae, ok := err.(*APIError); ok {
		return ae.StatusCode == 404
	}
	return false
}

// postGetString sends a POST and returns the trimmed response body as a string.
func (c *Client) postGetString(ctx context.Context, scope, method string, form url.Values) (string, error) {
	encoded := form.Encode()
	req, err := c.newRequest(ctx, "POST", c.url(scope, method), strings.NewReader(encoded))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.do(req)
	if err != nil {
		return "", err
	}
	return consumeString(resp)
}

// postMultipartGetString sends a multipart POST and returns the body string.
// The multipart body is written synchronously into a pooled buffer — no
// goroutine or io.Pipe is needed.
func (c *Client) postMultipartGetString(ctx context.Context, scope, method string, extra url.Values, fieldName, filename string, r io.Reader) (string, error) {
	buf := getBuf()
	mw := multipart.NewWriter(buf)

	for key, vals := range extra {
		for _, val := range vals {
			if err := mw.WriteField(key, val); err != nil {
				putBuf(buf)
				return "", err
			}
		}
	}
	part, err := mw.CreateFormFile(fieldName, filename)
	if err != nil {
		putBuf(buf)
		return "", err
	}
	if _, err = io.Copy(part, r); err != nil {
		putBuf(buf)
		return "", err
	}
	if err = mw.Close(); err != nil {
		putBuf(buf)
		return "", err
	}

	contentType := mw.FormDataContentType()
	req, err := c.newRequest(ctx, "POST", c.url(scope, method), bytes.NewReader(buf.Bytes()))
	if err != nil {
		putBuf(buf)
		return "", err
	}
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = int64(buf.Len())

	resp, rerr := c.do(req)
	putBuf(buf)
	if rerr != nil {
		return "", rerr
	}
	return consumeString(resp)
}
