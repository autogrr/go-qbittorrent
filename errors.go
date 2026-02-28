package qbittorrent

import "fmt"

// Sentinel errors returned by the client.
var (
ErrBadResponse    = fmt.Errorf("qbittorrent: bad response from server")
ErrUnauthorized   = fmt.Errorf("qbittorrent: unauthorized (403)")
ErrNotFound       = fmt.Errorf("qbittorrent: not found (404)")
ErrConflict       = fmt.Errorf("qbittorrent: conflict (409)")
ErrBadRequest     = fmt.Errorf("qbittorrent: bad request (400)")
ErrLoginFailed    = fmt.Errorf("qbittorrent: login failed")
ErrIPBanned       = fmt.Errorf("qbittorrent: IP banned (too many failed logins)")
ErrInvalidTorrent = fmt.Errorf("qbittorrent: invalid torrent file (415)")
ErrAlreadyExists  = fmt.Errorf("qbittorrent: torrent already exists")
)

// APIError wraps an HTTP-level or API-level error with additional context.
type APIError struct {
StatusCode int
Body       string
Sentinel   error
}

func (e *APIError) Error() string {
if e.Body != "" {
return fmt.Sprintf("%v: status=%d body=%q", e.Sentinel, e.StatusCode, e.Body)
}
return fmt.Sprintf("%v: status=%d", e.Sentinel, e.StatusCode)
}

func (e *APIError) Unwrap() error { return e.Sentinel }

func apiErr(sentinel error, code int, body string) *APIError {
return &APIError{Sentinel: sentinel, StatusCode: code, Body: body}
}
