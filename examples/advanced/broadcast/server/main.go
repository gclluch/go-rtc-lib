package main

import (
	"encoding/json"
	"go-rtc-lib/pkg/connection"
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
func (h *BroadcastHandler) HandleMessage(conn *connection.Connection, message []byte) ([]byte, error) {

	// Construct the message.
	msg := StructuredMessage{
		ID:      conn.ID,
		Message: string(message),
	}

	// Serialize the structured message to JSON.
	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error serializing message: %v", err)
		return nil, err // Handle the error appropriately.
	}

	log.Printf("Broadcasting structured message: %s", jsonData)

	// Use the global registry to broadcast the message.
	// Ensure you're using the modified Broadcast method that accepts a *Message struct.
	globalRegistry := connection.GetGlobalRegistry()
	globalRegistry.Broadcast(&connection.Message{
		Data:      jsonData,
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
