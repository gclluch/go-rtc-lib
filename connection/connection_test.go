package connection_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gclluch/go-rtc-lib/connection"
	"github.com/gclluch/go-rtc-lib/message"

	"github.com/gorilla/websocket"
)

// echoHandler implements connection.MessageHandler for testing.
type echoHandler struct{}

func (echoHandler) HandleMessage(conn *connection.Connection, msg []byte) ([]byte, error) {
	return msg, nil
}

// dialWebSocket helps in establishing a WebSocket connection for testing.
func dialWebSocket(serverURL string) (*websocket.Conn, *http.Response, error) {
	wsURL := "ws" + serverURL[len("http"):]
	return websocket.DefaultDialer.Dial(wsURL, nil)
}

// newTestRegistry returns a running Registry whose Run loop is stopped when
// the test ends.
func newTestRegistry(t *testing.T) *connection.Registry {
	t.Helper()
	r := connection.NewRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go r.Run(ctx)
	return r
}

// TestConnectionUpgrade verifies that an HTTP request can be upgraded to a WebSocket connection.
func TestConnectionUpgrade(t *testing.T) {
	r := newTestRegistry(t)
	server := httptest.NewServer(http.HandlerFunc(r.RegisterHandler(echoHandler{})))
	defer server.Close()

	ws, resp, err := dialWebSocket(server.URL)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	ws.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("Expected status code %d, got %d", http.StatusSwitchingProtocols, resp.StatusCode)
	}
}

// TestEchoMessage verifies the echo functionality by sending a message and expecting the same message in return.
func TestEchoMessage(t *testing.T) {
	r := newTestRegistry(t)
	server := httptest.NewServer(http.HandlerFunc(r.RegisterHandler(echoHandler{})))
	defer server.Close()

	ws, _, err := dialWebSocket(server.URL)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer ws.Close()

	testMsg := []byte("hello world")
	if err := ws.WriteMessage(websocket.TextMessage, testMsg); err != nil {
		t.Fatal("WriteMessage failed:", err)
	}

	_, message, err := ws.ReadMessage()
	if err != nil {
		t.Fatal("ReadMessage failed:", err)
	}

	if !bytes.Equal(message, testMsg) {
		t.Errorf("Expected message %s, got %s", testMsg, message)
	}
}

// TestGroupBroadcastLifecycle exercises AddToGroup/RemoveFromGroup end to
// end through the public API: a connection only receives group broadcasts
// while it is a member.
type joinHandler struct {
	registry *connection.Registry
	joined   chan struct{}
}

func (h *joinHandler) HandleMessage(conn *connection.Connection, msg []byte) ([]byte, error) {
	h.registry.AddToGroup("room1", conn)
	close(h.joined)
	return nil, nil
}

func TestGroupBroadcastLifecycle(t *testing.T) {
	r := newTestRegistry(t)
	handler := &joinHandler{registry: r, joined: make(chan struct{})}

	server := httptest.NewServer(http.HandlerFunc(r.RegisterHandler(handler)))
	defer server.Close()

	ws, _, err := dialWebSocket(server.URL)
	if err != nil {
		t.Fatalf("Failed to establish WebSocket connection: %v", err)
	}
	defer ws.Close()

	// Trigger the handler above so this connection joins "room1".
	if err := ws.WriteMessage(websocket.TextMessage, []byte("join")); err != nil {
		t.Fatal("WriteMessage failed:", err)
	}
	<-handler.joined

	r.Broadcast(&message.ByteMessage{Data: []byte("hi")}, "room1")

	if _, msg, err := ws.ReadMessage(); err != nil {
		t.Fatalf("expected group broadcast, got error: %v", err)
	} else if string(msg) != "hi" {
		t.Fatalf("got %q, want %q", msg, "hi")
	}
}

// TestCheckOriginRejectsCrossOrigin verifies the default CheckOrigin blocks
// cross-site WebSocket handshakes (CSWSH) instead of allowing everything.
func TestCheckOriginRejectsCrossOrigin(t *testing.T) {
	r := newTestRegistry(t)
	server := httptest.NewServer(http.HandlerFunc(r.RegisterHandler(echoHandler{})))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]
	header := http.Header{"Origin": {"http://evil.example.com"}}

	_, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err == nil {
		t.Fatal("expected handshake to fail for a mismatched Origin")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got resp=%v err=%v", resp, err)
	}
}

// AllowOrigins is the sanctioned alternative to a blanket `return true` for a
// frontend served from another port. It has to keep the safe parts of the
// default - non-browser clients and same-origin - while adding only the exact
// origins named, so a look-alike host is still rejected.
func TestAllowOriginsExtendsTheDefaultWithoutWideningIt(t *testing.T) {
	check := connection.AllowOrigins("http://localhost:5173", " https://App.Example.com ")

	cases := []struct {
		name   string
		origin string
		want   bool
	}{
		{"no Origin header (non-browser client)", "", true},
		{"same origin as the request Host", "http://api.example.com", true},
		{"listed origin", "http://localhost:5173", true},
		{"listed origin, differing case", "https://app.example.com", true},
		{"unlisted origin", "https://evil.example.com", false},
		{"look-alike of a listed origin", "http://localhost:5174", false},
		{"listed host over the wrong scheme", "https://localhost:5173", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "http://api.example.com/ws", nil)
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if got := check(r); got != tc.want {
				t.Errorf("origin %q: allowed=%v, want %v", tc.origin, got, tc.want)
			}
		})
	}
}
