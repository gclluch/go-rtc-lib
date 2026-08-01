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

// A deliberate server-side close must reach the peer as a WebSocket Close
// frame, not as a dropped socket.
//
// CloseConnection used to close `done` and the socket back to back. The write
// pump woke on `done` and tried to write the Close frame against a socket that
// was often already gone, so the client saw an abnormal closure (1006) - a
// planned shutdown was indistinguishable from a network fault. Whether it
// worked came down to which goroutine the scheduler ran first, so this test
// repeats to catch a regression that only sometimes loses the race.
func TestServerCloseSendsCloseFrame(t *testing.T) {
	for attempt := 0; attempt < 5; attempt++ {
		r := NewRegistry()
		ctx, cancel := context.WithCancel(context.Background())
		go r.Run(ctx)

		r.CheckOrigin = func(*http.Request) bool { return true }
		srv := httptest.NewServer(http.HandlerFunc(r.RegisterHandler(nil)))

		c, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
		if err != nil {
			t.Fatalf("attempt %d: dial: %v", attempt, err)
		}
		time.Sleep(50 * time.Millisecond) // let register land

		cancel() // triggers closeAll -> CloseConnection on every live connection

		// ReadMessage surfaces the close handshake as an error we can classify.
		c.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, _, readErr := c.ReadMessage()

		if websocket.IsCloseError(readErr, websocket.CloseAbnormalClosure) {
			c.Close()
			srv.Close()
			t.Fatalf("attempt %d: peer saw an abnormal closure (1006); "+
				"the socket was closed before the Close frame was written: %v", attempt, readErr)
		}
		if !websocket.IsCloseError(readErr, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
			c.Close()
			srv.Close()
			t.Fatalf("attempt %d: expected a clean close frame, got %v", attempt, readErr)
		}

		c.Close()
		srv.Close()
	}
}
