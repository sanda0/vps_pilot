package tcpserver

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/sanda0/vps_pilot/internal/agentauth"
	"github.com/sanda0/vps_pilot/internal/db"
)

func StartTcpServer(ctx context.Context, repo *db.Repo, registry *Registry, cert *agentauth.Cert, port string) {
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert.TLS},
		MinVersion:   tls.VersionTLS13,
	}
	listener, err := tls.Listen("tcp", ":"+port, tlsConfig)
	if err != nil {
		fmt.Println("TCP server failed to start:", err)
		return
	}
	defer listener.Close()
	fmt.Println("TCP server (TLS) listening on port", port)
	fmt.Println("Agent TLS fingerprint:", cert.Fingerprint)

	chans := statChans{
		stat:    make(chan Msg, 100),
		monitor: make(chan Msg, 100),
	}

	go StoreSystemStats(ctx, repo, chans.stat)
	go MontiorAlerts(ctx, repo, chans.monitor)

	router := NewRouter()
	registerHandlers(router, chans)

	router.Serve(ctx, repo, registry, listener)
}

func registerHandlers(router *Router, chans statChans) {
	// Middleware
	router.Use(RequireHandshake(MsgConnected, MsgEnroll))

	// Handlers
	router.Handle(MsgEnroll, HandleEnroll)
	router.Handle(MsgConnected, HandleConnected)
	router.Handle(MsgSysInfo, HandleSysInfo)
	router.Handle(MsgSysStat, HandleSysStat(chans))
	router.Handle(MsgProjects, HandleProjects)
}
