package tcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sanda0/vps_pilot/internal/db"
)

// LoadProjectSettings builds the project_settings payload for a node from the
// dashboard's settings (defaults if never saved) and its manual paths.
func LoadProjectSettings(ctx context.Context, q *db.Queries, nodeID int64) (ProjectSettings, error) {
	scan, err := q.GetNodeProjectScan(ctx, nodeID)
	if err != nil {
		return ProjectSettings{}, err
	}
	rows, err := q.ListNodeProjectPaths(ctx, nodeID)
	if err != nil {
		return ProjectSettings{}, err
	}
	paths := make([]string, 0, len(rows))
	for _, r := range rows {
		paths = append(paths, r.Path)
	}
	return ProjectSettings{AutoScan: scan.AutoScan != 0, ScanInterval: scan.ScanInterval, Paths: paths}, nil
}

// sendProjectSettings sends a node's settings to its agent. It runs right
// after the handshake, so a node that was offline when its settings changed
// gets them on reconnect.
func sendProjectSettings(ctx context.Context, c *ConnContext) {
	settings, err := LoadProjectSettings(ctx, c.Repo().Queries, c.NodeID)
	if err != nil {
		fmt.Printf("project settings for node %d: %v\n", c.NodeID, err)
		return
	}
	data, _ := json.Marshal(settings)
	if err := c.Send(Msg{Msg: MsgProjectSettings, Data: data}); err != nil {
		fmt.Printf("send project settings to node %d: %v\n", c.NodeID, err)
	}
}

// PushProjectSettings sends a node's current settings to its agent if it is
// connected. Call it after changing the settings or manual paths; an offline
// node gets them when it reconnects.
func (r *Registry) PushProjectSettings(ctx context.Context, nodeID int64) {
	if c, ok := r.Get(nodeID); ok {
		sendProjectSettings(ctx, c)
	}
}

// RequestProjectRescan asks a node's agent to search for projects now. It
// reports false if the node isn't connected.
func (r *Registry) RequestProjectRescan(nodeID int64) (bool, error) {
	c, ok := r.Get(nodeID)
	if !ok {
		return false, nil
	}
	return true, c.Send(Msg{Msg: MsgRescanProjects})
}

// recordProjectScan stores when the node was last scanned and the errors it
// reported, replacing the previous ones.
func recordProjectScan(ctx context.Context, q *db.Queries, nodeID int64, errs []ProjectError) {
	if errs == nil {
		errs = []ProjectError{}
	}
	data, _ := json.Marshal(errs)
	if err := q.RecordNodeProjectScan(ctx, db.RecordNodeProjectScanParams{NodeID: nodeID, Errors: string(data)}); err != nil {
		fmt.Printf("record project scan for node %d: %v\n", nodeID, err)
	}
}
