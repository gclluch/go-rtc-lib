package main

import (
	"go-rtc-lib/pkg/connection"
	"log"
	"net/http"
)

// Handler defines a custom handler for WebSocket messages.
// It needs to implement the handler.Handler interface.
type Handler struct{}

// HandleMessage now includes a *connection.Connection parameter.
func (h *Handler) HandleMessage(conn *connection.Connection, message []byte) ([]byte, error) {
	log.Printf("Received message: %s", string(message))
	// For this example, we just echo the message back.
	return message, nil
}

func main() {
	handler := &Handler{} // Assume CustomHandler is defined elsewhere

	// Use the new factory function to create a handler with the custom logic
	http.HandleFunc("/ws", connection.RegisterHandler(handler))

	log.Println("WebSocket server starting on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
