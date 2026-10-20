package tcpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/sanda0/vps_pilot/internal/agentauth"
	"github.com/sanda0/vps_pilot/internal/db"
)

// EnrollNode consumes a join token and issues fresh node credentials. If the
// token was created for an existing node, that node's credentials are rotated
// (keeping its history); otherwise a new node is created. All of it runs in
// one transaction so a token can't be spent without producing credentials.
func EnrollNode(ctx context.Context, repo *db.Repo, joinSecret, ip string, sysInfo SystemInfo) (*EnrolledPayload, error) {
	if joinSecret == "" {
		return nil, errors.New("missing join secret")
	}

	agentID, nodeKey, err := agentauth.NewNodeCredentials()
	if err != nil {
		return nil, err
	}
	keyHash := sqlString(agentauth.Hash(nodeKey))

	tx, err := repo.OperationalDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	q := repo.Queries.WithTx(tx)

	token, err := q.ConsumeJoinToken(ctx, agentauth.Hash(joinSecret))
	if err != nil {
		return nil, fmt.Errorf("consume join token: %w", err)
	}

	var nodeID int64
	if token.NodeID.Valid {
		nodeID = token.NodeID.Int64
		n, err := q.SetNodeCredentials(ctx, db.SetNodeCredentialsParams{
			AgentID: sqlString(agentID),
			KeyHash: keyHash,
			ID:      nodeID,
		})
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, fmt.Errorf("node %d no longer exists", nodeID)
		}
		if err := q.TouchNode(ctx, touchNodeParams(ip, nodeID)); err != nil {
			return nil, err
		}
	} else {
		name := sysInfo.Hostname
		if name == "" {
			name = "node-" + ip
		}
		node, err := q.CreateEnrolledNode(ctx, db.CreateEnrolledNodeParams{
			Name:    sqlString(name),
			Ip:      ip,
			AgentID: sqlString(agentID),
			KeyHash: keyHash,
		})
		if err != nil {
			return nil, err
		}
		nodeID = node.ID
	}

	if err := q.SetJoinTokenUsedBy(ctx, db.SetJoinTokenUsedByParams{
		UsedByNodeID: sql.NullInt64{Int64: nodeID, Valid: true},
		ID:           token.ID,
	}); err != nil {
		return nil, err
	}

	if err := UpsertSysInfo(ctx, q, nodeID, sysInfo); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &EnrolledPayload{AgentID: agentID, NodeKey: nodeKey, NodeID: nodeID}, nil
}

// UpsertSysInfo stores the latest system info for a node.
func UpsertSysInfo(ctx context.Context, q *db.Queries, nodeID int64, s SystemInfo) error {
	params := db.UpdateNodeSysInfoParams{
		Os:              sqlString(s.OS),
		Platform:        sqlString(s.Platform),
		PlatformVersion: sqlString(s.PlatformVersion),
		KernelVersion:   sqlString(s.KernelVersion),
		Cpus:            sql.NullInt64{Int64: int64(s.CPUs), Valid: true},
		TotalMemory:     sql.NullFloat64{Float64: float64(s.TotalMemory), Valid: true},
		NodeID:          nodeID,
	}
	_, err := q.UpdateNodeSysInfo(ctx, params)
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = q.AddNodeSysInfo(ctx, db.AddNodeSysInfoParams{
		NodeID:          nodeID,
		Os:              params.Os,
		Platform:        params.Platform,
		PlatformVersion: params.PlatformVersion,
		KernelVersion:   params.KernelVersion,
		Cpus:            params.Cpus,
		TotalMemory:     params.TotalMemory,
	})
	return err
}

func sqlString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}

func touchNodeParams(ip string, nodeID int64) db.TouchNodeParams {
	return db.TouchNodeParams{Ip: ip, ID: nodeID}
}

// SyncProjects upserts all projects reported by the agent for a given node.
// Called when the TCP server receives a "projects" message.
func SyncProjects(ctx context.Context, repo *db.Repo, nodeID int64, payload ProjectSyncPayload) {
	for _, p := range payload.Projects {
		// Marshal JSON fields for DB storage
		techJSON, _ := json.Marshal(p.Tech)
		commandsJSON, _ := json.Marshal(p.Commands)
		logsJSON, _ := json.Marshal(p.Logs)

		var backupsJSON []byte
		if p.Backups != nil {
			backupsJSON, _ = json.Marshal(p.Backups)
		} else {
			backupsJSON = []byte("{}")
		}

		_, err := repo.Queries.UpsertProject(ctx, db.UpsertProjectParams{
			NodeID:   nodeID,
			Name:     p.Name,
			Path:     p.Path,
			Tech:     string(techJSON),
			Commands: string(commandsJSON),
			Logs:     string(logsJSON),
			Backups:  string(backupsJSON),
		})
		if err != nil {
			fmt.Printf("Error upserting project %q (node %d): %v\n", p.Path, nodeID, err)
		} else {
			fmt.Printf("Project synced: %q (node %d)\n", p.Name, nodeID)
		}
	}

	// Remove projects that are no longer present on the node.
	// Build the set of reported paths and delete anything not in it.
	if len(payload.Projects) == 0 {
		// Agent explicitly reported zero projects — clear them all
		if _, err := repo.Queries.DeleteProjectsByNode(ctx, nodeID); err != nil {
			fmt.Printf("Error clearing projects for node %d: %v\n", nodeID, err)
		}
		return
	}

	// Fetch current DB rows for this node and delete any stale ones
	existing, err := repo.Queries.ListProjectsByNode(ctx, db.ListProjectsByNodeParams{
		NodeID: nodeID,
		Limit:  1000,
		Offset: 0,
	})
	if err != nil {
		fmt.Printf("Error listing projects for node %d: %v\n", nodeID, err)
		return
	}

	reported := make(map[string]struct{}, len(payload.Projects))
	for _, p := range payload.Projects {
		reported[p.Path] = struct{}{}
	}

	for _, row := range existing {
		if _, found := reported[row.Path]; !found {
			if _, err := repo.Queries.DeleteProject(ctx, row.ID); err != nil {
				fmt.Printf("Error deleting stale project %q: %v\n", row.Path, err)
			} else {
				fmt.Printf("Stale project removed: %q (node %d)\n", row.Path, nodeID)
			}
		}
	}

}

func StoreSystemStats(ctx context.Context, repo *db.Repo, statChan chan Msg) {
	for msg := range statChan {
		sysStat := SystemStat{}
		err := sysStat.FromBytes(msg.Data)
		if err != nil {
			fmt.Println("Error unmarshalling system stat", err)
			continue
		}

		fmt.Println("Net sent ps", sysStat.NetSentPS, "Net recv ps", sysStat.NetRecvPS)

		now := time.Now().Unix() // Use Unix timestamp for SQLite

		// Start transaction for batch insert
		tx, err := repo.TimeseriesDB.Begin()
		if err != nil {
			fmt.Println("Error starting transaction:", err)
			continue
		}

		// Insert CPU stats
		for i, cpuUsage := range sysStat.CPUUsage {
			err = repo.TimeseriesQueries.WithTx(tx).InsertSystemStats(ctx, db.InsertSystemStatsParams{
				Timestamp: now,
				NodeID:    int64(msg.NodeId),
				StatType:  "cpu",
				CpuID:     sql.NullInt64{Int64: int64(i + 1), Valid: true},
				Value:     cpuUsage,
			})
			if err != nil {
				fmt.Println("Error inserting CPU stat:", err)
				tx.Rollback()
				break
			}
		}

		// Insert memory stat
		err = repo.TimeseriesQueries.WithTx(tx).InsertSystemStats(ctx, db.InsertSystemStatsParams{
			Timestamp: now,
			NodeID:    int64(msg.NodeId),
			StatType:  "mem",
			CpuID:     sql.NullInt64{Int64: 0, Valid: true},
			Value:     sysStat.MemUsage,
		})
		if err != nil {
			fmt.Println("Error inserting memory stat:", err)
			tx.Rollback()
			continue
		}

		// Insert network stats
		err = repo.TimeseriesQueries.WithTx(tx).InsertNetStats(ctx, db.InsertNetStatsParams{
			Timestamp: now,
			NodeID:    int64(msg.NodeId),
			Sent:      sysStat.NetSentPS,
			Recv:      sysStat.NetRecvPS,
		})
		if err != nil {
			fmt.Println("Error inserting net stats:", err)
			tx.Rollback()
			continue
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			fmt.Println("Error committing transaction:", err)
		}
	}
}
