package main

import (
	"encoding/json"
	"go-rtc-lib/pkg/connection"
	"go-rtc-lib/pkg/message"
	"log"
	"net/http"
)

// Custom WebSocket message handler that supports join/leave group operations and messaging
type GroupMessageHandler struct{}

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
		connection.GetGlobalRegistry().AddToGroup(parsedMsg.Group, conn)
		log.Printf("Connection %s joined group %s", conn.ID, parsedMsg.Group)
	case "leave":
		// Leave the specified group.
		connection.GetGlobalRegistry().RemoveFromGroup(parsedMsg.Group, conn)
		log.Printf("Connection %s left group %s", conn.ID, parsedMsg.Group)
	case "message":
		// Broadcast the message to the group.
		broadcastMessage(conn, parsedMsg.Group, parsedMsg.Message)
	default:
		log.Printf("Unknown action: %s", parsedMsg.Action)
	}

	return nil, nil
}

// Broadcasts a structured message to all connections in the specified group.
func broadcastMessage(conn *connection.Connection, groupName, messageContent string) {
	msgData := map[string]string{
		"from":    conn.ID,        // Sender ID
		"message": messageContent, // The message text
	}

	// Create a new JSONMessage instance with the content.
	jsonMsg := message.NewJSONMessage(msgData)
	log.Printf("Broadcasting structured message: %+v", jsonMsg)

	// Broadcast the JSON message to the specified group.
	globalRegistry := connection.GetGlobalRegistry()
	globalRegistry.Broadcast(jsonMsg, groupName)
}

func main() {
	handler := &GroupMessageHandler{}
	http.HandleFunc("/ws", connection.RegisterHandler(handler))

	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
