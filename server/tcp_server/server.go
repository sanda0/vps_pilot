package tcpserver

import (
	"context"
	"encoding/gob"
	"fmt"
	"net"
	"strings"

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

	decoder := gob.NewDecoder(conn)
	// encoder := gob.NewEncoder(conn)
	var msg Msg
	for {
		err := decoder.Decode(&msg)
		if err != nil {
			break
		}
		if msg.Msg == "connected" {
			ip := strings.Split(conn.RemoteAddr().String(), ":")[0]
			go CreateNode(ctx, repo, ip, msg.Data)

		}
		if msg.Msg == "sys_info" {
			fmt.Println("Sys info received", string(msg.Data))
		}
		if msg.Msg == "sys_stat" {
			fmt.Println("Sys stat received", string(msg.Data))
		}
	}

}
