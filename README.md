# go-rtc-lib - Real-Time Communication Library for Go

`go-rtc-lib` is a Go package designed to facilitate real-time, bidirectional communication between servers and clients. Built on top of WebSockets, it aims to simplify the creation of real-time applications like chat systems, live updates, and multiplayer games by handling the complexities of connection management, data transmission, and more.

## Features

- **WebSockets Management:** Simplifies establishing and maintaining WebSocket connections.
- **Connection Pooling:** Efficiently manages and pools active connections.
- **Broadcasting:** Supports broadcasting messages to all connected clients.
- **Data Handling:** Seamlessly handles different types of data (text, binary, etc.).
- **Middleware Support:** Easily extend functionality with custom middleware.

## Getting Started

### Installation

To use `go-rtc-lib` in your Go project, run:

```bash
go get github.com/yourusername/go-rtc-lib
```

### Basic Usage

Here's a simple example of how to create a WebSocket server using go-rtc-lib:

```go
package main

import (
    "github.com/yourusername/go-rtc-lib/pkg/rtc"
    "log"
    "net/http"
)

func main() {
    wsServer := rtc.NewServer()

    wsServer.OnConnect(func(c *rtc.Client) {
        log.Printf("Client connected: %s", c.ID)
    })

    wsServer.OnMessage(func(c *rtc.Client, msgType int, msg []byte) {
        // Echo the message back to the client
        c.Send(msgType, msg)
    })

    http.HandleFunc("/ws", wsServer.HandleWebSocket)
    log.Println("WebSocket server starting on port :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

This example demonstrates setting up a WebSocket server that echoes messages back to clients.

## Documentation

For more detailed documentation, including API reference and advanced usage, please refer to pkg.go.dev/github.com/yourusername/go-rtc-lib.

## Contributing

We welcome contributions from the community! If you'd like to contribute to go-rtc-lib, please see our contributing guidelines for more information.

## License

go-rtc-lib is released under the MIT License. See the LICENSE file for more details.