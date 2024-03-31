package connection

import (
	"go-rtc-lib/internal/handler"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var globalRegistry = NewRegistry()

func init() {
	go globalRegistry.Run()
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
	WS             *websocket.Conn
	Send           chan []byte
	wg             sync.WaitGroup
	closeOnce      sync.Once
	messageHandler handler.Handler
	closeSignal    chan struct{}
}

func NewConnection(ws *websocket.Conn, handler handler.Handler) *Connection {
	return &Connection{
		WS:             ws,
		Send:           make(chan []byte, 256),
		messageHandler: handler,
		closeSignal:    make(chan struct{}),
	}
}

func (c *Connection) CloseConnection() {
	c.closeOnce.Do(func() {
		close(c.Send)
		if err := c.WS.Close(); err != nil {
			log.Printf("Error closing WebSocket connection: %v", err)
		}
		close(c.closeSignal)
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
			response, handlerErr := c.messageHandler.HandleMessage(message)
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
	client := &Connection{WS: conn, Send: make(chan []byte, 256)}
	globalRegistry.register <- client
	defer func() { globalRegistry.unregister <- client }()

	client.wg.Add(2)
	go client.writePump()
	go client.readPump()
	client.wg.Wait()
}

func RegisterHandler(customHandler handler.Handler) http.HandlerFunc {
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
