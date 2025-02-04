package tcpserver

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sanda0/vps_pilot/db"
)

func CreateNode(ctx context.Context, repo *db.Repo, ip string, data []byte) {
	node, err := repo.Queries.CreateNode(ctx, db.CreateNodeParams{
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

}
