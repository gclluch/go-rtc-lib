package main

import (
	"go-rtc-lib/internal/connection"
	"log"
	"net/http"
)

// BroadcastHandler defines a handler for WebSocket messages that broadcasts incoming messages to all connected clients.
type BroadcastHandler struct{}

// HandleMessage broadcasts the received message to all connected clients.
func (h *BroadcastHandler) HandleMessage(message []byte) ([]byte, error) {
	// Broadcast the message to all connected clients
	connection.GetGlobalRegistry().Broadcast(message)
	// No response to the sender
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
