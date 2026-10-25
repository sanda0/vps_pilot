package tcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/sanda0/vps_pilot/internal/db"
)

// HandshakeTimeout bounds how long an unauthenticated connection may stay open.
var HandshakeTimeout = 10 * time.Second

const writeTimeout = 10 * time.Second

// HandlerFunc is the signature every message handler must implement.
// ctx      — request context (carries cancellation / deadline)
// c        — the connection context for this agent session
// msg      — the decoded message from the agent
type HandlerFunc func(ctx context.Context, c *ConnContext, msg Msg)

// ConnContext holds everything that belongs to a single agent connection.
// Handlers receive a pointer to it so they can read state (node ID) and
// write back to the agent.
type ConnContext struct {
	// RemoteAddr is the agent's remote TCP address ("ip:port")
	RemoteAddr string

	// NodeID is populated once the agent has authenticated ("connected" or
	// "enroll"). It is zero until then. Only the connection's own goroutine
	// writes it.
	NodeID int64

	// Features the agent listed in SystemInfo at the handshake.
	Features []string

	conn   net.Conn
	sendMu sync.Mutex

	// repo gives handlers access to both databases.
	repo *db.Repo
}

// Send writes a message back to the agent. Safe for concurrent use.
func (c *ConnContext) Send(msg Msg) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := WriteFrame(c.conn, msg); err != nil {
		return fmt.Errorf("send to %s: %w", c.RemoteAddr, err)
	}
	return nil
}

// Reject tells the agent why it was refused and closes the connection.
func (c *ConnContext) Reject(reason string) {
	data, _ := json.Marshal(AuthFailedPayload{Reason: reason})
	if err := c.Send(Msg{Msg: MsgAuthFailed, Data: data}); err != nil {
		fmt.Printf("Reject: %v\n", err)
	}
	c.Close()
}

// Close closes the underlying connection; the read loop then exits.
func (c *ConnContext) Close() error {
	return c.conn.Close()
}

// IP returns the remote IP without the port (IPv6-safe).
func (c *ConnContext) IP() string {
	host, _, err := net.SplitHostPort(c.RemoteAddr)
	if err != nil {
		return c.RemoteAddr
	}
	return host
}

// Supports reports whether the agent listed feature f at the handshake.
func (c *ConnContext) Supports(f string) bool {
	return slices.Contains(c.Features, f)
}

// Repo exposes the database repo to handlers (read-only pointer, safe to share).
func (c *ConnContext) Repo() *db.Repo {
	return c.repo
}

// Router maps message type strings to handler functions.
type Router struct {
	handlers   map[string]HandlerFunc
	middleware []MiddlewareFunc
}

// MiddlewareFunc wraps a HandlerFunc, enabling pre/post processing.
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// NewRouter creates an empty Router.
func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]HandlerFunc),
	}
}

// Use appends middleware that will run around every handler.
// Middleware is applied in the order it is registered.
func (r *Router) Use(mw ...MiddlewareFunc) {
	r.middleware = append(r.middleware, mw...)
}

// Handle registers a HandlerFunc for the given message type.
// Registering the same type twice panics to catch mistakes early.
func (r *Router) Handle(msgType string, h HandlerFunc) {
	if _, exists := r.handlers[msgType]; exists {
		panic(fmt.Sprintf("tcpserver: handler already registered for message type %q", msgType))
	}
	r.handlers[msgType] = h
}

// Dispatch looks up the handler for msg.Msg, wraps it with middleware,
// and calls it. If no handler is registered it logs and returns.
func (r *Router) Dispatch(ctx context.Context, c *ConnContext, msg Msg) {
	h, ok := r.handlers[msg.Msg]
	if !ok {
		fmt.Printf("tcpserver: no handler for message type %q (node %d)\n", msg.Msg, c.NodeID)
		return
	}

	// Apply middleware in reverse registration order so the first registered
	// middleware is the outermost wrapper (same convention as net/http).
	final := h
	for i := len(r.middleware) - 1; i >= 0; i-- {
		final = r.middleware[i](final)
	}

	final(ctx, c, msg)
}

// Serve accepts connections on the given listener, creates a ConnContext per
// connection, and drives the decode → dispatch loop. Authenticated
// connections are tracked in registry.
func (r *Router) Serve(ctx context.Context, repo *db.Repo, registry *Registry, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			// Listener was closed or a fatal OS error occurred.
			return
		}

		c := &ConnContext{
			RemoteAddr: conn.RemoteAddr().String(),
			conn:       conn,
			repo:       repo,
		}

		go r.serveConn(ctx, c, registry)
	}
}

// serveConn runs the decode → dispatch loop for a single agent connection.
func (r *Router) serveConn(ctx context.Context, c *ConnContext, registry *Registry) {
	defer func() {
		c.Close()
		if c.NodeID != 0 {
			registry.Unregister(c)
		}
		fmt.Println("Connection closed:", c.RemoteAddr)
	}()

	fmt.Println("New connection from", c.RemoteAddr)

	// Covers the TLS handshake and the first message.
	c.conn.SetReadDeadline(time.Now().Add(HandshakeTimeout))

	for {
		var msg Msg
		if err := ReadFrame(c.conn, &msg); err != nil {
			// EOF, broken pipe, deadline or a malformed frame — drop the agent.
			return
		}

		authed := c.NodeID != 0
		r.Dispatch(ctx, c, msg)

		if !authed && c.NodeID != 0 {
			c.conn.SetReadDeadline(time.Time{})
			registry.Register(c)
		}
	}
}

// RequireHandshake returns a middleware that drops every message until the
// agent has authenticated. The handshake message types themselves are always
// let through so the handshake is never blocked.
func RequireHandshake(handshakeMsgTypes ...string) MiddlewareFunc {
	allowed := make(map[string]bool, len(handshakeMsgTypes))
	for _, t := range handshakeMsgTypes {
		allowed[t] = true
	}
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, c *ConnContext, msg Msg) {
			if allowed[msg.Msg] {
				next(ctx, c, msg)
				return
			}

			if c.NodeID == 0 {
				fmt.Printf("RequireHandshake: dropping %q from %s — handshake not complete\n", msg.Msg, c.RemoteAddr)
				return
			}

			next(ctx, c, msg)
		}
	}
}
