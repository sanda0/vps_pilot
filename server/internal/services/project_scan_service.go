package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/dto"
)

// ErrNodeNotFound is returned for an unknown node ID.
var ErrNodeNotFound = errors.New("node not found")

// ProjectAgents reaches live agents about projects (implemented by tcpserver.Registry).
type ProjectAgents interface {
	PushProjectSettings(ctx context.Context, nodeID int64)
	RequestProjectRescan(nodeID int64) (bool, error)
	Online(nodeID int64) bool
}

// ProjectScanService manages how each node searches for projects.
type ProjectScanService interface {
	Get(nodeID int64) (*dto.ProjectScanDto, error)
	UpdateSettings(nodeID int64, req dto.ProjectScanSettingsRequest) (*dto.ProjectScanDto, error)
	// Rescan asks the node's agent to search now. It reports false if the node is offline.
	Rescan(nodeID int64) (bool, error)
}

type projectScanService struct {
	ctx    context.Context
	repo   *db.Repo
	agents ProjectAgents
}

func NewProjectScanService(ctx context.Context, repo *db.Repo, agents ProjectAgents) ProjectScanService {
	return &projectScanService{ctx: ctx, repo: repo, agents: agents}
}

func (s *projectScanService) Get(nodeID int64) (*dto.ProjectScanDto, error) {
	scan, err := s.repo.Queries.GetNodeProjectScan(s.ctx, nodeID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNodeNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.Queries.ListNodeProjectPaths(s.ctx, nodeID)
	if err != nil {
		return nil, err
	}
	paths := make([]dto.ProjectPathDto, 0, len(rows))
	for _, r := range rows {
		paths = append(paths, dto.ProjectPathDto{ID: r.ID, Path: r.Path, CreatedAt: r.CreatedAt})
	}
	out := &dto.ProjectScanDto{
		NodeID:       nodeID,
		AutoScan:     scan.AutoScan != 0,
		ScanInterval: scan.ScanInterval,
		Errors:       json.RawMessage(scan.Errors),
		Paths:        paths,
		Online:       s.agents.Online(nodeID),
	}
	if scan.LastScannedAt.Valid {
		out.LastScannedAt = &scan.LastScannedAt.Int64
	}
	return out, nil
}

func (s *projectScanService) UpdateSettings(nodeID int64, req dto.ProjectScanSettingsRequest) (*dto.ProjectScanDto, error) {
	if _, err := s.Get(nodeID); err != nil {
		return nil, err
	}
	autoScan := int64(0)
	if *req.AutoScan {
		autoScan = 1
	}
	if err := s.repo.Queries.UpdateNodeProjectScanSettings(s.ctx, db.UpdateNodeProjectScanSettingsParams{
		NodeID: nodeID, AutoScan: autoScan, ScanInterval: req.ScanInterval,
	}); err != nil {
		return nil, err
	}
	s.agents.PushProjectSettings(s.ctx, nodeID)
	return s.Get(nodeID)
}

func (s *projectScanService) Rescan(nodeID int64) (bool, error) {
	if _, err := s.Get(nodeID); err != nil {
		return false, err
	}
	return s.agents.RequestProjectRescan(nodeID)
}
