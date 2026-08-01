# go-rtc-lib - Real-Time Communication Library for Go

`go-rtc-lib` is a Go package designed to facilitate real-time, bidirectional communication between servers and clients. Built on top of WebSockets, it aims to simplify the creation of real-time applications like chat systems, live updates, and multiplayer games by handling the complexities of connection management, data transmission, and more.

## Features

- **WebSockets Management:** Simplifies establishing and maintaining WebSocket connections.
- **Group Support:** Facilitates creating and managing groups (or rooms) for targeted message broadcasting, allowing for more organized communication channels.
- **Broadcasting:** Supports broadcasting messages to all connected clients.
- **Data Handling:** Seamlessly handles different types of data (JSON, binary, etc.).
- **Custom Message Handlers:** Supports custom message handling logic to accommodate specific application requirements.
- **Graceful Shutdown:** A `Registry` is driven by a `context.Context`; canceling it drains and closes every connection.

## Getting Started

### Prerequisites
- Go 1.22 or later

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
	"context"
	"log"
	"net/http"

	"github.com/gclluch/go-rtc-lib/connection"
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
	// A Registry tracks connections and groups. It's driven by a context, so
	// canceling ctx drains and closes every connection cleanly.
	registry := connection.NewRegistry()
	go registry.Run(context.Background())

	handler := &Handler{}
	http.HandleFunc("/ws", registry.RegisterHandler(handler))

	log.Println("WebSocket server starting on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
```

See `examples/basic/server` for the same example wired up with `signal.NotifyContext` for a graceful shutdown on Ctrl-C / SIGTERM.

## Advanced Usage

Detailed examples found in `examples/advanced/`

### Broadcasting to All Connections vs Groups

```go
// Broadcast to all connections tracked by the registry.
registry.BroadcastToAll(msg)

// Broadcast to a single group.
registry.Broadcast(jsonMsg, groupName)
```

### Custom Message Types

You can create custom message types to enhance the flexibility and efficiency of data handling, allowing for structured and meaningful communication tailored to specific application needs. To create a custom message type, implement the `IMessage` interface. For example, a `ChatMessage` might look like this:

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

The `Broadcast` method will automatically serialize the message if the message type has been properly defined.

```go
chatMsg := &message.ChatMessage{
	Sender:  "server",
	Content: "Welcome to the chat room!",
}
registry.Broadcast(chatMsg, "") // "" for `groupName` broadcasts to all clients.
```

### Origin Checking

By default, `Registry` only accepts WebSocket upgrades from the same origin as the request's `Host` (or requests with no `Origin` header at all, e.g. non-browser clients). This blocks cross-site WebSocket hijacking (CSWSH) out of the box. If your frontend is hosted on a different origin than your API, set `Registry.CheckOrigin` to a function that allows the specific origins you trust:

```go
registry := connection.NewRegistry()
registry.CheckOrigin = func(r *http.Request) bool {
	return r.Header.Get("Origin") == "https://app.example.com"
}
```

Do not blanket-allow all origins (`return true`) unless every caller of the endpoint is trusted.

## Why this over gorilla/websocket or melody?

It isn't a production-scale alternative to either. `go-rtc-lib` is a small, readable hub built directly on `gorilla/websocket` - roughly 700 lines including examples - that you can read start to finish in one sitting and modify to fit your app. If you need battle-tested scale, more configuration knobs, or an actively maintained ecosystem, reach for `gorilla/websocket` directly or a more mature framework. Reach for this when you'd rather own and understand every line of your connection-management code than pull in something bigger than you need.

## License

`go-rtc-lib` is released under the MIT License. See the LICENSE file for more details.
