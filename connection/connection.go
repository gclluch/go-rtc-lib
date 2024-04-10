package connection

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/gorilla/websocket"
)

var globalRegistry = NewRegistry()

func init() {
	go globalRegistry.Run()
}

func GetGlobalRegistry() *Registry {
	return globalRegistry
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Implement origin check for production.
		return true
	},
}

type Connection struct {
	ID             string // Unique identifier for the connection
	WS             *websocket.Conn
	Send           chan []byte
	wg             sync.WaitGroup
	closeOnce      sync.Once
	messageHandler MessageHandler
	closeSignal    chan struct{}
	groups         map[string]bool // Tracks which groups this connection is part of.
}

func NewConnection(ws *websocket.Conn, handler MessageHandler) *Connection {
	return &Connection{
		ID:             uuid.NewString(), // Assign a unique ID to the connection
		WS:             ws,
		Send:           make(chan []byte, 256),
		messageHandler: handler,
		closeSignal:    make(chan struct{}),
		groups:         make(map[string]bool),
	}
}

func (c *Connection) CloseConnection() {
	c.closeOnce.Do(func() {
		close(c.Send)
		c.WS.Close() // Error handling omitted for brevity.
		close(c.closeSignal)
		// Remove the connection from all groups it was part of.
		for groupID := range c.groups {
			globalRegistry.RemoveFromGroup(groupID, c)
		}
	})
}

func (c *Connection) setupPongHandler() {
	c.WS.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.WS.SetPongHandler(func(string) error {
		c.WS.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
}

// RegisterHandler creates a new WebSocket connection handler that uses the provided custom handler.
func RegisterHandler(customHandler MessageHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade failed:", err)
			return
		}

		// Initialize the connection with the custom handler.
		client := NewConnection(conn, customHandler)

		globalRegistry.register <- client
		defer func() { globalRegistry.unregister <- client }()

		client.wg.Add(2)
		go client.writePump()
		go client.readPump()
		client.wg.Wait()
	}
}
