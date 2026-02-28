package qbittorrent

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- RSS tests ----------

func TestAddRSSFolder(t *testing.T) {
	fs := newFakeServer(t)
	var gotPath string
	fs.handle("POST", "/api/v2/rss/addFolder", func(w http.ResponseWriter, r *http.Request) {
		gotPath = formValue(r, "path")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.AddRSSFolder(testCtx(t), "My Feeds"))
	assert.Equal(t, "My Feeds", gotPath)
}

func TestAddRSSFeed(t *testing.T) {
	fs := newFakeServer(t)
	var gotURL, gotPath string
	fs.handle("POST", "/api/v2/rss/addFeed", func(w http.ResponseWriter, r *http.Request) {
		gotURL = formValue(r, "url")
		gotPath = formValue(r, "path")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.AddRSSFeed(testCtx(t), "https://example.com/feed.rss", "My Feeds\\Example"))
	assert.Equal(t, "https://example.com/feed.rss", gotURL)
	assert.Equal(t, "My Feeds\\Example", gotPath)
}

func TestRemoveRSSItem(t *testing.T) {
	fs := newFakeServer(t)
	var gotPath string
	fs.handle("POST", "/api/v2/rss/removeItem", func(w http.ResponseWriter, r *http.Request) {
		gotPath = formValue(r, "path")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.RemoveRSSItem(testCtx(t), "My Feeds\\Example"))
	assert.Equal(t, "My Feeds\\Example", gotPath)
}

func TestMoveRSSItem(t *testing.T) {
	fs := newFakeServer(t)
	var gotOrig, gotDest string
	fs.handle("POST", "/api/v2/rss/moveItem", func(w http.ResponseWriter, r *http.Request) {
		gotOrig = formValue(r, "itemPath")
		gotDest = formValue(r, "destPath")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.MoveRSSItem(testCtx(t), "Old\\Path", "New\\Path"))
	assert.Equal(t, "Old\\Path", gotOrig)
	assert.Equal(t, "New\\Path", gotDest)
}

func TestGetRSSItems(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/rss/items", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "true", queryValue(r, "withData"))
		writeJSON(w, map[string]any{
			"My Feeds": map[string]any{
				"Example": map[string]any{
					"uid":   "abc-123",
					"url":   "https://example.com/feed.rss",
					"title": "Example Feed",
				},
			},
		})
	})
	c := fs.loggedInClient(t)
	items, err := c.GetRSSItems(testCtx(t), true)
	require.NoError(t, err)
	assert.NotNil(t, items)
}

func TestSetRSSFeedURL(t *testing.T) {
	fs := newFakeServer(t)
	var gotPath, gotURL string
	fs.handle("POST", "/api/v2/rss/setFeedURL", func(w http.ResponseWriter, r *http.Request) {
		gotPath = formValue(r, "path")
		gotURL = formValue(r, "url")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.SetRSSFeedURL(testCtx(t), "My Feeds\\Example", "https://new.example.com/feed.rss"))
	assert.Equal(t, "My Feeds\\Example", gotPath)
	assert.Equal(t, "https://new.example.com/feed.rss", gotURL)
}

func TestMarkRSSItemAsRead(t *testing.T) {
	fs := newFakeServer(t)
	var gotPath, gotID string
	fs.handle("POST", "/api/v2/rss/markAsRead", func(w http.ResponseWriter, r *http.Request) {
		gotPath = formValue(r, "itemPath")
		gotID = formValue(r, "articleId")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.MarkRSSItemAsRead(testCtx(t), "My Feeds\\Example", "article-001"))
	assert.Equal(t, "My Feeds\\Example", gotPath)
	assert.Equal(t, "article-001", gotID)
}

func TestRefreshRSSItem(t *testing.T) {
	fs := newFakeServer(t)
	var gotPath string
	fs.handle("POST", "/api/v2/rss/refreshItem", func(w http.ResponseWriter, r *http.Request) {
		gotPath = formValue(r, "itemPath")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.RefreshRSSItem(testCtx(t), "My Feeds\\Example"))
	assert.Equal(t, "My Feeds\\Example", gotPath)
}

func TestGetRSSRules(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/rss/rules", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]RSSAutoDownloadRule{
			"Arch ISO": {
				Enabled:       true,
				MustContain:   "Arch",
				AffectedFeeds: []string{"https://example.com/feed.rss"},
				SavePath:      "/data/arch",
			},
		})
	})
	c := fs.loggedInClient(t)
	rules, err := c.GetRSSRules(testCtx(t))
	require.NoError(t, err)
	rule, ok := rules["Arch ISO"]
	require.True(t, ok)
	assert.True(t, rule.Enabled)
	assert.Equal(t, "Arch", rule.MustContain)
}

func TestSetRSSRule(t *testing.T) {
	fs := newFakeServer(t)
	var gotName string
	var gotRuleDef string
	fs.handle("POST", "/api/v2/rss/setRule", func(w http.ResponseWriter, r *http.Request) {
		gotName = formValue(r, "ruleName")
		gotRuleDef = formValue(r, "ruleDef")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	rule := RSSAutoDownloadRule{
		Enabled:     true,
		MustContain: "Arch",
	}
	require.NoError(t, c.SetRSSRule(testCtx(t), "Arch ISO", rule))
	assert.Equal(t, "Arch ISO", gotName)
	assert.Contains(t, gotRuleDef, "Arch") // ruleDef is JSON-encoded
}

func TestRenameRSSRule(t *testing.T) {
	fs := newFakeServer(t)
	var gotOld, gotNew string
	fs.handle("POST", "/api/v2/rss/renameRule", func(w http.ResponseWriter, r *http.Request) {
		gotOld = formValue(r, "ruleName")
		gotNew = formValue(r, "newRuleName")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.RenameRSSRule(testCtx(t), "Old Rule", "New Rule"))
	assert.Equal(t, "Old Rule", gotOld)
	assert.Equal(t, "New Rule", gotNew)
}

func TestRemoveRSSRule(t *testing.T) {
	fs := newFakeServer(t)
	var gotName string
	fs.handle("POST", "/api/v2/rss/removeRule", func(w http.ResponseWriter, r *http.Request) {
		gotName = formValue(r, "ruleName")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.RemoveRSSRule(testCtx(t), "Arch ISO"))
	assert.Equal(t, "Arch ISO", gotName)
}

func TestGetRSSMatchingArticles(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/rss/matchingArticles", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Arch ISO", queryValue(r, "ruleName"))
		writeJSON(w, map[string][]string{
			"https://example.com/feed.rss": {"Arch Linux 2024.01.01"},
		})
	})
	c := fs.loggedInClient(t)
	articles, err := c.GetRSSMatchingArticles(testCtx(t), "Arch ISO")
	require.NoError(t, err)
	assert.Contains(t, articles["https://example.com/feed.rss"], "Arch Linux 2024.01.01")
}

// ---------- Search tests ----------

func TestStartSearch(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("POST", "/api/v2/search/start", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "arch linux", formValue(r, "pattern"))
		assert.Equal(t, "all", formValue(r, "plugins"))
		assert.Equal(t, "all", formValue(r, "category"))
		writeJSON(w, SearchJob{ID: 42})
	})
	c := fs.loggedInClient(t)
	id, err := c.StartSearch(testCtx(t), "arch linux", "all", "all")
	require.NoError(t, err)
	assert.Equal(t, 42, id)
}

func TestStopSearch(t *testing.T) {
	fs := newFakeServer(t)
	var gotID string
	fs.handle("POST", "/api/v2/search/stop", func(w http.ResponseWriter, r *http.Request) {
		gotID = formValue(r, "id")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.StopSearch(testCtx(t), 42))
	assert.Equal(t, "42", gotID)
}

func TestGetSearchStatus(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/search/status", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "42", queryValue(r, "id"))
		writeJSON(w, []SearchStatus{
			{ID: Ptr(42), Status: Ptr("Running"), Total: Ptr(15)},
		})
	})
	c := fs.loggedInClient(t)
	statuses, err := c.GetSearchStatus(testCtx(t), 42)
	require.NoError(t, err)
	require.Len(t, statuses, 1)
	assert.Equal(t, "Running", Deref(statuses[0].Status))
	assert.Equal(t, 15, Deref(statuses[0].Total))
}

func TestGetSearchResults(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/search/results", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "42", queryValue(r, "id"))
		assert.Equal(t, "20", queryValue(r, "limit"))
		assert.Equal(t, "0", queryValue(r, "offset"))
		writeJSON(w, SearchResults{
			Results: []SearchResult{
				{FileName: Ptr("arch-linux-2024.01.01-x86_64.iso"), FileSize: Ptr[int64](800 * 1024 * 1024), FileURL: Ptr("magnet:?xt=urn:btih:abc")},
			},
			Status: Ptr("Stopped"),
			Total:  Ptr(1),
		})
	})
	c := fs.loggedInClient(t)
	results, err := c.GetSearchResults(testCtx(t), 42, 20, 0)
	require.NoError(t, err)
	assert.Equal(t, "Stopped", Deref(results.Status))
	require.Len(t, results.Results, 1)
	assert.Equal(t, "arch-linux-2024.01.01-x86_64.iso", Deref(results.Results[0].FileName))
}

func TestDeleteSearch(t *testing.T) {
	fs := newFakeServer(t)
	var gotID string
	fs.handle("POST", "/api/v2/search/delete", func(w http.ResponseWriter, r *http.Request) {
		gotID = formValue(r, "id")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.DeleteSearch(testCtx(t), 42))
	assert.Equal(t, "42", gotID)
}

func TestGetSearchPlugins(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("GET", "/api/v2/search/plugins", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []SearchPlugin{
			{Name: "legittorrents", FullName: "LegitTorrents", Enabled: true, Version: "1.0"},
			{Name: "piratebay", FullName: "The Pirate Bay", Enabled: false, Version: "2.3"},
		})
	})
	c := fs.loggedInClient(t)
	plugins, err := c.GetSearchPlugins(testCtx(t))
	require.NoError(t, err)
	require.Len(t, plugins, 2)
	assert.True(t, plugins[0].Enabled)
	assert.Equal(t, "The Pirate Bay", plugins[1].FullName)
}

func TestInstallSearchPlugins(t *testing.T) {
	fs := newFakeServer(t)
	var gotSources string
	fs.handle("POST", "/api/v2/search/installPlugin", func(w http.ResponseWriter, r *http.Request) {
		gotSources = formValue(r, "sources")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.InstallSearchPlugins(testCtx(t), []string{
		"https://raw.githubusercontent.com/example/plugin.py",
	}))
	assert.Contains(t, gotSources, "raw.githubusercontent.com")
}

func TestUninstallSearchPlugins(t *testing.T) {
	fs := newFakeServer(t)
	var gotNames string
	fs.handle("POST", "/api/v2/search/uninstallPlugin", func(w http.ResponseWriter, r *http.Request) {
		gotNames = formValue(r, "names")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.UninstallSearchPlugins(testCtx(t), []string{"legittorrents", "piratebay"}))
	assert.Equal(t, "legittorrents|piratebay", gotNames)
}

func TestEnableSearchPlugins(t *testing.T) {
	fs := newFakeServer(t)
	var gotNames, gotEnabled string
	fs.handle("POST", "/api/v2/search/enablePlugin", func(w http.ResponseWriter, r *http.Request) {
		gotNames = formValue(r, "names")
		gotEnabled = formValue(r, "enable")
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.EnableSearchPlugins(testCtx(t), []string{"legittorrents"}, true))
	assert.Equal(t, "legittorrents", gotNames)
	assert.Equal(t, "true", gotEnabled)
}

func TestUpdateSearchPlugins(t *testing.T) {
	fs := newFakeServer(t)
	var called bool
	fs.handle("POST", "/api/v2/search/updatePlugins", func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.UpdateSearchPlugins(testCtx(t)))
	assert.True(t, called)
}
