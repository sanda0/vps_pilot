package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path"
	"strings"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/dto"
)

var (
	ErrNodeNotFound = errors.New("node not found")
	ErrPathNotFound = errors.New("project path not found")
	ErrPathExists   = errors.New("this path is already added for the node")
	ErrInvalidPath  = errors.New("path must be an absolute directory, e.g. /var/www/app")
)

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
	// AddPath and DeletePath manage project directories added by hand. The
	// agent decides whether to accept them (inside its --projects-dir).
	AddPath(nodeID int64, p string) (*dto.ProjectPathDto, error)
	DeletePath(nodeID, pathID int64) error
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

func (s *projectScanService) AddPath(nodeID int64, p string) (*dto.ProjectPathDto, error) {
	p = strings.TrimSpace(p)
	if !path.IsAbs(p) || len(p) > 4096 || strings.ContainsRune(p, 0) {
		return nil, ErrInvalidPath
	}
	if _, err := s.Get(nodeID); err != nil {
		return nil, err
	}
	row, err := s.repo.Queries.AddNodeProjectPath(s.ctx, db.AddNodeProjectPathParams{NodeID: nodeID, Path: path.Clean(p)})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrPathExists
		}
		return nil, err
	}
	s.agents.PushProjectSettings(s.ctx, nodeID)
	return &dto.ProjectPathDto{ID: row.ID, Path: row.Path, CreatedAt: row.CreatedAt}, nil
}

func (s *projectScanService) DeletePath(nodeID, pathID int64) error {
	n, err := s.repo.Queries.DeleteNodeProjectPath(s.ctx, db.DeleteNodeProjectPathParams{ID: pathID, NodeID: nodeID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrPathNotFound
	}
	s.agents.PushProjectSettings(s.ctx, nodeID)
	return nil
}
