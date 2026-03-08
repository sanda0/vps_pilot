package tcpserver

import (
	"context"
	"encoding/gob"
	"fmt"
	"net"

	"github.com/sanda0/vps_pilot/internal/db"
)

// HandlerFunc is the signature every message handler must implement.
// ctx      — request context (carries cancellation / deadline)
// c        — the connection context for this agent session
// msg      — the decoded message from the agent
type HandlerFunc func(ctx context.Context, c *ConnContext, msg Msg)

// ConnContext holds everything that belongs to a single agent connection.
// Handlers receive a pointer to it so they can read state (node ID, encoder)
// and write back to the agent.
type ConnContext struct {
	// RemoteAddr is the agent's remote TCP address ("ip:port")
	RemoteAddr string

	// NodeID is populated after the "connected" handshake completes.
	// It is zero until then.
	NodeID int64

	// encoder is used to write messages back to the agent.
	encoder *gob.Encoder

	// repo gives handlers access to both databases.
	repo *db.Repo
}

// Send encodes and writes a message back to the agent.
func (c *ConnContext) Send(msg Msg) error {
	if err := c.encoder.Encode(msg); err != nil {
		return fmt.Errorf("send to %s: %w", c.RemoteAddr, err)
	}
	return nil
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
// connection, and drives the decode → dispatch loop.
func (r *Router) Serve(ctx context.Context, repo *db.Repo, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			// Listener was closed or a fatal OS error occurred.
			return
		}

		remoteAddr := conn.RemoteAddr().String()
		AgentConnections[remoteAddr] = conn

		c := &ConnContext{
			RemoteAddr: remoteAddr,
			encoder:    gob.NewEncoder(conn),
			repo:       repo,
		}

		go r.serveConn(ctx, c, conn)
	}
}

// serveConn runs the decode → dispatch loop for a single agent connection.
func (r *Router) serveConn(ctx context.Context, c *ConnContext, conn net.Conn) {
	defer func() {
		conn.Close()
		delete(AgentConnections, c.RemoteAddr)
		fmt.Println("Connection closed:", c.RemoteAddr)
	}()

	fmt.Println("New connection from", c.RemoteAddr)

	decoder := gob.NewDecoder(conn)
	for {
		var msg Msg
		if err := decoder.Decode(&msg); err != nil {
			// EOF or broken pipe — agent disconnected.
			break
		}
		r.Dispatch(ctx, c, msg)
	}
}

// RequireHandshake returns a middleware that ensures the "connected" handshake
// has completed before any other message type is dispatched.
// The handshake message type (typically "connected") is always allowed through
// so the handshake itself is never blocked.
func RequireHandshake(handshakeMsgType string) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, c *ConnContext, msg Msg) {
			// Always allow the handshake message through.
			if msg.Msg == handshakeMsgType {
				next(ctx, c, msg)
				return
			}

			// Block any other message until the handshake has set a NodeID.
			if c.NodeID == 0 {
				fmt.Printf("RequireHandshake: dropping %q from %s — handshake not complete\n", msg.Msg, c.RemoteAddr)
				return
			}

			next(ctx, c, msg)
		}
	}
}
