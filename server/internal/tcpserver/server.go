package tcpserver

import (
	"context"
	"fmt"
	"net"

	"github.com/sanda0/vps_pilot/internal/db"
)

var AgentConnections map[string]net.Conn

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
	router.Use(RequireHandshake("connected"))

	// Handlers
	router.Handle("connected", HandleConnected)
	router.Handle("sys_info", HandleSysInfo)
	router.Handle("sys_stat", HandleSysStat(chans))
	router.Handle("projects", HandleProjects)

	router.Serve(ctx, repo, listener)
}
