package main

import (
	"go-rtc-lib/pkg/connection"
	"log"
	"net/http"
)

// BroadcastHandler defines a handler for WebSocket messages that broadcasts incoming messages to all connected clients.
type BroadcastHandler struct{}

// HandleMessage broadcasts the received message to all connected clients.
func (h *BroadcastHandler) HandleMessage(conn *connection.Connection, message []byte) ([]byte, error) {
	log.Printf("Broadcasting message: %s", string(message))
	// Prepare the message to include the sender's ID or any other required information.
	// Since this is a broadcast, we might simply forward the message as is, or prepend/append information.
	msgWithSender := []byte(conn.ID + ": " + string(message))

	// Use the global registry to broadcast the message.
	// Ensure you're using the modified Broadcast method that accepts a *Message struct.
	globalRegistry := connection.GetGlobalRegistry()
	globalRegistry.Broadcast(&connection.Message{
		Data:      msgWithSender,
		GroupName: "", // Leave empty to broadcast to all connections; specify a group name to target a group.
	})

	// No direct response to the sender in this broadcast scenario.
	return nil, nil
}
func main() {
	broadcastHandler := &BroadcastHandler{}

	http.HandleFunc("/ws", connection.RegisterHandler(broadcastHandler))

	log.Println("Broadcast WebSocket server starting on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
