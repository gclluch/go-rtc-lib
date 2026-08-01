package connection

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/gclluch/go-rtc-lib/message"
)

// Registry manages active WebSocket connections and supports broadcasting to groups.
type Registry struct {
	connections map[*Connection]bool            // Global list of all connections
	groups      map[string]map[*Connection]bool // Groups of connections
	register    chan *Connection
	unregister  chan *Connection
	mu          sync.Mutex

	// stopped is closed when Run returns. Handler goroutines select on it so a
	// connection that outlives Run doesn't park forever sending to unregister.
	stopped  chan struct{}
	stopOnce sync.Once

	// CheckOrigin decides whether an incoming upgrade request's Origin is
	// allowed. It defaults to same-origin-only (see defaultCheckOrigin).
	// Override it to allow specific additional origins.
	CheckOrigin func(r *http.Request) bool
}

// NewRegistry creates a new Registry instance. Each Registry is independent,
// so a process can run multiple isolated servers (or one per test).
func NewRegistry() *Registry {
	return &Registry{
		connections: make(map[*Connection]bool),
		groups:      make(map[string]map[*Connection]bool),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		stopped:     make(chan struct{}),
		CheckOrigin: defaultCheckOrigin,
	}
}

// Run processes connection registration and unregistration until ctx is
// canceled. On cancellation it closes every active connection (draining the
// server) and returns. Callers start it with `go registry.Run(ctx)`.
func (r *Registry) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			r.closeAll()
			// Release any handler blocked on the unregister send. The registry
			// is being torn down, so there is nothing left to unregister from.
			r.stopOnce.Do(func() { close(r.stopped) })
			return

		case conn := <-r.register:
			r.mu.Lock()
			r.connections[conn] = true
			r.mu.Unlock()

		case conn := <-r.unregister:
			r.unregisterConnection(conn)
		}
	}
}

// unregisterConnection removes conn from the registry and from every group
// it had joined. It's called once a connection's pumps have exited for good.
func (r *Registry) unregisterConnection(conn *Connection) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.connections, conn)
	for groupName := range conn.groups {
		if group, exists := r.groups[groupName]; exists {
			delete(group, conn)
		}
	}
	conn.groups = nil
}

// closeAll closes and removes every currently registered connection.
func (r *Registry) closeAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for conn := range r.connections {
		delete(r.connections, conn)
		conn.CloseConnection()
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

	group, exists := r.groups[name]
	if !exists {
		return
	}
	for conn := range group {
		delete(r.connections, conn)
		conn.CloseConnection()
	}
	delete(r.groups, name)
}

// AddToGroup adds a connection to a specific group. Adding a connection the
// registry has already unregistered is a no-op.
func (r *Registry) AddToGroup(groupName string, conn *Connection) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// unregisterConnection nils this map once a connection's pumps have exited.
	// Writing to a nil map panics, and the panic is unrecoverable - so an
	// application that calls AddToGroup on a peer that just disconnected would
	// take the whole process down. Re-adding it would be wrong anyway: nothing
	// will unregister it a second time, so it would sit in the group forever.
	if conn.groups == nil {
		log.Printf("AddToGroup: connection %s is unregistered; not adding to %q", conn.ID, groupName)
		return
	}

	group, exists := r.groups[groupName]
	if !exists {
		group = make(map[*Connection]bool)
		r.groups[groupName] = group
	}
	group[conn] = true
	conn.groups[groupName] = true
}

// RemoveFromGroup removes a connection from a specific group.
func (r *Registry) RemoveFromGroup(groupName string, conn *Connection) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if group, exists := r.groups[groupName]; exists {
		delete(group, conn)
	}
	delete(conn.groups, groupName)
}

// BroadcastToAll sends a message to all connections.
func (r *Registry) BroadcastToAll(msg message.IMessage) {
	r.Broadcast(msg, "")
}

// Broadcast sends a message to all connections or to a specific group.
//
// Serialization and the fan-out both happen outside the registry lock. Holding
// it across them serialized every register, unregister, and group operation
// behind whichever broadcast happened to be running - with N connections and a
// non-trivial Serialize, that is the whole server's throughput ceiling. The
// lock now covers only the snapshot of who to send to.
func (r *Registry) Broadcast(msg message.IMessage, groupName string) {
	serializedMsg, err := msg.Serialize()
	if err != nil {
		log.Printf("Error serializing message: %v", err)
		return
	}

	r.mu.Lock()
	var source map[*Connection]bool
	if groupName == "" {
		source = r.connections
	} else if group, ok := r.groups[groupName]; ok {
		source = group
	} else {
		r.mu.Unlock()
		log.Printf("Group %s not found.", groupName)
		return
	}
	// Copy rather than range outside the lock: the maps are mutated by
	// register/unregister, and ranging one concurrently is a fatal runtime
	// throw, not a recoverable race.
	targets := make([]*Connection, 0, len(source))
	for conn := range source {
		targets = append(targets, conn)
	}
	r.mu.Unlock()

	// Send is never closed (see Connection.CloseConnection), so this can never
	// panic - a connection that's mid-close just has a queue nobody drains, and
	// the default case below keeps that from blocking the broadcaster.
	for _, conn := range targets {
		select {
		case conn.Send <- serializedMsg:
			// Message successfully queued to send.
		default:
			log.Printf("Failed to send to connection %s. Channel full.", conn.ID)
		}
	}
}

// ClearConnections closes and removes all active connections. For testing use only.
func (r *Registry) ClearConnections() {
	r.closeAll()
}
