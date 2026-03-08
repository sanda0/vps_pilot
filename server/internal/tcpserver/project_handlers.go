package tcpserver

import (
	"context"
	"fmt"
)

// HandleProjects is called when an agent sends a "projects" message.
// The agent sends the full list of config.vpspilot.json files it found on
// disk. The handler upserts every reported project and removes any stale
// entries that are no longer present.
func HandleProjects(ctx context.Context, c *ConnContext, msg Msg) {
	if c.NodeID == 0 {
		fmt.Printf("HandleProjects: dropping projects message from %s — handshake not complete\n", c.RemoteAddr)
		return
	}

	var payload ProjectSyncPayload
	if err := payload.FromBytes(msg.Data); err != nil {
		fmt.Printf("HandleProjects: failed to decode payload from node %d: %v\n", c.NodeID, err)
		return
	}

	fmt.Printf("HandleProjects: node %d reported %d project(s)\n", c.NodeID, len(payload.Projects))

	// Run in the same goroutine — the TCP loop is already per-connection so
	// this does not block other connections. SyncProjects is DB-bound and
	// relatively infrequent, so no extra goroutine is needed.
	SyncProjects(ctx, c.Repo(), c.NodeID, payload)
}
