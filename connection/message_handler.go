// message_handler.go
package connection

// MessageHandler defines the interface for processing incoming WebSocket messages.
type MessageHandler interface {
	HandleMessage(conn *Connection, msg []byte) ([]byte, error)
}
