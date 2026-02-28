package qbittorrent

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/publicsuffix"
)

// bufPool is a package-level pool of *bytes.Buffer used for all HTTP response
// and multipart-request bodies. Each buffer starts at 64 KiB and grows as
// needed; Reset() on return keeps the underlying memory for the next caller.
var bufPool = sync.Pool{
	New: func() any { return bytes.NewBuffer(make([]byte, 0, bufSize)) },
}

func getBuf() *bytes.Buffer  { return bufPool.Get().(*bytes.Buffer) }
func putBuf(b *bytes.Buffer) { b.Reset(); bufPool.Put(b) }

const (
	defaultTimeout       = 60 * time.Second
	defaultRetryAttempts = 3
	defaultRetryDelay    = 500 * time.Millisecond
	apiPrefix            = "/api/v2/"
	bufSize              = 64 * 1024 // 64 KiB read/write buffers

	// latencyThreshold is the round-trip cutoff below which a connection is
	// considered local/LAN. Below this threshold compression is auto-disabled:
	// qBittorrent compresses responses synchronously on its event-loop thread
	// at gzip level 6, which wastes CPU on both sides with no bandwidth benefit
	// for low-latency paths. Above the threshold (WAN), compression helps.
	latencyThreshold = 5 * time.Millisecond
)

// Config holds the connection settings for a qBittorrent client.
type Config struct {
	// Host is the base URL, e.g. "http://localhost:8080" (no trailing slash).
	Host string

	// Credentials for the WebUI.
	Username string
	Password string

	// Optional HTTP Basic-Auth credentials (e.g. for a reverse proxy).
	BasicUser string
	BasicPass string

	// TLS
	TLSSkipVerify bool

	// Timeouts & retries.
	Timeout       time.Duration // default 60s
	RetryAttempts int           // default 3 (0 disables retry)
	RetryDelay    time.Duration // default 500ms

	// DisableCompression prevents the client from advertising gzip support in
	// the Accept-Encoding request header. By default Go's transport sends
	// "Accept-Encoding: gzip", which causes qBittorrent to compress every API
	// response > 1 KiB at gzip level 6, synchronously on its event-loop thread.
	// For local or LAN deployments this is pure overhead: CPU burned on both
	// ends with no bandwidth benefit. Set to true to skip compression entirely.
	DisableCompression bool
}

// Client is a thread-safe qBittorrent WebUI API client.
// All methods accept a context.Context as the first argument.
type Client struct {
	cfg     Config
	baseURL string

	// loginForm is the URL-encoded login body pre-computed at construction time.
	loginForm string

	http      *http.Client
	transport *http.Transport // retained so loginLocked can tune DisableCompression
	jar       http.CookieJar

	// loggedIn tracks whether we currently have a valid session.
	// 0 = no session, 1 = session active.
	loggedIn atomic.Int32

	mu sync.Mutex // guards re-login
}

// NewClient creates a new Client with sensible defaults.
func NewClient(cfg Config) (*Client, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("qbittorrent: Host must not be empty")
	}
	// Normalise host: strip trailing slash.
	cfg.Host = strings.TrimRight(cfg.Host, "/")

	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.RetryAttempts == 0 {
		cfg.RetryAttempts = defaultRetryAttempts
	}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = defaultRetryDelay
	}

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("qbittorrent: cookiejar: %w", err)
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 90 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          64,
		MaxIdleConnsPerHost:   32,
		MaxConnsPerHost:       64,
		IdleConnTimeout:       120 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ReadBufferSize:        bufSize,
		WriteBufferSize:       bufSize,
		DisableCompression:    cfg.DisableCompression,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.TLSSkipVerify, //nolint:gosec
		},
	}

	httpClient := &http.Client{
		Transport: transport,
		Jar:       jar,
		Timeout:   cfg.Timeout,
	}

	c := &Client{
		cfg:       cfg,
		baseURL:   cfg.Host + apiPrefix,
		http:      httpClient,
		transport: transport,
		jar:       jar,
		loginForm: url.Values{"username": {cfg.Username}, "password": {cfg.Password}}.Encode(),
	}
	return c, nil
}

// SetHTTPClient replaces the underlying *http.Client (e.g. for custom transports).
func (c *Client) SetHTTPClient(hc *http.Client) {
	c.http = hc
}

// GetHTTPClient returns the underlying *http.Client.
func (c *Client) GetHTTPClient() *http.Client { return c.http }

// ---- Low-level HTTP helpers ----

func (c *Client) url(scope, method string) string {
	return c.baseURL + scope + "/" + method
}

// do executes the request, retrying on transient errors and re-logging in on 403.
// On each retry the request body is reset via req.GetBody (set automatically by
// http.NewRequestWithContext for strings.Reader / bytes.Reader bodies). If the
// body cannot be replayed (e.g. an io.Pipe for multipart uploads), a 403 is
// surfaced immediately as ErrUnauthorized rather than attempting a garbled retry.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	attempts := c.cfg.RetryAttempts
	if attempts < 1 {
		attempts = 1
	}
	var (
		resp *http.Response
		err  error
	)
	for i := 0; i < attempts; i++ {
		if i > 0 {
			// Reset the request body so we send the same payload again.
			if req.GetBody != nil {
				var body io.ReadCloser
				if body, err = req.GetBody(); err != nil {
					return nil, err
				}
				req.Body = body
			}
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(c.cfg.RetryDelay):
			}
		}
		resp, err = c.http.Do(req)
		if err != nil {
			// Retry transient network errors.
			continue
		}
		if resp.StatusCode == http.StatusForbidden {
			// Drain the 403 body fully so this connection can be reused.
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			// Mark session invalid before re-login.
			c.loggedIn.Store(0)
			if loginErr := c.reLogin(req.Context()); loginErr != nil {
				return nil, loginErr
			}
			// Non-seekable bodies (pipes) cannot be replayed; bail out.
			if req.GetBody == nil && req.Body != nil {
				return nil, &APIError{
					StatusCode: http.StatusForbidden,
					Body:       "session expired; re-login succeeded but request body is not replayable",
					Sentinel:   ErrUnauthorized,
				}
			}
			continue
		}
		return resp, nil
	}
	if err != nil {
		return nil, fmt.Errorf("qbittorrent: request failed after %d attempts: %w", attempts, err)
	}
	return resp, nil
}

// reLogin acquires the mutex so only one goroutine re-authenticates at a time.
func (c *Client) reLogin(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Another goroutine may have already re-authenticated.
	if c.loggedIn.Load() == 1 {
		return nil
	}
	return c.loginLocked(ctx)
}

func (c *Client) loginLocked(ctx context.Context) error {
	// No credentials configured → qBittorrent has auth disabled.
	// Skip the HTTP round-trip but still probe TCP latency so we can
	// auto-disable compression on local/LAN connections.
	if c.cfg.Username == "" && c.cfg.Password == "" {
		c.loggedIn.Store(1)
		if !c.cfg.DisableCompression {
			c.autoTuneCompression(ctx)
		}
		return nil
	}

	// Probe TCP latency before the login request. We cannot use the HTTP RTT
	// because qBittorrent's login handler runs PBKDF2 verification (10-100 ms)
	// regardless of network distance, which would mask whether the host is local.
	if !c.cfg.DisableCompression {
		c.autoTuneCompression(ctx)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("auth", "login"), strings.NewReader(c.loginForm))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if c.cfg.BasicUser != "" {
		req.SetBasicAuth(c.cfg.BasicUser, c.cfg.BasicPass)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("qbittorrent: login request: %w", err)
	}
	defer resp.Body.Close()

	b, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("qbittorrent: login read body: %w", readErr)
	}
	body := strings.TrimSpace(string(b))

	switch {
	case resp.StatusCode == http.StatusForbidden || body == "Fails.":
		return ErrIPBanned
	case body == "Ok.":
		c.loggedIn.Store(1)
		return nil
	default:
		return fmt.Errorf("%w: unexpected response: %s", ErrLoginFailed, body)
	}
}

// autoTuneCompression dials the server host with a short timeout and disables
// gzip compression on the transport when the TCP round-trip is below the
// latency threshold. Failures are silently ignored: the transport stays at its
// current (default-on) compression setting.
func (c *Client) autoTuneCompression(ctx context.Context) {
	u, err := url.Parse(c.cfg.Host)
	if err != nil {
		return
	}
	addr := u.Host
	if u.Port() == "" {
		if u.Scheme == "https" {
			addr = addr + ":443"
		} else {
			addr = addr + ":80"
		}
	}
	// Give the dial 4× the threshold to avoid false-negatives on a momentarily
	// busy loopback stack, but only disable compression if the actual RTT is
	// below the threshold itself.
	dialCtx, cancel := context.WithTimeout(ctx, 4*latencyThreshold)
	defer cancel()
	start := time.Now()
	conn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", addr)
	rtt := time.Since(start)
	if err != nil {
		return
	}
	conn.Close()
	if rtt < latencyThreshold {
		c.transport.DisableCompression = true
	}
}

// newRequest builds a request with Basic-Auth if configured.
func (c *Client) newRequest(ctx context.Context, method, u string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	if c.cfg.BasicUser != "" {
		req.SetBasicAuth(c.cfg.BasicUser, c.cfg.BasicPass)
	}
	return req, nil
}

// consumeJSON streams the response body directly into dst via json.NewDecoder,
// so network bytes flow straight into the JSON parser with no intermediate copy.
// Non-2xx responses are read with io.ReadAll (error path — allocation is fine).
func consumeJSON(resp *http.Response, dst any) error {
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("qbittorrent: read body: %w", err)
		}
		return checkStatusBytes(resp.StatusCode, b)
	}
	if dst == nil {
		_, err := io.Copy(io.Discard, resp.Body)
		return err
	}
	err := json.NewDecoder(resp.Body).Decode(dst)
	_, _ = io.Copy(io.Discard, resp.Body)
	return err
}

// consumeString reads the response body into a pooled buffer, checks the status,
// and returns the trimmed body as an owned string.
func consumeString(resp *http.Response) (string, error) {
	buf := getBuf()
	_, readErr := buf.ReadFrom(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		putBuf(buf)
		return "", fmt.Errorf("qbittorrent: read body: %w", readErr)
	}
	if err := checkStatusBytes(resp.StatusCode, buf.Bytes()); err != nil {
		putBuf(buf)
		return "", err
	}
	s := strings.TrimSpace(buf.String())
	putBuf(buf)
	return s, nil
}

// consumeRaw reads the response body into an owned slice via io.ReadAll and
// checks the status code. The caller owns the returned bytes.
func consumeRaw(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("qbittorrent: read body: %w", err)
	}
	if err := checkStatusBytes(resp.StatusCode, b); err != nil {
		return nil, err
	}
	return b, nil
}

// get performs a GET request and decodes the JSON response body into dst.
// If dst is nil the body is discarded after the status check.
func (c *Client) get(ctx context.Context, scope, method string, params url.Values, dst any) error {
	u := c.url(scope, method)
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := c.newRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	return consumeJSON(resp, dst)
}

// getString performs a GET request and returns the trimmed response body.
func (c *Client) getString(ctx context.Context, scope, method string, params url.Values) (string, error) {
	u := c.url(scope, method)
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := c.newRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.do(req)
	if err != nil {
		return "", err
	}
	return consumeString(resp)
}

// post performs a POST with URL-encoded form data and decodes JSON into dst.
// If dst is nil the body is discarded after the status check.
func (c *Client) post(ctx context.Context, scope, method string, form url.Values, dst any) error {
	encoded := form.Encode()
	req, err := c.newRequest(ctx, http.MethodPost, c.url(scope, method), strings.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	return consumeJSON(resp, dst)
}

// postMultipart POSTs a multipart/form-data request with torrent bytes from r
// under the given fieldName/filename. Extra form fields from extra are added
// before the file part.
//
// The entire multipart body is buffered synchronously into a pooled bytes.Buffer
// before the request is issued. This eliminates the io.Pipe + goroutine pattern,
// removing all closure allocations from the hot path. Torrent metadata files are
// small (typically ≤ 100 KiB) so in-memory buffering is always appropriate.
func (c *Client) postMultipart(ctx context.Context, scope, method string, extra url.Values, fieldName, filename string, r io.Reader, dst any) error {
	buf := getBuf()
	mw := multipart.NewWriter(buf)

	for key, vals := range extra {
		for _, val := range vals {
			if err := mw.WriteField(key, val); err != nil {
				putBuf(buf)
				return err
			}
		}
	}
	part, err := mw.CreateFormFile(fieldName, filename)
	if err != nil {
		putBuf(buf)
		return err
	}
	if _, err = io.Copy(part, r); err != nil {
		putBuf(buf)
		return err
	}
	if err = mw.Close(); err != nil {
		putBuf(buf)
		return err
	}

	contentType := mw.FormDataContentType()
	// Copy the multipart body to an owned slice, then return the pool buffer
	// immediately. bytes.NewReader over buf.Bytes() would hold a reference to the
	// pool buffer's underlying array; with HTTP/2 the transport can send the
	// request body in a background goroutine after Do returns, creating a race
	// if another caller reuses the buffer from the pool in the meantime.
	body := make([]byte, buf.Len())
	copy(body, buf.Bytes())
	putBuf(buf)

	req, err := c.newRequest(ctx, http.MethodPost, c.url(scope, method), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = int64(len(body))

	resp, rerr := c.do(req)
	if rerr != nil {
		return rerr
	}
	return consumeJSON(resp, dst)
}

// postMultipartFiles POSTs multiple .torrent files from a map of filename→reader.
func (c *Client) postMultipartFiles(ctx context.Context, scope, method string, extra url.Values, files map[string]io.Reader, dst any) error {
	buf := getBuf()
	mw := multipart.NewWriter(buf)

	for key, vals := range extra {
		for _, val := range vals {
			if err := mw.WriteField(key, val); err != nil {
				putBuf(buf)
				return err
			}
		}
	}
	for name, r := range files {
		part, err := mw.CreateFormFile("torrents", filepath.Base(name))
		if err != nil {
			putBuf(buf)
			return err
		}
		if _, err = io.Copy(part, r); err != nil {
			putBuf(buf)
			return err
		}
	}
	if err := mw.Close(); err != nil {
		putBuf(buf)
		return err
	}

	contentType := mw.FormDataContentType()
	body := make([]byte, buf.Len())
	copy(body, buf.Bytes())
	putBuf(buf)

	req, err := c.newRequest(ctx, http.MethodPost, c.url(scope, method), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = int64(len(body))

	resp, rerr := c.do(req)
	if rerr != nil {
		return rerr
	}
	return consumeJSON(resp, dst)
}

// getRaw performs a GET and returns the raw response bytes.
func (c *Client) getRaw(ctx context.Context, scope, method string, params url.Values) ([]byte, error) {
	u := c.url(scope, method)
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := c.newRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	return consumeRaw(resp)
}

// checkStatusBytes converts non-2xx responses into typed errors.
// It receives the already-fully-read body so no draining is needed by the caller.
func checkStatusBytes(code int, body []byte) error {
	if code >= 200 && code < 300 {
		return nil
	}
	snippet := strings.TrimSpace(string(body))
	if len(snippet) > 512 {
		snippet = snippet[:512]
	}
	switch code {
	case http.StatusBadRequest:
		return apiErr(ErrBadRequest, code, snippet)
	case http.StatusUnauthorized, http.StatusForbidden:
		return apiErr(ErrUnauthorized, code, snippet)
	case http.StatusNotFound:
		return apiErr(ErrNotFound, code, snippet)
	case http.StatusConflict:
		return apiErr(ErrConflict, code, snippet)
	case http.StatusUnsupportedMediaType:
		return apiErr(ErrInvalidTorrent, code, snippet)
	default:
		return apiErr(ErrBadResponse, code, snippet)
	}
}

// joinHashes encodes a slice of hashes as pipe-separated values,
// or the literal "all" when the slice is empty.
// The builder is pre-sized to avoid any internal reallocation.
func joinHashes(hashes []string) string {
	if len(hashes) == 0 {
		return "all"
	}
	return joinSep(hashes, '|')
}

// joinPipe joins a slice with pipe separators.
func joinPipe(ss []string) string { return joinSep(ss, '|') }

// joinNewline joins a slice with newline separators.
func joinNewline(ss []string) string { return joinSep(ss, '\n') }

// joinComma joins a slice with comma separators.
func joinComma(ss []string) string { return joinSep(ss, ',') }

// joinSep is the shared implementation: pre-computes the exact output length,
// grows the Builder once, then writes without further allocation.
func joinSep(ss []string, sep byte) string {
	if len(ss) == 0 {
		return ""
	}
	n := len(ss) - 1 // separators
	for _, s := range ss {
		n += len(s)
	}
	var b strings.Builder
	b.Grow(n)
	b.WriteString(ss[0])
	for _, s := range ss[1:] {
		b.WriteByte(sep)
		b.WriteString(s)
	}
	return b.String()
}
