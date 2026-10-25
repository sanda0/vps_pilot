package tcpserver

import "sync"

// Registry tracks authenticated agent connections by node ID. It is safe for
// concurrent use; Phase 2 command dispatch looks agents up here.
type Registry struct {
	mu           sync.RWMutex
	conns        map[int64]*ConnContext
	onDisconnect []func(*ConnContext)
}

func NewRegistry() *Registry {
	return &Registry{conns: make(map[int64]*ConnContext)}
}

// Register records c under its node ID. If the node already had a connection,
// the old one is closed: a node key is only valid for one live session.
func (r *Registry) Register(c *ConnContext) {
	r.mu.Lock()
	old := r.conns[c.NodeID]
	r.conns[c.NodeID] = c
	r.mu.Unlock()

	if old != nil && old != c {
		old.Close()
	}
}

// OnDisconnect registers f to run whenever an authenticated connection
// closes, even one already replaced by a newer connection. Call it before
// the server starts.
func (r *Registry) OnDisconnect(f func(*ConnContext)) {
	r.onDisconnect = append(r.onDisconnect, f)
}

// Unregister removes c, unless the node has since reconnected on a newer
// connection, and runs the OnDisconnect functions for c.
func (r *Registry) Unregister(c *ConnContext) {
	r.mu.Lock()
	if r.conns[c.NodeID] == c {
		delete(r.conns, c.NodeID)
	}
	r.mu.Unlock()
	for _, f := range r.onDisconnect {
		f(c)
	}
}

// Get returns the live connection for a node, if any.
func (r *Registry) Get(nodeID int64) (*ConnContext, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.conns[nodeID]
	return c, ok
}

// Online reports whether a node has a live connection.
func (r *Registry) Online(nodeID int64) bool {
	_, ok := r.Get(nodeID)
	return ok
}

// Kick closes a node's live connection. It reports whether one was open.
func (r *Registry) Kick(nodeID int64) bool {
	c, ok := r.Get(nodeID)
	if ok {
		c.Close()
	}
	return ok
}
