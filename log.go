package qbittorrent

import (
	"context"
	"net/url"
	"strconv"
)

// GetLogs returns main log entries. Use LogOptions to filter by severity.
func (c *Client) GetLogs(ctx context.Context, opts LogOptions) ([]LogEntry, error) {
	var logs []LogEntry
	if err := c.get(ctx, "log", "main", opts.Encode(), &logs); err != nil {
		return nil, err
	}
	return logs, nil
}

// GetPeerLogs returns peer log entries.
// lastKnownID filters results to entries after that ID; use -1 for all.
func (c *Client) GetPeerLogs(ctx context.Context, lastKnownID int) ([]PeerLogEntry, error) {
	params := url.Values{}
	if lastKnownID >= 0 {
		params.Set("last_known_id", strconv.Itoa(lastKnownID))
	}
	var logs []PeerLogEntry
	if err := c.get(ctx, "log", "peers", params, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}
