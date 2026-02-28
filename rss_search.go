package qbittorrent

import (
	"context"
	"net/url"
	"strconv"

	"github.com/bytedance/sonic"
)

// ---- RSS Feeds ----

// AddRSSFolder creates an RSS folder at the given path (e.g. "Linux/Arch").
func (c *Client) AddRSSFolder(ctx context.Context, path string) error {
	return c.post(ctx, "rss", "addFolder", url.Values{"path": {path}}, nil)
}

// AddRSSFeed subscribes to an RSS feed URL, optionally placing it at path.
func (c *Client) AddRSSFeed(ctx context.Context, feedURL, path string) error {
	v := url.Values{"url": {feedURL}}
	if path != "" {
		v.Set("path", path)
	}
	return c.post(ctx, "rss", "addFeed", v, nil)
}

// SetRSSFeedURL changes the URL of an existing RSS feed (qBit ≥ v2.9.1).
func (c *Client) SetRSSFeedURL(ctx context.Context, path, feedURL string) error {
	v := url.Values{"path": {path}, "url": {feedURL}}
	return c.post(ctx, "rss", "setFeedURL", v, nil)
}

// RemoveRSSItem removes an RSS feed or folder by path.
func (c *Client) RemoveRSSItem(ctx context.Context, path string) error {
	return c.post(ctx, "rss", "removeItem", url.Values{"path": {path}}, nil)
}

// MoveRSSItem moves or renames an RSS item.
func (c *Client) MoveRSSItem(ctx context.Context, itemPath, destPath string) error {
	v := url.Values{"itemPath": {itemPath}, "destPath": {destPath}}
	return c.post(ctx, "rss", "moveItem", v, nil)
}

// GetRSSItems returns the hierarchical RSS feed/article tree.
// Set withData=true to include downloaded article data.
func (c *Client) GetRSSItems(ctx context.Context, withData bool) (map[string]any, error) {
	params := url.Values{"withData": {boolStr(withData)}}
	var items map[string]any
	if err := c.get(ctx, "rss", "items", params, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// MarkRSSItemAsRead marks an RSS article as read.
// Set articleID to empty string to mark all articles in the feed as read.
func (c *Client) MarkRSSItemAsRead(ctx context.Context, itemPath, articleID string) error {
	v := url.Values{"itemPath": {itemPath}}
	if articleID != "" {
		v.Set("articleId", articleID)
	}
	return c.post(ctx, "rss", "markAsRead", v, nil)
}

// RefreshRSSItem triggers an immediate refresh of an RSS feed or folder.
func (c *Client) RefreshRSSItem(ctx context.Context, itemPath string) error {
	return c.post(ctx, "rss", "refreshItem", url.Values{"itemPath": {itemPath}}, nil)
}

// ---- RSS Rules ----

// GetRSSRules returns all RSS auto-download rules.
func (c *Client) GetRSSRules(ctx context.Context) (map[string]RSSAutoDownloadRule, error) {
	var rules map[string]RSSAutoDownloadRule
	if err := c.get(ctx, "rss", "rules", nil, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

// SetRSSRule creates or updates an RSS auto-download rule.
func (c *Client) SetRSSRule(ctx context.Context, name string, rule RSSAutoDownloadRule) error {
	b, err := sonic.Marshal(rule)
	if err != nil {
		return err
	}
	v := url.Values{"ruleName": {name}, "ruleDef": {string(b)}}
	return c.post(ctx, "rss", "setRule", v, nil)
}

// RenameRSSRule renames an existing RSS rule.
func (c *Client) RenameRSSRule(ctx context.Context, ruleName, newRuleName string) error {
	v := url.Values{"ruleName": {ruleName}, "newRuleName": {newRuleName}}
	return c.post(ctx, "rss", "renameRule", v, nil)
}

// RemoveRSSRule deletes an RSS auto-download rule.
func (c *Client) RemoveRSSRule(ctx context.Context, ruleName string) error {
	return c.post(ctx, "rss", "removeRule", url.Values{"ruleName": {ruleName}}, nil)
}

// GetRSSMatchingArticles returns articles that match a named rule.
// Returns a map of feed name → []article title.
func (c *Client) GetRSSMatchingArticles(ctx context.Context, ruleName string) (map[string][]string, error) {
	params := url.Values{"ruleName": {ruleName}}
	var result map[string][]string
	if err := c.get(ctx, "rss", "matchingArticles", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ---- Search ----

// StartSearch begins a search and returns the job ID.
// plugins should be "all" or a pipe-separated list of plugin names.
// category should be "all" or a specific category string.
func (c *Client) StartSearch(ctx context.Context, pattern, plugins, category string) (int, error) {
	v := url.Values{
		"pattern":  {pattern},
		"plugins":  {plugins},
		"category": {category},
	}
	var job SearchJob
	if err := c.post(ctx, "search", "start", v, &job); err != nil {
		return 0, err
	}
	return job.ID, nil
}

// StopSearch stops a running search job.
func (c *Client) StopSearch(ctx context.Context, id int) error {
	v := url.Values{"id": {strconv.Itoa(id)}}
	return c.post(ctx, "search", "stop", v, nil)
}

// GetSearchStatus returns the status of search jobs.
// Pass id=-1 to get status for all jobs.
func (c *Client) GetSearchStatus(ctx context.Context, id int) ([]SearchStatus, error) {
	params := url.Values{}
	if id >= 0 {
		params.Set("id", strconv.Itoa(id))
	}
	var status []SearchStatus
	if err := c.get(ctx, "search", "status", params, &status); err != nil {
		return nil, err
	}
	return status, nil
}

// GetSearchResults returns paginated results for a search job.
func (c *Client) GetSearchResults(ctx context.Context, id, limit, offset int) (*SearchResults, error) {
	params := url.Values{
		"id":     {strconv.Itoa(id)},
		"limit":  {strconv.Itoa(limit)},
		"offset": {strconv.Itoa(offset)},
	}
	var results SearchResults
	if err := c.get(ctx, "search", "results", params, &results); err != nil {
		return nil, err
	}
	return &results, nil
}

// DeleteSearch deletes a search job.
func (c *Client) DeleteSearch(ctx context.Context, id int) error {
	v := url.Values{"id": {strconv.Itoa(id)}}
	return c.post(ctx, "search", "delete", v, nil)
}

// GetSearchPlugins returns all installed search plugins.
func (c *Client) GetSearchPlugins(ctx context.Context) ([]SearchPlugin, error) {
	var plugins []SearchPlugin
	if err := c.get(ctx, "search", "plugins", nil, &plugins); err != nil {
		return nil, err
	}
	return plugins, nil
}

// InstallSearchPlugins installs search plugins from URLs or file paths.
func (c *Client) InstallSearchPlugins(ctx context.Context, sources []string) error {
	v := url.Values{"sources": {joinPipe(sources)}}
	return c.post(ctx, "search", "installPlugin", v, nil)
}

// UninstallSearchPlugins uninstalls plugins by name.
func (c *Client) UninstallSearchPlugins(ctx context.Context, names []string) error {
	v := url.Values{"names": {joinPipe(names)}}
	return c.post(ctx, "search", "uninstallPlugin", v, nil)
}

// EnableSearchPlugins enables or disables plugins by name.
func (c *Client) EnableSearchPlugins(ctx context.Context, names []string, enable bool) error {
	v := url.Values{
		"names":  {joinPipe(names)},
		"enable": {boolStr(enable)},
	}
	return c.post(ctx, "search", "enablePlugin", v, nil)
}

// UpdateSearchPlugins triggers an update check for all installed plugins.
func (c *Client) UpdateSearchPlugins(ctx context.Context) error {
	return c.post(ctx, "search", "updatePlugins", url.Values{}, nil)
}
