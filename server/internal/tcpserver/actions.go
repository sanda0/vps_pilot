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
		Cpus: sql.NullInt32{
			Int32: int32(sysInfo.CPUs),
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
		}
		fmt.Println("System stat received", sysStat)

		var times []time.Time
		var nodeIDs []int32
		var statTypes []string
		var cpuIDs []int32
		var values []float64

		for i, cpuUsage := range sysStat.CPUUsage {
			times = append(times, time.Now())
			nodeIDs = append(nodeIDs, msg.NodeId)
			statTypes = append(statTypes, "cpu")
			cpuIDs = append(cpuIDs, int32(i+1))
			values = append(values, cpuUsage)
		}
		times = append(times, time.Now())
		nodeIDs = append(nodeIDs, msg.NodeId)
		statTypes = append(statTypes, "mem")
		cpuIDs = append(cpuIDs, 0)
		values = append(values, sysStat.MemUsage)

		err = repo.Queries.InsertSystemStats(ctx, db.InsertSystemStatsParams{
			Column1: times,
			Column2: nodeIDs,
			Column3: statTypes,
			Column4: cpuIDs,
			Column5: values,
		})

		if err != nil {
			fmt.Println("Error inserting system stats", err)
		}

	}
}
