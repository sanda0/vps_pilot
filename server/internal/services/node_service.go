package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/dto"
)

type NodeService interface {
	CreateNode(ip string, data string) error
	GetNodesWithSysInfo(search string, limit int32, page int32) ([]db.GetNodesWithSysInfoRow, error)
	UpdateName(nodeId int32, name string) error
	GetNode(nodeId int32) (db.Node, error)
	GetSystemStat(queryParams chan dto.NodeSystemStatRequestDto, result chan dto.SystemStatResponseDto)
}

type nodeService struct {
	repo *db.Repo
	ctx  context.Context
}

// GetSystemStat implements NodeService.
func (n *nodeService) GetSystemStat(queryParams chan dto.NodeSystemStatRequestDto, result chan dto.SystemStatResponseDto) {
	for query := range queryParams {
		fmt.Println("Query received", query)
		node, err := n.repo.Queries.GetNodeWithSysInfo(n.ctx, query.ID)
		if err != nil {
			fmt.Println("Error getting node", err)
			continue
		}
		cpuStats := make(map[int][]db.GetSystemStatsRow)
		for i := 1; i <= int(node.Cpus.Int32); i++ {
			cpustat, err := n.repo.Queries.GetSystemStats(n.ctx, db.GetSystemStatsParams{
				NodeID:   query.ID,
				StatType: "cpu",
				CpuID:    int32(i),
				Column4: sql.NullString{
					String: query.TimeRange,
					Valid:  true,
				},
			})
			if err != nil {
				fmt.Println("Error getting cpu stats", err)
				continue
			}
			cpuStats[i] = cpustat
		}
		memStat, err := n.repo.Queries.GetSystemStats(n.ctx, db.GetSystemStatsParams{
			NodeID:   query.ID,
			StatType: "mem",
			Column4: sql.NullString{
				String: query.TimeRange,
				Valid:  true,
			},
		})
		if err != nil {
			fmt.Println("Error getting mem stats", err)
			continue
		}
		result <- dto.SystemStatResponseDto{
			NodeID:    query.ID,
			TimeRange: query.TimeRange,
			Cpu:       cpuStats,
			Mem:       memStat,
		}
	}

	fmt.Println("Query processing done")
}

// GetNode implements NodeService.
func (n *nodeService) GetNode(nodeId int32) (db.Node, error) {
	node, err := n.repo.Queries.GetNode(n.ctx, nodeId)
	if err != nil {
		return db.Node{}, err
	}
	return node, nil
}

// UpdateName implements NodeService.
func (n *nodeService) UpdateName(nodeId int32, name string) error {
	err := n.repo.Queries.UpdateNodeName(n.ctx, db.UpdateNodeNameParams{
		ID: nodeId,
		Name: sql.NullString{
			String: name,
			Valid:  true,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

// GetNodesWithSysInfo implements NodeService.
func (n *nodeService) GetNodesWithSysInfo(search string, limit int32, page int32) ([]db.GetNodesWithSysInfoRow, error) {
	offset := (page - 1) * limit
	nodes, err := n.repo.Queries.GetNodesWithSysInfo(n.ctx, db.GetNodesWithSysInfoParams{
		Column1: sql.NullString{
			String: search,
			Valid:  true,
		},
		Limit:  limit,
		Offset: offset,
	})
	fmt.Println(nodes)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// CreateNode implements NodeService.
func (n *nodeService) CreateNode(ip string, data string) error {
	panic("unimplemented")
}

func NewNodeService(ctx context.Context, repo *db.Repo) NodeService {
	return &nodeService{
		repo: repo,
		ctx:  ctx,
	}
}
