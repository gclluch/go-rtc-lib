# go-rtc-lib - Real-Time Communication Library for Go

`go-rtc-lib` is a Go package designed to facilitate real-time, bidirectional communication between servers and clients. Built on top of WebSockets, it aims to simplify the creation of real-time applications like chat systems, live updates, and multiplayer games by handling the complexities of connection management, data transmission, and more.

## Features

- **WebSockets Management:** Simplifies establishing and maintaining WebSocket connections.
- **Connection Pooling:** Efficiently manages and pools active connections.
- **Group Support:** Facilitates creating and managing groups (or rooms) for targeted message broadcasting, allowing for more organized communication channels.
- **Broadcasting:** Supports broadcasting messages to all connected clients.
- **Data Handling:** Seamlessly handles different types of data (JSON, binary, etc.).
- **Custom Message Handlers:** Supports custom message handling logic to accommodate specific application requirements.


## Getting Started

### Prerequisties 
- Go 1.13 or later

### Installation

To use `go-rtc-lib` in your Go project, run:

```bash
go get github.com/gclluch/go-rtc-lib
```

### Basic Usage

Here's a simple example of how to create a WebSocket server using go-rtc-lib:

```go
package main

import (
	"log"
	"net/http"
	"github.com/gclluch /go-rtc-lib/pkg/connection"
)

// Handler defines a type that will implement the MessageHandler interface.
type Handler struct{}

// HandleMessage is the method where the custom logic for handling messages is defined.
// This method makes Handler adhere to the MessageHandler interface.
func (h *Handler) HandleMessage(conn *connection.Connection, message []byte) ([]byte, error) {
    log.Printf("Received message: %s", string(message))
    // Echo the message back
    return message, nil
}

func main() {
    // Create an instance of Handler
    handler := &Handler{}

    // Use RegisterHandler to create a handler with custom logic
	http.HandleFunc("/ws", connection.RegisterHandler(customHandler))

	log.Println("WebSocket server starting on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
```

## Advanced Usage

Detailed examples found in `examples/advanced/`

# Broadcasting to All connections vs Groups

```go

// Broadcast the JSON message to the specified group.
globalRegistry := connection.GetGlobalRegistry()

// Broadcast to all connections
globalRegistry.BroadcastToAll(msg)

// Broadcast to group
globalRegistry.Broadcast(jsonMsg, groupName) // empty groupName broadcasts to al
```

# Custom Message Types

To create a custom message type, implement the `IMessage` interface. For example, a `ChatMessage` might look like this:

```go
package message

import (
	"encoding/json"
)

// ChatMessage represents a chat message.
type ChatMessage struct {
	Sender  string `json:"sender"`
	Content string `json:"content"`
}

// Serialize converts the ChatMessage into a JSON byte slice.
func (cm *ChatMessage) Serialize() ([]byte, error) {
	return json.Marshal(cm)
}

// Deserialize populates the ChatMessage fields from a byte slice.
func (cm *ChatMessage) Deserialize(data []byte) error {
	return json.Unmarshal(data, cm)
}

// Type returns the type of the ChatMessage.
func (cm *ChatMessage) Type() string {
	return "chat"
}
```

The `Broadcast` method should automatically serialize the message.

```go
chatMsg := &message.ChatMessage{
	Sender:  "server",
	Content: "Welcome to the chat room!",
}
globalRegistry.Broadcast(chatMsg, "") // Broadcast to all clients
```

## Contributing

We welcome contributions from the community! If you'd like to contribute to `go-rtc-lib`, please see our contributing guidelines for more information.

## License

`go-rtc-lib` is released under the MIT License. See the LICENSE file for more details.