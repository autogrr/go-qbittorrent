package qbittorrent

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_MissingHost(t *testing.T) {
	_, err := NewClient(Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Host")
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	c, err := NewClient(Config{Host: "http://localhost:8080/"})
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080", c.cfg.Host)
}

func TestNewClient_Defaults(t *testing.T) {
	c, err := NewClient(Config{Host: "http://localhost:8080"})
	require.NoError(t, err)
	assert.Equal(t, defaultTimeout, c.cfg.Timeout)
	assert.Equal(t, defaultRetryAttempts, c.cfg.RetryAttempts)
	assert.Equal(t, defaultRetryDelay, c.cfg.RetryDelay)
}

func TestLogin_Success(t *testing.T) {
	fs := newFakeServer(t)
	c := fs.client(t)

	require.NoError(t, c.Login(testCtx(t)))
	assert.EqualValues(t, 1, c.loggedIn.Load())
}

func TestLogin_NoCredentials(t *testing.T) {
	// When no username or password is configured (auth-disabled qBittorrent),
	// Login must succeed without making any network request.
	fs := newFakeServer(t)
	c, err := NewClient(Config{Host: fs.srv.URL, RetryAttempts: 1})
	require.NoError(t, err)

	require.NoError(t, c.Login(testCtx(t)))
	assert.EqualValues(t, 1, c.loggedIn.Load())
	// The login endpoint must never have been called.
	assert.Equal(t, 0, fs.callCount("/api/v2/auth/login"))
}

func TestLogin_WrongPassword(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("POST", "/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		writeString(w, "Fails.")
	})
	c := fs.client(t)
	err := c.Login(testCtx(t))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrIPBanned)
	assert.EqualValues(t, 0, c.loggedIn.Load())
}

func TestLogin_Forbidden(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("POST", "/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	c := fs.client(t)
	// With RetryAttempts=1, the 403 from login itself returns an error.
	err := c.Login(testCtx(t))
	require.Error(t, err)
}

func TestLogout(t *testing.T) {
	fs := newFakeServer(t)
	fs.handle("POST", "/api/v2/auth/logout", func(w http.ResponseWriter, _ *http.Request) {
		writeString(w, "Ok.")
	})
	c := fs.loggedInClient(t)
	require.NoError(t, c.Logout(testCtx(t)))
	assert.EqualValues(t, 0, c.loggedIn.Load())
}

func TestReloginOn403(t *testing.T) {
	fs := newFakeServer(t)

	loginCalls := 0
	fs.handle("POST", "/api/v2/auth/login", func(w http.ResponseWriter, _ *http.Request) {
		loginCalls++
		writeString(w, "Ok.")
	})

	callCount := 0
	fs.handle("GET", "/api/v2/app/version", func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount == 1 {
			// First call returns 403 to trigger re-login.
			w.WriteHeader(http.StatusForbidden)
			return
		}
		writeString(w, "5.0.0")
	})

	c, err := NewClient(Config{
		Host:          fs.srv.URL,
		Username:      "admin",
		Password:      "adminadmin",
		RetryAttempts: 2,
	})
	require.NoError(t, err)
	// Pre-login so loggedIn=1; the 403 will flip it to 0 and trigger re-login.
	require.NoError(t, c.Login(testCtx(t)))

	ver, err := c.GetAppVersion(testCtx(t))
	require.NoError(t, err)
	assert.Equal(t, "5.0.0", ver)
	assert.Equal(t, 2, loginCalls) // initial + re-login
}

func TestSetGetHTTPClient(t *testing.T) {
	c, _ := NewClient(Config{Host: "http://localhost"})
	hc := &http.Client{}
	c.SetHTTPClient(hc)
	assert.Equal(t, hc, c.GetHTTPClient())
}

func TestContextCancellation(t *testing.T) {
	fs := newFakeServer(t)
	c := fs.loggedInClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	_, err := c.GetAppVersion(ctx)
	require.Error(t, err)
}

func TestHTTPErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		sentinel   error
	}{
		{"bad_request", http.StatusBadRequest, ErrBadRequest},
		{"not_found", http.StatusNotFound, ErrNotFound},
		{"conflict", http.StatusConflict, ErrConflict},
		{"unsupported_media", http.StatusUnsupportedMediaType, ErrInvalidTorrent},
		{"server_error", http.StatusInternalServerError, ErrBadResponse},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fs := newFakeServer(t)
			fs.handle("GET", "/api/v2/app/version", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.statusCode)
				writeString(w, "error body")
			})
			c := fs.loggedInClient(t)
			_, err := c.GetAppVersion(testCtx(t))
			require.Error(t, err)
			assert.ErrorIs(t, err, tc.sentinel)

			var ae *APIError
			require.ErrorAs(t, err, &ae)
			assert.Equal(t, tc.statusCode, ae.StatusCode)
			assert.Equal(t, "error body", ae.Body)
		})
	}
}

// ---- DisableCompression / autoTuneCompression ----

func TestDisableCompression_FlowsToTransport(t *testing.T) {
	// Config.DisableCompression=true must be applied to the transport at
	// construction time, before any network activity.
	c, err := NewClient(Config{Host: "http://localhost:9999", DisableCompression: true})
	require.NoError(t, err)
	assert.True(t, c.transport.DisableCompression)
}

func TestAutoTuneCompression_LocalhostDisablesCompression(t *testing.T) {
	// Logging in to a localhost server (RTT << 5 ms) must auto-disable
	// compression on the transport.
	fs := newFakeServer(t)
	c, err := NewClient(Config{
		Host:          fs.srv.URL,
		Username:      "admin",
		Password:      "adminadmin",
		RetryAttempts: 1,
	})
	require.NoError(t, err)
	assert.False(t, c.transport.DisableCompression, "should start with compression enabled")

	require.NoError(t, c.Login(testCtx(t)))
	assert.True(t, c.transport.DisableCompression, "localhost login must auto-disable compression")
}

func TestAutoTuneCompression_NoCredentials(t *testing.T) {
	// The no-credentials path must also probe latency and auto-disable
	// compression on a localhost target.
	fs := newFakeServer(t)
	c, err := NewClient(Config{Host: fs.srv.URL, RetryAttempts: 1})
	require.NoError(t, err)
	assert.False(t, c.transport.DisableCompression)

	require.NoError(t, c.Login(testCtx(t)))
	assert.True(t, c.transport.DisableCompression, "no-cred localhost login must auto-disable compression")
}

func TestAutoTuneCompression_ExplicitConfigSkipsAutoTune(t *testing.T) {
	// When DisableCompression is set in Config the transport must reflect it
	// immediately, and the probe must not overwrite it after Login.
	fs := newFakeServer(t)
	c, err := NewClient(Config{
		Host:               fs.srv.URL,
		Username:           "admin",
		Password:           "adminadmin",
		RetryAttempts:      1,
		DisableCompression: true,
	})
	require.NoError(t, err)
	assert.True(t, c.transport.DisableCompression, "must be set at construction")

	require.NoError(t, c.Login(testCtx(t)))
	// Still true — the explicit flag must be preserved.
	assert.True(t, c.transport.DisableCompression)
}

// ---- Non-replayable body ----

// opaqueReader wraps an io.Reader so that http.NewRequestWithContext does not
// recognise it as a replayable type (*bytes.Reader / *strings.Reader /
// *bytes.Buffer) and therefore leaves req.GetBody nil.
type opaqueReader struct{ io.Reader }

func TestNonReplayableBody_Returns403Error(t *testing.T) {
	fs := newFakeServer(t)
	// Endpoint always returns 403 to trigger the re-login path.
	fs.handle("POST", "/api/v2/torrents/add", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	c, err := NewClient(Config{
		Host:          fs.srv.URL,
		Username:      "admin",
		Password:      "adminadmin",
		RetryAttempts: 1,
	})
	require.NoError(t, err)
	require.NoError(t, c.Login(testCtx(t)))

	// Build a request whose body is not a recognised replayable type, so
	// http.NewRequestWithContext leaves req.GetBody==nil.
	req, err := c.newRequest(
		testCtx(t), http.MethodPost,
		c.url("torrents", "add"),
		&opaqueReader{strings.NewReader("some=data")},
	)
	require.NoError(t, err)
	require.Nil(t, req.GetBody, "opaqueReader must leave GetBody nil")

	_, err = c.do(req)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
	var ae *APIError
	require.ErrorAs(t, err, &ae)
	assert.Equal(t, http.StatusForbidden, ae.StatusCode)
}

// ---- checkStatusBytes ----

func TestCheckStatusBytes_2xx_OK(t *testing.T) {
	for _, code := range []int{200, 201, 204, 299} {
		assert.NoError(t, checkStatusBytes(code, nil), "code %d must not error", code)
	}
}

func TestCheckStatusBytes_LongBodyTruncated(t *testing.T) {
	// The snippet in the error message must be capped at 512 bytes.
	long := strings.Repeat("x", 1024)
	err := checkStatusBytes(http.StatusBadRequest, []byte(long))
	require.Error(t, err)
	var ae *APIError
	require.ErrorAs(t, err, &ae)
	assert.Len(t, ae.Body, 512)
}

func TestCheckStatusBytes_AllCodes(t *testing.T) {
	tests := []struct {
		code     int
		sentinel error
	}{
		{http.StatusBadRequest, ErrBadRequest},
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusForbidden, ErrUnauthorized},
		{http.StatusNotFound, ErrNotFound},
		{http.StatusConflict, ErrConflict},
		{http.StatusUnsupportedMediaType, ErrInvalidTorrent},
		{http.StatusInternalServerError, ErrBadResponse},
		{http.StatusServiceUnavailable, ErrBadResponse},
	}
	for _, tc := range tests {
		t.Run(http.StatusText(tc.code), func(t *testing.T) {
			err := checkStatusBytes(tc.code, []byte("msg"))
			require.Error(t, err)
			assert.ErrorIs(t, err, tc.sentinel)
		})
	}
}
