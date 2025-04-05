package tcpserver

import (
	"context"
	"encoding/gob"
	"fmt"
	"net"
	"strings"

	"github.com/sanda0/vps_pilot/internal/db"
)

var AgentConnections map[string]net.Conn

func StartTcpServer(ctx context.Context, repo *db.Repo, port string) {

	var statChan = make(chan Msg, 100)
	var monitorChan = make(chan Msg, 100)

	AgentConnections = make(map[string]net.Conn)
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return
	}
	defer listener.Close()
	fmt.Println("TCP server Listening on port", port)

	go StoreSystemStats(ctx, repo, statChan)
	go MontiorAlerts(ctx, repo, monitorChan)

	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		AgentConnections[conn.RemoteAddr().String()] = conn
		go handleRequest(ctx, repo, conn, statChan, monitorChan)
	}
}

func handleRequest(ctx context.Context, repo *db.Repo, conn net.Conn, statChan chan Msg, monitorChan chan Msg) {
	defer conn.Close()
	fmt.Println("New connection from", conn.RemoteAddr())

	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)
	var msg Msg
	for {
		err := decoder.Decode(&msg)
		if err != nil {
			break
		}
		if msg.Msg == "connected" {
			ip := strings.Split(conn.RemoteAddr().String(), ":")[0]
			node, err := CreateNode(ctx, repo, ip, msg.Data)
			if err != nil {
				fmt.Println("Error creating node", err)
			}
			fmt.Println("Node connected", node)
			err = encoder.Encode(Msg{
				Msg:    "sys_stat",
				NodeId: node.ID,
			})
			if err != nil {
				fmt.Println("Error encoding message:", err)
			}
		}
		if msg.Msg == "sys_info" {

			fmt.Println("Sys info received", string(msg.Data))
		}
		if msg.Msg == "sys_stat" {
			statChan <- msg
			monitorChan <- msg
		}
	}

}
