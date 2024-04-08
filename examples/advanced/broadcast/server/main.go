package main

import (
	"go-rtc-lib/pkg/connection"
	"go-rtc-lib/pkg/message" // Import the message package.

	"log"
	"net/http"
)

// BroadcastHandler defines a handler for WebSocket messages that broadcasts incoming messages to all connected clients.
type BroadcastHandler struct{}

// Example of a structured message to be bradcasted.
type StructuredMessage struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// HandleMessage broadcasts the received message to all connected clients.
func (h *BroadcastHandler) HandleMessage(conn *connection.Connection, msg []byte) ([]byte, error) {

	// Assume msg is a JSON string; construct a structured message including the sender ID.
	structuredMsg := map[string]string{
		"id":      conn.ID,
		"message": string(msg),
	}

	// Create a new JSONMessage instance with the structured message.
	jsonMsg := message.NewJSONMessage(structuredMsg)
	log.Printf("Broadcasting structured message: %s", jsonMsg)

	// Use the global registry to broadcast the message.
	globalRegistry := connection.GetGlobalRegistry()
	groupName := "" // Indicates broadcasting to all groups or use BroadcastToAll
	globalRegistry.Broadcast(jsonMsg, groupName)

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
