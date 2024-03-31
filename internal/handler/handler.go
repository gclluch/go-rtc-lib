// internal/handler/handler.go
package handler

// Handler defines the interface for processing incoming WebSocket messages.
type Handler interface {
	HandleMessage(msg []byte) ([]byte, error)
}
