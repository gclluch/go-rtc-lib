package connection

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// A connection still alive when Run's context is cancelled must not strand its
// HTTP handler goroutine on the unregister send.
func TestNoGoroutineLeakOnShutdown(t *testing.T) {
	r := NewRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	go r.Run(ctx)

	r.CheckOrigin = func(*http.Request) bool { return true }
	srv := httptest.NewServer(http.HandlerFunc(r.RegisterHandler(nil)))
	defer srv.Close()

	c, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()
	time.Sleep(50 * time.Millisecond) // let register land

	cancel()
	time.Sleep(300 * time.Millisecond)

	// Count goroutines parked inside RegisterHandler. The pumps exit on close,
	// so a raw NumGoroutine comparison hides this - the handler is what leaks.
	buf := make([]byte, 1<<18)
	stacks := string(buf[:runtime.Stack(buf, true)])
	stranded := strings.Count(stacks, "RegisterHandler")
	if stranded > 0 {
		t.Fatalf("%d handler goroutine(s) stranded after shutdown:\n%s", stranded, stacks)
	}
}

// An oversized frame must be rejected, not allocated.
func TestReadLimitRejectsOversizedFrame(t *testing.T) {
	r := NewRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)

	r.CheckOrigin = func(*http.Request) bool { return true }
	srv := httptest.NewServer(http.HandlerFunc(r.RegisterHandler(nil)))
	defer srv.Close()

	c, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	if err := c.WriteMessage(websocket.TextMessage, make([]byte, maxMessageBytes+1)); err != nil {
		t.Fatalf("write: %v", err)
	}
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err = c.ReadMessage(); err == nil {
		t.Fatal("expected the server to close the connection on an oversized frame")
	}
}
