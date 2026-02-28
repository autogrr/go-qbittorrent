package qbittorrent

import (
	"context"
	"testing"
	"time"
)

// testCtx returns a context that is cancelled when the test ends (10s deadline).
func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}
