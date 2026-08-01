package connection

import (
	"sync"
	"testing"
	"time"

	"github.com/gclluch/go-rtc-lib/message"
)

// TestAddToGroupTracksMembershipOnConnection is a regression test for the
// bug where AddToGroup updated the registry's group map but never recorded
// the membership on the connection itself, which meant unregisterConnection
// (formerly the CloseConnection cleanup loop) had nothing to iterate and
// dead connections were never removed from groups.
func TestAddToGroupTracksMembershipOnConnection(t *testing.T) {
	r := NewRegistry()
	conn := NewConnection(nil, nil)

	r.AddToGroup("room1", conn)

	if !conn.groups["room1"] {
		t.Fatal("AddToGroup did not record group membership on the connection")
	}
}

// TestUnregisterConnectionCleansUpGroups is a regression test for the group
// leak: previously a disconnected connection stayed in r.groups forever
// because conn.groups was always empty.
func TestUnregisterConnectionCleansUpGroups(t *testing.T) {
	r := NewRegistry()
	conn := NewConnection(nil, nil)

	r.AddToGroup("room1", conn)
	r.unregisterConnection(conn)

	r.mu.Lock()
	_, stillMember := r.groups["room1"][conn]
	r.mu.Unlock()

	if stillMember {
		t.Fatal("connection was not removed from its group on unregister")
	}
}

func TestRemoveFromGroupClearsConnectionSide(t *testing.T) {
	r := NewRegistry()
	conn := NewConnection(nil, nil)

	r.AddToGroup("room1", conn)
	r.RemoveFromGroup("room1", conn)

	if conn.groups["room1"] {
		t.Fatal("RemoveFromGroup left stale membership on the connection")
	}
}

// TestConcurrentBroadcastDuringDisconnect is a regression test for the
// send-on-closed-channel panic: CloseConnection used to close(c.Send) while
// Broadcast concurrently sent on it via select/default, which does not
// protect against a closed channel. Run with -race; the old code would
// panic (and race) under this interleaving.
func TestConcurrentBroadcastDuringDisconnect(t *testing.T) {
	r := NewRegistry()

	const n = 50
	conns := make([]*Connection, n)
	for i := range conns {
		c := NewConnection(nil, nil)
		conns[i] = c

		r.mu.Lock()
		r.connections[c] = true
		r.mu.Unlock()

		// Drain Send so the broadcaster doesn't just fill the buffer and
		// hit the default case every time. Stops once the connection closes.
		go func(c *Connection) {
			for {
				select {
				case <-c.Send:
				case <-c.done:
					return
				}
			}
		}(c)
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg := message.NewJSONMessage("ping")
			for {
				select {
				case <-stop:
					return
				default:
					r.Broadcast(msg, "")
				}
			}
		}()
	}

	for _, c := range conns {
		wg.Add(1)
		go func(c *Connection) {
			defer wg.Done()
			c.CloseConnection()
		}(c)
	}

	time.Sleep(50 * time.Millisecond)
	close(stop)
	wg.Wait()
}
