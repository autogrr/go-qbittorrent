package qbittorrent

import (
	"context"
	"net/url"
	"strings"
)

// Login authenticates the client and stores the session cookie.
// This must be called before any other API methods (or use LoginCtx).
func (c *Client) Login(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loggedIn.Store(0)
	return c.loginLocked(ctx)
}

// Logout invalidates the current session.
func (c *Client) Logout(ctx context.Context) error {
	err := c.post(ctx, "auth", "logout", url.Values{}, nil)
	c.loggedIn.Store(0)
	return err
}

// GetAppVersion returns the qBittorrent application version string.
func (c *Client) GetAppVersion(ctx context.Context) (string, error) {
	return c.getString(ctx, "app", "version", nil)
}

// GetWebAPIVersion returns the qBittorrent WebUI API version string.
func (c *Client) GetWebAPIVersion(ctx context.Context) (string, error) {
	return c.getString(ctx, "app", "webapiVersion", nil)
}

// GetBuildInfo returns build metadata (Qt, libtorrent, boost, etc.).
func (c *Client) GetBuildInfo(ctx context.Context) (*BuildInfo, error) {
	var info BuildInfo
	if err := c.get(ctx, "app", "buildInfo", nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Shutdown triggers graceful shutdown of the qBittorrent application.
func (c *Client) Shutdown(ctx context.Context) error {
	return c.post(ctx, "app", "shutdown", url.Values{}, nil)
}

// GetPreferences returns the full application preferences object.
func (c *Client) GetPreferences(ctx context.Context) (*AppPreferences, error) {
	var prefs AppPreferences
	if err := c.get(ctx, "app", "preferences", nil, &prefs); err != nil {
		return nil, err
	}
	return &prefs, nil
}

// SetPreferences updates application preferences.
// Only the fields included in the map are changed; use the JSON field names
// from AppPreferences as keys.
func (c *Client) SetPreferences(ctx context.Context, prefs map[string]any) error {
	b, err := jsonMarshal(prefs)
	if err != nil {
		return err
	}
	v := url.Values{"json": {string(b)}}
	return c.post(ctx, "app", "setPreferences", v, nil)
}

// GetDefaultSavePath returns the default download save path.
func (c *Client) GetDefaultSavePath(ctx context.Context) (string, error) {
	s, err := c.getString(ctx, "app", "defaultSavePath", nil)
	return strings.TrimSpace(s), err
}

// GetCookies returns the list of stored browser-style cookies (qBit ≥ v2.11.2).
func (c *Client) GetCookies(ctx context.Context) ([]Cookie, error) {
	var cookies []Cookie
	if err := c.get(ctx, "app", "cookies", nil, &cookies); err != nil {
		return nil, err
	}
	return cookies, nil
}

// SetCookies sets browser-style cookies (qBit ≥ v2.11.2).
func (c *Client) SetCookies(ctx context.Context, cookies []Cookie) error {
	b, err := jsonMarshal(cookies)
	if err != nil {
		return err
	}
	v := url.Values{"cookies": {string(b)}}
	return c.post(ctx, "app", "setCookies", v, nil)
}

// --- Convenience preference setters ---

// SetQueueingEnabled enables or disables torrent queueing.
func (c *Client) SetQueueingEnabled(ctx context.Context, enabled bool) error {
	return c.SetPreferences(ctx, map[string]any{"queueing_enabled": enabled})
}

// SetMaxActiveDownloads sets the global maximum concurrent downloads.
func (c *Client) SetMaxActiveDownloads(ctx context.Context, max int) error {
	return c.SetPreferences(ctx, map[string]any{"max_active_downloads": max})
}

// SetMaxActiveTorrents sets the global maximum concurrent active torrents.
func (c *Client) SetMaxActiveTorrents(ctx context.Context, max int) error {
	return c.SetPreferences(ctx, map[string]any{"max_active_torrents": max})
}

// SetMaxActiveUploads sets the global maximum concurrent uploads.
func (c *Client) SetMaxActiveUploads(ctx context.Context, max int) error {
	return c.SetPreferences(ctx, map[string]any{"max_active_uploads": max})
}

// SetSubcategoriesEnabled enables or disables subcategories.
func (c *Client) SetSubcategoriesEnabled(ctx context.Context, enabled bool) error {
	return c.SetPreferences(ctx, map[string]any{"subcategories_enabled": enabled})
}

// SetRSSAutoDownloadingEnabled enables or disables RSS auto-downloading.
func (c *Client) SetRSSAutoDownloadingEnabled(ctx context.Context, enabled bool) error {
	return c.SetPreferences(ctx, map[string]any{"rss_auto_downloading_enabled": enabled})
}

// SetRSSProcessingEnabled enables or disables RSS processing.
func (c *Client) SetRSSProcessingEnabled(ctx context.Context, enabled bool) error {
	return c.SetPreferences(ctx, map[string]any{"rss_processing_enabled": enabled})
}
