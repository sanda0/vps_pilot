package tcpserver

import (
	"context"
	"fmt"
	"strings"
)

// HandleConnected is called when an agent sends the initial "connected" message.
// It creates (or retrieves) the node record, stores the node ID on the
// ConnContext so subsequent handlers can use it, and tells the agent to start
// streaming system stats.
func HandleConnected(ctx context.Context, c *ConnContext, msg Msg) {
	ip := strings.Split(c.RemoteAddr, ":")[0]

	node, err := CreateNode(ctx, c.Repo(), ip, msg.Data)
	if err != nil {
		fmt.Printf("HandleConnected: error upserting node for %s: %v\n", c.RemoteAddr, err)
		return
	}

	// Store the resolved node ID on the connection context so every
	// subsequent handler can access it without a DB round-trip.
	c.NodeID = node.ID
	fmt.Printf("HandleConnected: node %d (%s) connected\n", node.ID, ip)

	// Tell the agent its assigned node ID and ask it to start sending stats.
	if err := c.Send(Msg{
		Msg:    "sys_stat",
		NodeId: int32(node.ID),
	}); err != nil {
		fmt.Printf("HandleConnected: failed to send sys_stat ack to node %d: %v\n", node.ID, err)
	}
}

// HandleSysInfo is called when an agent sends a "sys_info" message.
// Currently just logged; can be extended to update node metadata.
func HandleSysInfo(ctx context.Context, c *ConnContext, msg Msg) {
	fmt.Printf("HandleSysInfo: node %d sent sys_info: %s\n", c.NodeID, string(msg.Data))
}
