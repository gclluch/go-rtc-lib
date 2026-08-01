package connection

import (
	"testing"
	"time"

	"github.com/gclluch/go-rtc-lib/message"
)

// A connection whose Send buffer has filled has stopped draining, and the old
// behaviour - drop the message, log it, keep the connection - left that client
// silently stale for the rest of its life. Broadcast now closes it so the
// unregister path can reap it.
func TestBroadcastClosesAConnectionThatStoppedDraining(t *testing.T) {
	r := NewRegistry()
	conn := NewConnection(nil, nil) // nil WS: CloseConnection tolerates it

	r.mu.Lock()
	r.connections[conn] = true
	r.mu.Unlock()

	// Nobody reads Send, so fill it to capacity and the next send has nowhere
	// to go - exactly the state a wedged peer leaves behind.
	for len(conn.Send) < cap(conn.Send) {
		conn.Send <- []byte("backlog")
	}

	r.Broadcast(message.NewJSONMessage("one too many"), "")

	// CloseConnection runs off the broadcaster and waits out its close-frame
	// grace, so allow well past that before calling it a failure.
	select {
	case <-conn.done:
	case <-time.After(2 * time.Second):
		t.Fatal("a connection that stopped draining was left open; its client " +
			"would sit stale with no error anywhere")
	}
}
