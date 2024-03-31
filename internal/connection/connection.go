package connection

import (
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
	WS        *websocket.Conn
	Send      chan []byte
	wg        sync.WaitGroup // Use a WaitGroup to manage goroutine lifecycle.
	closeOnce sync.Once      // Ensure connection is closed exactly once
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
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("unexpected close error: %v", err)
			} else {
				log.Printf("read error: %v", err)
			}
			break
		}
		log.Printf("recv: %s", message)
		// Echo the message back for testing purposes
		c.Send <- message
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
