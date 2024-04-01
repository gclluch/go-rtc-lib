package connection

import (
	"sync"
)

// Registry manages active WebSocket connections and supports broadcasting.
type Registry struct {
	connections map[*Connection]bool
	broadcast   chan []byte
	register    chan *Connection
	unregister  chan *Connection
	mu          sync.Mutex
}

// NewRegistry creates a new Registry instance.
func NewRegistry() *Registry {
	return &Registry{
		broadcast:   make(chan []byte),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		connections: make(map[*Connection]bool),
	}
}

// Run starts the registry's main loop, handling connection registration, unregistration, and broadcasting.
func (r *Registry) Run() {
	for {
		select {
		case conn := <-r.register:
			r.mu.Lock()
			r.connections[conn] = true
			r.mu.Unlock()
		case conn := <-r.unregister:
			r.mu.Lock()
			if _, ok := r.connections[conn]; ok {
				delete(r.connections, conn)
				close(conn.Send) // Adjust based on your closing logic
			}
			r.mu.Unlock()
		case message := <-r.broadcast:
			r.mu.Lock()
			for conn := range r.connections {
				select {
				case conn.Send <- message:
				default:
					close(conn.Send) // Adjust based on your closing logic
					delete(r.connections, conn)
				}
			}
			r.mu.Unlock()
		}
	}
}

// ClearConnections removes all connections from the registry. For testing use only.
func (r *Registry) ClearConnections() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for conn := range r.connections {
		delete(r.connections, conn)
		close(conn.Send) // Ensure graceful closure of all connection send channels
	}
}

// Broadcast sends a message to all registered connections.
func (r *Registry) Broadcast(message []byte) {
	r.broadcast <- message
}
