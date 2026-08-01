package connection

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/gorilla/websocket"
)

// Connection wraps a single upgraded WebSocket connection: its outgoing
// message queue, the goroutines that pump data to/from the socket, and the
// set of broadcast groups it currently belongs to.
type Connection struct {
	ID             string // Unique identifier for the connection
	WS             *websocket.Conn
	Send           chan []byte
	wg             sync.WaitGroup
	closeOnce      sync.Once
	messageHandler MessageHandler
	done           chan struct{}   // closed exactly once, by CloseConnection
	groups         map[string]bool // Tracks which groups this connection is part of.
}

func NewConnection(ws *websocket.Conn, handler MessageHandler) *Connection {
	return &Connection{
		ID:             uuid.NewString(), // Assign a unique ID to the connection
		WS:             ws,
		Send:           make(chan []byte, 256),
		messageHandler: handler,
		done:           make(chan struct{}),
		groups:         make(map[string]bool),
	}
}

// CloseConnection closes the underlying WebSocket and signals the read/write
// pumps to stop. It deliberately never closes Send: registry.Broadcast sends
// to Send from other goroutines, and a closed channel panics on send even
// under select/default. The write pump is the only reader of Send, and it
// learns to stop via done instead - so Send is simply left for GC once the
// connection is unregistered. Safe to call more than once or concurrently.
func (c *Connection) CloseConnection() {
	c.closeOnce.Do(func() {
		close(c.done)
		if c.WS != nil {
			c.WS.Close() // Error handling omitted for brevity.
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

// defaultCheckOrigin allows requests with no Origin header (non-browser
// clients, e.g. server-to-server dialers) and requests whose Origin host
// matches the request's own Host (same-origin browser clients). It rejects
// everything else. This is a safe default that blocks cross-site WebSocket
// hijacking (CSWSH) out of the box. If your deployment needs cross-origin
// clients (e.g. a separately-hosted frontend), set Registry.CheckOrigin to a
// function that allows the specific origins you trust - do not blanket-allow
// all origins unless every caller of the endpoint is trusted.
func defaultCheckOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

// RegisterHandler returns an http.HandlerFunc that upgrades incoming
// requests to WebSocket connections tracked by r, dispatching incoming
// messages to customHandler.
func (r *Registry) RegisterHandler(customHandler MessageHandler) http.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     r.CheckOrigin,
	}

	return func(w http.ResponseWriter, req *http.Request) {
		ws, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			log.Println("Upgrade failed:", err)
			return
		}

		// Initialize the connection with the custom handler.
		client := NewConnection(ws, customHandler)

		select {
		case r.register <- client:
		case <-r.stopped:
			client.CloseConnection()
			return
		}
		defer func() {
			select {
			case r.unregister <- client:
			case <-r.stopped:
			}
		}()

		client.wg.Add(2)
		go client.writePump()
		go client.readPump()
		client.wg.Wait()
	}
}
