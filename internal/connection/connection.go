package connection

import (
	"go-rtc-lib/internal/handler"

	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Adjust this for production to validate the origin.
	},
}

type Connection struct {
	WS             *websocket.Conn
	Send           chan []byte
	wg             sync.WaitGroup // Use a WaitGroup to manage goroutine lifecycle.
	closeOnce      sync.Once      // Ensure connection is closed exactly once
	messageHandler handler.Handler
}

// RegisterHandler allows users to set a custom message handler for the connection.
func (c *Connection) RegisterHandler(handler handler.Handler) {
	c.messageHandler = handler
}

func (c *Connection) closeConnection() {
	c.closeOnce.Do(func() {
		if err := c.WS.Close(); err != nil {
			log.Printf("close connection error: %v", err)
		}
	})
}

func (c *Connection) readPump() {
	defer func() {
		c.closeConnection()
		c.wg.Done()
	}()
	c.WS.SetReadLimit(512)
	c.WS.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.WS.SetPongHandler(func(string) error {
		c.WS.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, message, err := c.WS.ReadMessage()
		if err != nil {
			log.Printf("read error: %v", err)
			break
		}

		// Check if a messageHandler is set, then use it
		if c.messageHandler != nil {
			response, err := c.messageHandler.HandleMessage(message)
			if err != nil {
				log.Printf("handler error: %v", err)
				// Decide how to handle handler errors; maybe close the connection
				break
			}
			if response != nil {
				// Optionally, send a response back to the client
				c.Send <- response
			}
		} else {
			// Fallback or default behavior if no handler is registered
			log.Printf("recv: %s", message)
			c.Send <- message // Echo back for testing purposes
		}
	}
}

func (c *Connection) writePump() {
	ticker := time.NewTicker(60 * time.Second)
	defer func() {
		ticker.Stop()
		c.closeConnection()
		c.wg.Done()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.WS.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.WS.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("write error: %v", err)
				return
			}
		case <-ticker.C:
			c.WS.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.WS.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("ping error: %v", err)
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
	client.wg.Add(2) // Add two goroutines to the WaitGroup.
	go client.writePump()
	go client.readPump()
	client.wg.Wait()   // Wait for both pumps to finish.
	close(client.Send) // Safely close the send channel after pumps finish.
}

// Create a function to generate a new WebSocket handler with a custom message handler.
func NewHandler(customHandler handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Upgrade failed:", err)
			return
		}
		client := &Connection{WS: conn, Send: make(chan []byte, 256)}
		client.RegisterHandler(customHandler) // Register the custom handler

		client.wg.Add(2)
		go client.writePump()
		go client.readPump()
		client.wg.Wait()
		close(client.Send)
	}
}
