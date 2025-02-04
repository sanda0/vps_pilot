package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sanda0/vps_pilot/db"
)

type NodeService interface {
	CreateNode(ip string, data string) error
	GetNodesWithSysInfo(search string, limit int32, page int32) ([]db.GetNodesWithSysInfoRow, error)
	UpdateName(nodeId int32, name string) (*db.Node, error)
}

type nodeService struct {
	repo *db.Repo
	ctx  context.Context
}

// UpdateName implements NodeService.
func (n *nodeService) UpdateName(nodeId int32, name string) (*db.Node, error) {
	node, err := n.repo.Queries.UpdateNodeName(n.ctx, db.UpdateNodeNameParams{
		ID: nodeId,
		Name: sql.NullString{
			String: name,
			Valid:  true,
		},
	})
	if err != nil {
		return nil, err
	}
	return &node, nil
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
