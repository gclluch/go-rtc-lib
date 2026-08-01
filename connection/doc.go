// Package connection is a WebSocket hub: it tracks live connections, groups
// them into rooms, and broadcasts to one group or to everything.
//
// A [Registry] owns the connections. Start its loop with a context, mount its
// handler, and cancel the context to drain:
//
//	reg := connection.NewRegistry()
//	ctx, cancel := context.WithCancel(context.Background())
//	go reg.Run(ctx)
//
//	http.Handle("/ws", reg.RegisterHandler(myHandler))
//	http.ListenAndServe(":8080", nil)
//
//	cancel() // closes every live connection and returns from Run
//
// myHandler is any [MessageHandler]. Its HandleMessage is called for each
// inbound frame and may return bytes to send straight back to that client.
//
// # Groups
//
// Groups are created on demand - [Registry.AddToGroup] makes the group if it
// does not exist. Membership is tracked on both sides, so unregistering a
// connection removes it from every group it had joined.
//
//	reg.AddToGroup("room-1", conn)
//	reg.Broadcast(msg, "room-1") // just that room
//	reg.BroadcastToAll(msg)      // everyone
//
// Adding a connection that has already been unregistered is a no-op rather than
// an error: the registry has forgotten it, so nothing would ever remove it again.
//
// # What it does not do
//
// Delivery is best-effort. Each connection has a 256-message outbound buffer and
// [Registry.Broadcast] drops - with a log line - to any connection whose buffer
// is full, rather than blocking the broadcaster. There is no acknowledgement, no
// retry, and no replay for a client that reconnects.
//
// State lives in one process. A Registry is not shared across replicas, so two
// instances behind a load balancer do not see each other's connections.
//
// Inbound bytes reach your [MessageHandler] undecoded. [github.com/gclluch/go-rtc-lib/message]
// provides IMessage implementations for the outbound path; the library does not
// deserialize what it receives.
//
// # Origin checking
//
// [Registry.CheckOrigin] defaults to same-origin only, which blocks cross-site
// WebSocket hijacking out of the box. A browser client served from a different
// host than the WebSocket endpoint will be rejected until you widen it:
//
//	reg.CheckOrigin = func(r *http.Request) bool {
//	    return r.Header.Get("Origin") == "https://app.example.com"
//	}
//
// Do not blanket-allow every origin unless every caller of the endpoint is
// already trusted.
//
// # Concurrency
//
// Registry methods are safe to call from multiple goroutines. Each connection
// runs two goroutines, a read pump and a write pump; both exit when the
// connection closes, and [Connection.CloseConnection] is safe to call more than
// once and from either of them.
//
// The exported Connection.Send channel and Connection.WS field are part of the
// v1 API and are kept for compatibility. Prefer [Registry.Broadcast] over sending
// to Send directly, and treat WS as read-only - writing to the socket from
// outside the write pump races it.
package connection
