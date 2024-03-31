package connection

import (
	"sync"
)

type Registry struct {
	connections map[*Connection]bool
	broadcast   chan []byte
	register    chan *Connection
	unregister  chan *Connection
	mu          sync.Mutex
}

func NewRegistry() *Registry {
	return &Registry{
		broadcast:   make(chan []byte),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		connections: make(map[*Connection]bool),
	}
}

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
				conn.CloseConnection()
			}
			r.mu.Unlock()
		case message := <-r.broadcast:
			r.mu.Lock()
			for conn := range r.connections {
				select {
				case conn.Send <- message:
				default:
					// Log failure or take necessary action.
				}
			}
			r.mu.Unlock()
		}
	}
}

func (r *Registry) Broadcast(message []byte) {
	r.broadcast <- message
}
