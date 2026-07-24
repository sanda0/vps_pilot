package tcpserver

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/sanda0/vps_pilot/internal/db"
)

var AgentConnections map[string]net.Conn
var agentConnectionsMu sync.Mutex

func StartTcpServer(ctx context.Context, repo *db.Repo, port string) {

	AgentConnections = make(map[string]net.Conn)

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("TCP server failed to start:", err)
		return
	}
	defer listener.Close()
	fmt.Println("TCP server listening on port", port)

	chans := statChans{
		stat:    make(chan Msg, 100),
		monitor: make(chan Msg, 100),
	}

	go StoreSystemStats(ctx, repo, chans.stat)
	go MontiorAlerts(ctx, repo, chans.monitor)

	router := NewRouter()

	// Middleware
	router.Use(RequireAgentToken(os.Getenv("AGENT_TOKEN")))
	router.Use(RequireHandshake("connected"))

	// Handlers
	router.Handle("connected", HandleConnected)
	router.Handle("sys_info", HandleSysInfo)
	router.Handle("sys_stat", HandleSysStat(chans))

	router.Serve(ctx, repo, listener)
}

// RequireAgentToken validates the optional shared secret on every agent frame.
// Leaving AGENT_TOKEN empty preserves a convenient local-development mode.
func RequireAgentToken(expected string) MiddlewareFunc {
	if expected == "" {
		fmt.Println("Warning: AGENT_TOKEN is not set; agent TCP authentication is disabled")
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, c *ConnContext, msg Msg) {
			if expected != "" &&
				subtle.ConstantTimeCompare([]byte(msg.Token), []byte(expected)) != 1 {
				fmt.Printf("RequireAgentToken: rejected message from %s\n", c.RemoteAddr)
				return
			}
			next(ctx, c, msg)
		}
	}
}
