package connection

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/gorilla/websocket"
)

type MessageHandler interface {
	HandleMessage(conn *Connection, msg []byte) ([]byte, error)
}

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

// Adjust the pong handler and ping sender to maintain the connection
func (c *Connection) setupPongHandler() {
	c.WS.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.WS.SetPongHandler(func(string) error {
		c.WS.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
}

func (c *Connection) readPump() {
	defer func() {
		c.wg.Done()
		c.CloseConnection() // Ensure connection is closed at the end of readPump.
	}()
	c.setupPongHandler()

	for {
		_, message, err := c.WS.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			} else {
				log.Printf("read error: %v", err)
			}
			break // Exit the loop on read error.
		}

		if c.messageHandler != nil {
			// Process the message using the registered handler.
			response, handlerErr := c.messageHandler.HandleMessage(c, message)
			if handlerErr != nil {
				log.Printf("Handler error: %v", handlerErr)
				// Optionally, close the connection on handler error.
				break
			}
			if response != nil {
				// Send response if not blocked.
				select {
				case c.Send <- response:
				default:
					// Log or handle blocked send channel.
					log.Println("Send channel blocked. Unable to send handler response.")
				}
			}
		} else {
			// Fallback or default behavior if no handler is registered.
			log.Printf("No handler registered. Message received: %s", string(message))
			// Echo the message back or handle as needed.
		}
	}
}

func (c *Connection) writePump() {
	ticker := time.NewTicker(30 * time.Second) // Adjust the interval as needed.
	defer func() {
		ticker.Stop()
		c.CloseConnection() // Ensure connection is closed at the end of writePump.
		c.wg.Done()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				// The channel has been closed.
				c.WS.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.WS.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.WS.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Write error: %v", err)
				return
			}

		case <-ticker.C:
			c.WS.SetWriteDeadline(time.Now().Add(10 * time.Second))
			// Send a ping message.
			if err := c.WS.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping error: %v", err)
				return
			}
		}
	}
}

func Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := NewConnection(conn, nil) // Assign a default handler if needed.
	// Register the connection with the global registry.
	globalRegistry.register <- client
	defer func() { globalRegistry.unregister <- client }()

	// If there are query parameters to specify groups, join those groups.
	groups := r.URL.Query()["group"]
	for _, groupID := range groups {
		globalRegistry.AddToGroup(groupID, client)
		defer globalRegistry.RemoveFromGroup(groupID, client) // Ensure we leave the group on disconnect.
	}

	client.wg.Add(2)
	go client.writePump()
	go client.readPump()
	client.wg.Wait()
}

func RegisterHandler(customHandler MessageHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade failed:", err)
			return
		}
		client := NewConnection(conn, customHandler) // Initialize the connection with the custom handler.
		globalRegistry.register <- client
		defer func() { globalRegistry.unregister <- client }()

		client.wg.Add(2)
		go client.writePump()
		go client.readPump()
		client.wg.Wait()
	}
}
