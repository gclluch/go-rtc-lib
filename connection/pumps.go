// pumps.go
package connection

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// maxMessageBytes caps a single inbound frame. See readPump.
const maxMessageBytes = 1 << 20 // 1 MiB

func (c *Connection) readPump() {
	defer func() {
		c.wg.Done()
		c.CloseConnection() // Ensure connection is closed at the end of readPump.
	}()
	// Without a read limit a single peer can force an unbounded allocation with
	// one oversized frame. 1 MiB is generous for control/JSON traffic; raise it
	// deliberately if an application needs larger payloads.
	c.WS.SetReadLimit(maxMessageBytes)
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
		// Signal before CloseConnection, unconditionally: this is what lets
		// CloseConnection stop waiting, and it has to fire even on the write-error
		// paths below where no Close frame was ever sent. Closing it here rather
		// than after the frame keeps the ordering obvious - once writePump is
		// done, nothing else will ever write to this socket.
		close(c.writeDone)
		c.CloseConnection() // Ensure connection is closed at the end of writePump.
		c.wg.Done()
	}()

	for {
		select {
		case <-c.done:
			// Connection is closing (peer went away, or a graceful
			// shutdown); tell the peer and stop pumping. The deadline matters:
			// without it a wedged peer can block this write forever and strand
			// the goroutine.
			c.WS.SetWriteDeadline(time.Now().Add(10 * time.Second))
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
