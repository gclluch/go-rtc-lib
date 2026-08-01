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
	done           chan struct{} // closed exactly once, by CloseConnection

	// writeDone is closed by writePump when it stops, whether or not it managed
	// to put a Close frame on the wire. CloseConnection waits on it briefly so
	// the frame goes out before the socket does. See CloseConnection.
	writeDone chan struct{}

	groups map[string]bool // Tracks which groups this connection is part of.
}

// closeFrameGrace bounds how long CloseConnection waits for the write pump to
// emit its Close frame. It is a courtesy to the peer, not a correctness
// requirement, so it stays short enough never to matter to a shutdown.
const closeFrameGrace = 250 * time.Millisecond

func NewConnection(ws *websocket.Conn, handler MessageHandler) *Connection {
	return &Connection{
		ID:             uuid.NewString(), // Assign a unique ID to the connection
		WS:             ws,
		Send:           make(chan []byte, 256),
		messageHandler: handler,
		done:           make(chan struct{}),
		writeDone:      make(chan struct{}),
		groups:         make(map[string]bool),
	}
}

// CloseConnection closes the underlying WebSocket and signals the read/write
// pumps to stop. It deliberately never closes Send: registry.Broadcast sends
// to Send from other goroutines, and a closed channel panics on send even
// under select/default. The write pump is the only reader of Send, and it
// learns to stop via done instead - so Send is simply left for GC once the
// connection is unregistered. Safe to call more than once or concurrently.
// It closes the socket only after the write pump has had a chance to send a
// Close frame. Closing both at once raced: whether the peer saw a clean 1000 or
// an abnormal 1006 depended on which goroutine the scheduler picked, so a
// deliberate shutdown usually looked like a network fault to the client.
func (c *Connection) CloseConnection() {
	c.closeOnce.Do(func() {
		close(c.done)

		// writePump closes writeDone on its way out, so this returns as soon as
		// the frame is written - or immediately when called from writePump's own
		// defer. The timeout covers the case where no write pump ever ran.
		select {
		case <-c.writeDone:
		case <-time.After(closeFrameGrace):
		}

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

// AllowOrigins returns a CheckOrigin that accepts everything the default does
// (no Origin header, or same-origin) plus the origins listed. Reach for it when
// the frontend is served from somewhere other than the API - a Vite dev server
// on another port, say - rather than reaching for a blanket `return true`:
//
//	registry.CheckOrigin = connection.AllowOrigins("http://localhost:5173")
//
// Origins are matched whole and case-insensitively, so pass exact scheme+host
// values ("https://app.example.com"). There is no wildcard on purpose.
func AllowOrigins(origins ...string) func(r *http.Request) bool {
	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins {
		if origin = strings.TrimSpace(origin); origin != "" {
			allowed[strings.ToLower(origin)] = true
		}
	}

	return func(r *http.Request) bool {
		if defaultCheckOrigin(r) {
			return true
		}
		return allowed[strings.ToLower(r.Header.Get("Origin"))]
	}
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
