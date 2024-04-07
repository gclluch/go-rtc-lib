package connection

import (
	"sync"
)

// Registry manages active WebSocket connections and supports broadcasting to groups.
type Registry struct {
	connections map[*Connection]bool            // Global list of all connections
	groups      map[string]map[*Connection]bool // Groups of connections
	broadcast   chan *Message                   // Messages to be broadcasted globally
	register    chan *Connection
	unregister  chan *Connection
	mu          sync.Mutex
}

type Message struct {
	Data      []byte
	GroupName string // If empty, broadcast to all connections; if not, broadcast to the group
}

// NewRegistry creates a new Registry instance.
func NewRegistry() *Registry {
	return &Registry{
		connections: make(map[*Connection]bool),
		groups:      make(map[string]map[*Connection]bool),
		broadcast:   make(chan *Message),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
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
				close(conn.Send)
			}
			r.mu.Unlock()

		case msg := <-r.broadcast:
			r.mu.Lock()
			// Determine whether to send to a group or to all connections
			if msg.GroupName == "" {
				// Broadcast to all connections
				for conn := range r.connections {
					select {
					case conn.Send <- msg.Data:
					default:
						close(conn.Send)
						delete(r.connections, conn)
					}
				}
			} else if group, ok := r.groups[msg.GroupName]; ok {
				// Broadcast to specific group
				for conn := range group {
					select {
					case conn.Send <- msg.Data:
					default:
						close(conn.Send)
						delete(group, conn)
					}
				}
			}
			r.mu.Unlock()
		}
	}
}

// CreateGroup adds a new group for broadcasting messages.
func (r *Registry) CreateGroup(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.groups[name]; !exists {
		r.groups[name] = make(map[*Connection]bool)
	}
}

// DeleteGroup removes a group and closes all connections within it.
func (r *Registry) DeleteGroup(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if group, exists := r.groups[name]; exists {
		for conn := range group {
			delete(r.connections, conn)
			close(conn.Send)
		}
		delete(r.groups, name)
	}
}

// AddToGroup adds a connection to a specific group.
func (r *Registry) AddToGroup(groupName string, conn *Connection) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if group, exists := r.groups[groupName]; exists {
		group[conn] = true
	} else {
		group := make(map[*Connection]bool)
		group[conn] = true
		r.groups[groupName] = group
	}
}

// RemoveFromGroup removes a connection from a specific group.
func (r *Registry) RemoveFromGroup(groupName string, conn *Connection) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if group, exists := r.groups[groupName]; exists {
		if _, ok := group[conn]; ok {
			delete(group, conn)
		}
	}
}

// Broadcast sends a message to all connections or to a specific group.
func (r *Registry) Broadcast(msg *Message) {
	r.broadcast <- msg
}

// ClearConnections removes all connections from the registry. For testing use only.
func (r *Registry) ClearConnections() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for conn := range r.connections {
		delete(r.connections, conn)
		close(conn.Send)
	}
}
