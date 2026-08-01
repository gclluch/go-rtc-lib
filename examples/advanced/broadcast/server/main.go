package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/gclluch/go-rtc-lib/connection"
	"github.com/gclluch/go-rtc-lib/message" // Import the message package.
)

// BroadcastHandler defines a handler for WebSocket messages that broadcasts incoming messages to all connected clients.
type BroadcastHandler struct {
	registry *connection.Registry
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

	// Broadcast the message to every connection tracked by this registry.
	h.registry.BroadcastToAll(jsonMsg)

	// No direct response to the sender in this broadcast scenario.
	return nil, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	registry := connection.NewRegistry()
	go registry.Run(ctx)

	broadcastHandler := &BroadcastHandler{registry: registry}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", registry.RegisterHandler(broadcastHandler))

	srv := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	log.Println("Broadcast WebSocket server starting on :8080...")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server failed to start:", err)
	}
}
