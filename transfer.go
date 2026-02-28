package qbittorrent

import (
	"context"
	"net/url"
	"strconv"
)

// GetTransferInfo returns global transfer statistics.
func (c *Client) GetTransferInfo(ctx context.Context) (*TransferInfo, error) {
	var info TransferInfo
	if err := c.get(ctx, "transfer", "info", nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// GetAlternativeSpeedLimitsMode returns 1 if alternative speed limits are active.
func (c *Client) GetAlternativeSpeedLimitsMode(ctx context.Context) (int, error) {
	var mode int
	if err := c.get(ctx, "transfer", "speedLimitsMode", nil, &mode); err != nil {
		return 0, err
	}
	return mode, nil
}

// ToggleAlternativeSpeedLimits toggles the alternative speed limits mode.
func (c *Client) ToggleAlternativeSpeedLimits(ctx context.Context) error {
	return c.post(ctx, "transfer", "toggleSpeedLimitsMode", url.Values{}, nil)
}

// GetGlobalDownloadLimit returns the global download speed limit in bytes/s.
// 0 means unlimited.
func (c *Client) GetGlobalDownloadLimit(ctx context.Context) (int64, error) {
	var limit int64
	if err := c.get(ctx, "transfer", "downloadLimit", nil, &limit); err != nil {
		return 0, err
	}
	return limit, nil
}

// SetGlobalDownloadLimit sets the global download speed limit in bytes/s.
// 0 disables the limit.
func (c *Client) SetGlobalDownloadLimit(ctx context.Context, limit int64) error {
	v := url.Values{"limit": {strconv.FormatInt(limit, 10)}}
	return c.post(ctx, "transfer", "setDownloadLimit", v, nil)
}

// GetGlobalUploadLimit returns the global upload speed limit in bytes/s.
func (c *Client) GetGlobalUploadLimit(ctx context.Context) (int64, error) {
	var limit int64
	if err := c.get(ctx, "transfer", "uploadLimit", nil, &limit); err != nil {
		return 0, err
	}
	return limit, nil
}

// SetGlobalUploadLimit sets the global upload speed limit in bytes/s.
func (c *Client) SetGlobalUploadLimit(ctx context.Context, limit int64) error {
	v := url.Values{"limit": {strconv.FormatInt(limit, 10)}}
	return c.post(ctx, "transfer", "setUploadLimit", v, nil)
}

// BanPeers bans the provided peers. Each peer must be "host:port".
func (c *Client) BanPeers(ctx context.Context, peers []string) error {
	v := url.Values{"peers": {joinPipe(peers)}}
	return c.post(ctx, "transfer", "banPeers", v, nil)
}
