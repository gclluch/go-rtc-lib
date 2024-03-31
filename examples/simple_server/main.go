package main

import (
	"go-rtc-lib/internal/connection"
	"log"
	"net/http"
)

// CustomHandler defines a custom handler for WebSocket messages.
// It needs to implement the handler.Handler interface.
type CustomHandler struct{}

// HandleMessage processes incoming WebSocket messages and optionally returns a response.
func (h *CustomHandler) HandleMessage(message []byte) ([]byte, error) {
	log.Printf("Received message: %s", string(message))
	// For this example, let's just echo the message back.
	// In real scenarios, you'd likely do more complex processing here.
	return message, nil
}

func main() {
	customHandler := &CustomHandler{} // Assume CustomHandler is defined elsewhere

	// Use the new factory function to create a handler with the custom logic
	http.HandleFunc("/ws", connection.NewHandler(customHandler))

	log.Println("WebSocket server starting on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
