package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/gclluch/go-rtc-lib/connection"
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	registry := connection.NewRegistry()
	go registry.Run(ctx)

	handler := &Handler{}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", registry.RegisterHandler(handler))

	srv := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	log.Println("WebSocket server starting on :8080...")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server failed to start:", err)
	}
}
