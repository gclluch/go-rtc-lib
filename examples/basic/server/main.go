package main

import (
	"go-rtc-lib/connection"
	"log"
	"net/http"
)

// Handler defines a custom handler for WebSocket messages.
// It needs to implement the MessageHandler interface.
type Handler struct{}

// HandleMessage is the method where the custom logic for handling messages is defined.
func (h *Handler) HandleMessage(conn *connection.Connection, message []byte) ([]byte, error) {
	log.Printf("Received message: %s", string(message))
	// For this example, we just echo the message back.
	return message, nil
}

func main() {
	handler := &Handler{}

	// Use the registration function to create a handler with the custom logic
	http.HandleFunc("/ws", connection.RegisterHandler(handler))

	log.Println("WebSocket server starting on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
