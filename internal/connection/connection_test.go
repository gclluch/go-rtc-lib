package connection_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-rtc-lib/internal/connection"

	"github.com/gorilla/websocket"
)

// MockHandler implements the handler.Handler interface for testing.
type MockHandler struct{}

func (m *MockHandler) HandleMessage(msg []byte) ([]byte, error) {
	// Echo the message back to the client
	return msg, nil
}

// dialWebSocket helps in establishing a WebSocket connection for testing.
func dialWebSocket(serverURL string) (*websocket.Conn, *http.Response, error) {
	wsURL := "ws" + serverURL[len("http"):]
	return websocket.DefaultDialer.Dial(wsURL, nil)
}

// TestConnectionUpgrade verifies that an HTTP request can be upgraded to a WebSocket connection.
func TestConnectionUpgrade(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(connection.Handler))
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		customHandler := &MockHandler{}
		connection.RegisterHandler(customHandler)(w, r)
	}))
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

// Add more tests here, such as TestBroadcast functionality, ensuring messages are received by all connected clients.
