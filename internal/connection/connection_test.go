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

	url := "ws" + server.URL[len("http"):]

	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Could not open WebSocket connection: %v", err)
	}
	defer ws.Close()

	testMessage := []byte("hello")
	if err := ws.WriteMessage(websocket.TextMessage, testMessage); err != nil {
		t.Fatalf("Could not send message over WebSocket connection: %v", err)
	}

	// Setting a read deadline to ensure the test does not hang
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))

	_, message, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Could not read message from WebSocket connection: %v", err)
	}

	if string(message) != string(testMessage) {
		t.Errorf("Expected message %s, got %s", testMessage, message)
	}
	// Additional logic here to ensure the server has time to process and respond
	time.Sleep(time.Second) // Example of giving extra time
}
