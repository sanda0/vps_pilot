package tcpserver

import (
	"context"
	"fmt"
	"net"

	"github.com/sanda0/vps_pilot/db"
)

var AgentConnections map[string]net.Conn

func StartTcpServer(ctx context.Context, repo *db.Repo, port string) {
	AgentConnections = make(map[string]net.Conn)
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return
	}
	defer listener.Close()
	fmt.Println("TCP server Listening on port", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		AgentConnections[conn.RemoteAddr().String()] = conn
		go handleRequest(ctx, repo, conn)
	}
}

func handleRequest(ctx context.Context, repo *db.Repo, conn net.Conn) {
	defer conn.Close()
	fmt.Println("New connection from", conn.RemoteAddr())

}
