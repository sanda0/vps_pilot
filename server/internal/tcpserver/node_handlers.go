package tcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sanda0/vps_pilot/internal/agentauth"
)

// Rejection reasons are deliberately generic so they don't reveal which part
// of the credential was wrong.
const (
	reasonBadCredentials = "invalid credentials"
	reasonBadJoinToken   = "invalid, used or expired join token"
)

// HandleEnroll is called when a new agent sends "enroll" with a one-time join
// secret. It consumes the token, issues an agent ID and node key, and replies
// "enrolled" followed by "sys_stat" to start streaming.
func HandleEnroll(ctx context.Context, c *ConnContext, msg Msg) {
	if c.NodeID != 0 {
		return
	}

	sysInfo := SystemInfo{}
	if err := sysInfo.FromBytes(msg.Data); err != nil {
		fmt.Printf("HandleEnroll: bad sys info from %s: %v\n", c.RemoteAddr, err)
	}

	creds, err := EnrollNode(ctx, c.Repo(), msg.Token, c.IP(), sysInfo)
	if err != nil {
		fmt.Printf("HandleEnroll: rejected %s: %v\n", c.RemoteAddr, err)
		c.Reject(reasonBadJoinToken)
		return
	}

	data, err := json.Marshal(creds)
	if err != nil {
		c.Close()
		return
	}
	if err := c.Send(Msg{Msg: MsgEnrolled, NodeId: int32(creds.NodeID), Data: data}); err != nil {
		fmt.Printf("HandleEnroll: %v\n", err)
		c.Close()
		return
	}

	c.NodeID = creds.NodeID
	c.Features = sysInfo.Features
	fmt.Printf("HandleEnroll: node %d enrolled from %s\n", creds.NodeID, c.RemoteAddr)
	startStats(c)
	sendProjectSettings(ctx, c)
}

// HandleConnected is called when an enrolled agent sends "connected" with
// "<agent_id>:<node_key>". On success it stores the node ID on the
// ConnContext and tells the agent to start streaming system stats.
func HandleConnected(ctx context.Context, c *ConnContext, msg Msg) {
	if c.NodeID != 0 {
		return
	}

	agentID, nodeKey, ok := agentauth.SplitConnectToken(msg.Token)
	if !ok {
		c.Reject(reasonBadCredentials)
		return
	}

	node, err := c.Repo().Queries.GetNodeByAgentID(ctx, sqlString(agentID))
	if err != nil || !node.KeyHash.Valid || !agentauth.HashEqual(nodeKey, node.KeyHash.String) {
		fmt.Printf("HandleConnected: rejected %s (agent %q)\n", c.RemoteAddr, agentID)
		c.Reject(reasonBadCredentials)
		return
	}

	if err := c.Repo().Queries.TouchNode(ctx, touchNodeParams(c.IP(), node.ID)); err != nil {
		fmt.Printf("HandleConnected: touch node %d: %v\n", node.ID, err)
	}

	sysInfo := SystemInfo{}
	if err := sysInfo.FromBytes(msg.Data); err == nil {
		if err := UpsertSysInfo(ctx, c.Repo().Queries, node.ID, sysInfo); err != nil {
			fmt.Printf("HandleConnected: sys info for node %d: %v\n", node.ID, err)
		}
	}

	c.NodeID = node.ID
	c.Features = sysInfo.Features
	fmt.Printf("HandleConnected: node %d connected from %s\n", node.ID, c.RemoteAddr)
	startStats(c)
	sendProjectSettings(ctx, c)
}

// startStats tells the agent its node ID and asks it to start sending stats.
func startStats(c *ConnContext) {
	if err := c.Send(Msg{Msg: MsgSysStat, NodeId: int32(c.NodeID)}); err != nil {
		fmt.Printf("failed to send sys_stat ack to node %d: %v\n", c.NodeID, err)
	}
}

// HandleSysInfo is called when an agent sends a "sys_info" message.
func HandleSysInfo(ctx context.Context, c *ConnContext, msg Msg) {
	sysInfo := SystemInfo{}
	if err := sysInfo.FromBytes(msg.Data); err != nil {
		fmt.Printf("HandleSysInfo: bad payload from node %d: %v\n", c.NodeID, err)
		return
	}
	if err := UpsertSysInfo(ctx, c.Repo().Queries, c.NodeID, sysInfo); err != nil {
		fmt.Printf("HandleSysInfo: node %d: %v\n", c.NodeID, err)
	}
}
