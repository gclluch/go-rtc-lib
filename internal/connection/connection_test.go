package connection

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketConnectionUpgrade(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(Handler)) // Setup HTTP server with Handler
	defer server.Close()

	url := "ws" + server.URL[len("http"):] // Convert http://127.0.0.1 to ws://127...

	ws, _, err := websocket.DefaultDialer.Dial(url, nil) // Dial to the server
	if err != nil {
		t.Fatalf("Could not open WebSocket connection: %v", err)
	}
	defer ws.Close()

	testMessage := []byte("hello") // Test sending and receiving a message
	if err := ws.WriteMessage(websocket.TextMessage, testMessage); err != nil {
		t.Fatalf("Could not send message over WebSocket connection: %v", err)
	}

	ws.SetReadDeadline(time.Now().Add(5 * time.Second)) // Set a read deadline to prevent hanging
	_, message, err := ws.ReadMessage()                 // Read message
	if err != nil {
		t.Fatalf("Could not read message from WebSocket connection: %v", err)
	}

	if string(message) != string(testMessage) { // Verify message
		t.Errorf("Expected message %s, got %s", testMessage, message)
	}
}
