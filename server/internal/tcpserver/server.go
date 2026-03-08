package tcpserver

import (
	"context"
	"encoding/gob"
	"fmt"
	"net"
	"strings"

	"github.com/sanda0/vps_pilot/internal/db"
)

// nodeIDByConn maps remote address → node ID so project sync can look up
// the node ID that was assigned during the "connected" handshake.
var nodeIDByConn map[string]int64

var AgentConnections map[string]net.Conn

func StartTcpServer(ctx context.Context, repo *db.Repo, port string) {

	var statChan = make(chan Msg, 100)
	var monitorChan = make(chan Msg, 100)

	AgentConnections = make(map[string]net.Conn)
	nodeIDByConn = make(map[string]int64)
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
	remoteAddr := conn.RemoteAddr().String()
	defer func() {
		conn.Close()
		delete(AgentConnections, remoteAddr)
		delete(nodeIDByConn, remoteAddr)
		fmt.Println("Connection closed:", remoteAddr)
	}()

	fmt.Println("New connection from", remoteAddr)

	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)
	var msg Msg
	for {
		err := decoder.Decode(&msg)
		if err != nil {
			break
		}

		if msg.Msg == "connected" {
			ip := strings.Split(remoteAddr, ":")[0]
			node, err := CreateNode(ctx, repo, ip, msg.Data)
			if err != nil {
				fmt.Println("Error creating node", err)
			}
			fmt.Println("Node connected", node)

			// Remember the node ID for this connection
			nodeIDByConn[remoteAddr] = node.ID

			err = encoder.Encode(Msg{
				Msg:    "sys_stat",
				NodeId: int32(node.ID),
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

		if msg.Msg == "projects" {
			nodeID, ok := nodeIDByConn[remoteAddr]
			if !ok {
				fmt.Println("Received projects message but node ID not found for", remoteAddr)
				continue
			}
			var payload ProjectSyncPayload
			if err := payload.FromBytes(msg.Data); err != nil {
				fmt.Printf("Error decoding projects payload from node %d: %v\n", nodeID, err)
				continue
			}
			go SyncProjects(ctx, repo, nodeID, payload)
		}
	}
}
