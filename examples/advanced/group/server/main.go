package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/gclluch/go-rtc-lib/connection"
	"github.com/gclluch/go-rtc-lib/message"
)

// Custom WebSocket message handler that supports join/leave group operations and messaging
type GroupMessageHandler struct {
	registry *connection.Registry
}

func (h *GroupMessageHandler) HandleMessage(conn *connection.Connection, msg []byte) ([]byte, error) {
	// Parse the incoming JSON from client.
	var parsedMsg struct {
		Action  string `json:"action"`
		Group   string `json:"group,omitempty"`
		Message string `json:"message,omitempty"`
	}
	if err := json.Unmarshal(msg, &parsedMsg); err != nil {
		log.Printf("Error parsing message: %v", err)
		return nil, err
	}

	// Handle the message based on its action type.
	switch parsedMsg.Action {
	case "join":
		// Join the specified group.
		h.registry.AddToGroup(parsedMsg.Group, conn)
		log.Printf("Connection %s joined group %s", conn.ID, parsedMsg.Group)
	case "leave":
		// Leave the specified group.
		h.registry.RemoveFromGroup(parsedMsg.Group, conn)
		log.Printf("Connection %s left group %s", conn.ID, parsedMsg.Group)
	case "message":
		// Broadcast the message to the group.
		h.broadcastMessage(conn, parsedMsg.Group, parsedMsg.Message)
	default:
		log.Printf("Unknown action: %s", parsedMsg.Action)
	}

	return nil, nil
}

// Broadcasts a structured message to all connections in the specified group.
func (h *GroupMessageHandler) broadcastMessage(conn *connection.Connection, groupName, messageContent string) {
	msgData := map[string]string{
		"from":    conn.ID,        // Sender ID
		"message": messageContent, // The message text
	}

	// Create a new JSONMessage instance with the content.
	jsonMsg := message.NewJSONMessage(msgData)
	log.Printf("Broadcasting structured message: %+v", jsonMsg)

	// Broadcast the JSON message to the specified group.
	h.registry.Broadcast(jsonMsg, groupName)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	registry := connection.NewRegistry()
	go registry.Run(ctx)

	handler := &GroupMessageHandler{registry: registry}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", registry.RegisterHandler(handler))

	srv := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	log.Println("Server started on :8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}
