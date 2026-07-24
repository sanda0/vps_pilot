package tcpserver

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sanda0/vps_pilot/internal/db"
)

func CreateNode(ctx context.Context, repo *db.Repo, ip string, data []byte) (*db.Node, error) {

	node, err := repo.Queries.GetNodeByIP(ctx, ip)
	if err == nil {
		return &node, nil
	}

	node, err = repo.Queries.CreateNode(ctx, db.CreateNodeParams{
		Name: sql.NullString{
			String: "node-" + ip,
			Valid:  true,
		},
		Ip: ip,
	})
	if err != nil {
		fmt.Println("Error creating node", err)
	}
	fmt.Println("Node created", node)
	sysInfo := SystemInfo{}
	err = sysInfo.FromBytes(data)
	if err != nil {
		fmt.Println("Error unmarshalling system info", err)
	}
	sysInfoDb, err := repo.Queries.AddNodeSysInfo(ctx, db.AddNodeSysInfoParams{
		NodeID: node.ID,
		Os: sql.NullString{
			String: sysInfo.OS,
			Valid:  true,
		},
		Platform: sql.NullString{
			String: sysInfo.Platform,
			Valid:  true,
		},
		PlatformVersion: sql.NullString{
			String: sysInfo.PlatformVersion,
			Valid:  true,
		},
		KernelVersion: sql.NullString{
			String: sysInfo.KernelVersion,
			Valid:  true,
		},
		Cpus: sql.NullInt64{
			Int64: int64(sysInfo.CPUs),
			Valid: true,
		},
		TotalMemory: sql.NullFloat64{
			Float64: float64(sysInfo.TotalMemory),
			Valid:   true,
		},
	})
	if err != nil {
		fmt.Println("Error adding node sys info", err)
	}
	fmt.Println("Node sys info added", sysInfoDb)
	return &node, nil
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
