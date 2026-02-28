package qbittorrent

import (
	"context"
	"net/url"
	"strconv"
)

// SyncMainData fetches a partial or full maindata update.
// Pass rid=0 to request a full update. Every subsequent call should use
// the Rid value from the previous response to receive incremental updates.
func (c *Client) SyncMainData(ctx context.Context, rid int) (*MainData, error) {
	params := url.Values{"rid": {strconv.Itoa(rid)}}
	var data MainData
	if err := c.get(ctx, "sync", "maindata", params, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// SyncMainDataRaw returns the raw JSON bytes from sync/maindata.
// Use this when you want to parse partially with your own logic.
func (c *Client) SyncMainDataRaw(ctx context.Context, rid int) ([]byte, error) {
	params := url.Values{"rid": {strconv.Itoa(rid)}}
	return c.getRaw(ctx, "sync", "maindata", params)
}

// SyncTorrentPeers returns peer sync data for a specific torrent.
func (c *Client) SyncTorrentPeers(ctx context.Context, hash string, rid int) (*TorrentPeersResponse, error) {
	params := url.Values{
		"hash": {hash},
		"rid":  {strconv.Itoa(rid)},
	}
	var peers TorrentPeersResponse
	if err := c.get(ctx, "sync", "torrentPeers", params, &peers); err != nil {
		return nil, err
	}
	return &peers, nil
}
