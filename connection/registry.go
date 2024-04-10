package connection

import (
	"go-rtc-lib/message"
	"log"
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
			delete(r.connections, conn)
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

// BroadcastToAll sends a message to all connections.
func (r *Registry) BroadcastToAll(msg message.IMessage) {
	r.Broadcast(msg, "")
}

// Broadcast sends a message to all connections or to a specific group.
func (r *Registry) Broadcast(msg message.IMessage, groupName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	serializedMsg, err := msg.Serialize()
	if err != nil {
		log.Printf("Error serializing message: %v", err)
		return
	}

	// Identify the correct set of connections based on groupName.
	var targetConnections map[*Connection]bool
	if groupName == "" {
		targetConnections = r.connections
	} else if group, ok := r.groups[groupName]; ok {
		targetConnections = group
	} else {
		log.Printf("Group %s not found.", groupName)
		return
	}

	// Iterate over connections and send the message.
	for conn := range targetConnections {
		select {
		case conn.Send <- serializedMsg:
			// Message successfully queued to send.
		default:
			log.Printf("Failed to send to connection %s. Channel full or closed.", conn.ID)
		}
	}
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
