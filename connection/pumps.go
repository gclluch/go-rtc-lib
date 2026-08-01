// pumps.go
package connection

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func (c *Connection) readPump() {
	defer func() {
		c.wg.Done()
		c.CloseConnection() // Ensure connection is closed at the end of readPump.
	}()
	c.setupPongHandler()

	for {
		_, msg, err := c.WS.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			} else {
				log.Printf("read error: %v", err)
			}
			break // Exit the loop on read error.
		}

		if c.messageHandler == nil {
			// Fallback or default behavior if no handler is registered.
			log.Printf("No handler registered. Message received: %s", string(msg))
			continue
		}

		// Process the message using the registered handler.
		response, handlerErr := c.messageHandler.HandleMessage(c, msg)
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
		case <-c.done:
			// Connection is closing (peer went away, or a graceful
			// shutdown); tell the peer and stop pumping.
			c.WS.WriteMessage(websocket.CloseMessage, []byte{})
			return

		case message := <-c.Send:
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
