package tcpserver

import (
	"net"
	"sync"
	"testing"
)

func newTestConn(t *testing.T, nodeID int64) (*ConnContext, net.Conn) {
	t.Helper()
	server, client := net.Pipe()
	t.Cleanup(func() { server.Close(); client.Close() })
	return &ConnContext{RemoteAddr: "pipe", NodeID: nodeID, conn: server}, client
}

func TestRegistryReplacesAndKicksOldConnection(t *testing.T) {
	r := NewRegistry()
	first, firstPeer := newTestConn(t, 1)
	second, _ := newTestConn(t, 1)

	r.Register(first)
	r.Register(second)

	if got, _ := r.Get(1); got != second {
		t.Fatal("expected newest connection to win")
	}
	// The old connection was closed: its peer sees EOF.
	if _, err := firstPeer.Read(make([]byte, 1)); err == nil {
		t.Fatal("expected old connection to be closed")
	}

	// The old connection's cleanup must not evict the new one.
	r.Unregister(first)
	if _, ok := r.Get(1); !ok {
		t.Fatal("unregistering a stale connection removed the live one")
	}

	if !r.Kick(1) {
		t.Fatal("expected Kick to find the connection")
	}
}

func TestRegistryConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup
	for i := int64(1); i <= 50; i++ {
		c, _ := newTestConn(t, i%5+1)
		wg.Add(3)
		go func() { defer wg.Done(); r.Register(c) }()
		go func() { defer wg.Done(); r.Get(c.NodeID) }()
		go func() { defer wg.Done(); r.Unregister(c) }()
	}
	wg.Wait()
}
