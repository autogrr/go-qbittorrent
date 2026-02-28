// Package qbittorrent provides a test server helper shared across all _test.go files.
package qbittorrent

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// route is a single registered handler on the fake server.
type route struct {
	method  string // "GET" or "POST" or "" for any
	path    string // URL path, e.g. "/api/v2/auth/login"
	handler http.HandlerFunc
}

// fakeServer wraps httptest.Server with a simple route table.
type fakeServer struct {
	mu     sync.RWMutex
	routes []route
	srv    *httptest.Server
	calls  map[string]int // path -> call count
}

func newFakeServer(t *testing.T) *fakeServer {
	t.Helper()
	fs := &fakeServer{calls: make(map[string]int)}
	fs.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.mu.Lock()
		fs.calls[r.URL.Path]++
		fs.mu.Unlock()

		fs.mu.RLock()
		defer fs.mu.RUnlock()
		for _, rt := range fs.routes {
			if rt.path != r.URL.Path {
				continue
			}
			if rt.method != "" && rt.method != r.Method {
				continue
			}
			rt.handler(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(fs.srv.Close)

	// Default login handler — always succeeds.
	fs.handle("POST", "/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Ok."))
	})
	return fs
}

// handle registers (or replaces) a route handler.
func (fs *fakeServer) handle(method, path string, h http.HandlerFunc) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	for i, rt := range fs.routes {
		if rt.path == path && rt.method == method {
			fs.routes[i].handler = h
			return
		}
	}
	fs.routes = append(fs.routes, route{method: method, path: path, handler: h})
}

// callCount returns how many times path was hit.
func (fs *fakeServer) callCount(path string) int {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.calls[path]
}

// client returns a logged-in Client pointed at the fake server.
func (fs *fakeServer) client(t *testing.T) *Client {
	t.Helper()
	c, err := NewClient(Config{
		Host:          fs.srv.URL,
		Username:      "admin",
		Password:      "adminadmin",
		RetryAttempts: 1,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// loggedInClient creates and logs in a client.
func (fs *fakeServer) loggedInClient(t *testing.T) *Client {
	t.Helper()
	c := fs.client(t)
	ctx := testCtx(t)
	if err := c.Login(ctx); err != nil {
		t.Fatalf("Login: %v", err)
	}
	return c
}

// --- JSON helpers ---

// writeJSON writes v as JSON with status 200.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// writeString writes a plain text response.
func writeString(w http.ResponseWriter, s string) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = io.WriteString(w, s)
}

// formValue reads a POST form field, parsing the body once.
func formValue(r *http.Request, key string) string {
	_ = r.ParseForm()
	return r.FormValue(key)
}

// hasFormValue returns true if the form contains the key.
func hasFormValue(r *http.Request, key string) bool {
	_ = r.ParseForm()
	_, ok := r.Form[key]
	return ok
}

// queryValue reads a URL query parameter.
func queryValue(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

// decodeJSON decodes the request body as JSON.
func decodeJSON(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

// makeHashes builds a pipe-separated hash string from a slice.
func makeHashes(hh ...string) string { return strings.Join(hh, "|") }

// encodePrefs encodes preferences as a json URL param.
func encodePrefs(prefs map[string]any) url.Values {
	b, _ := json.Marshal(prefs)
	return url.Values{"json": {string(b)}}
}
