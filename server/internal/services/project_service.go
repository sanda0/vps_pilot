package services

import (
	"context"
	"fmt"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/dto"
)

type ProjectService interface {
	UpsertFromAgent(req *dto.AgentProjectSyncRequest) (*dto.ProjectResponse, error)
	BulkSyncFromAgent(req *dto.AgentProjectsBulkSyncRequest) ([]*dto.ProjectResponse, error)
	GetProject(id string) (*dto.ProjectResponse, error)
	ListProjects(limit, offset int32) ([]*dto.ProjectResponse, error)
	ListProjectsByNode(nodeID int32, limit, offset int32) ([]*dto.ProjectResponse, error)
	DeleteProject(id string) error
	CountProjects() (int64, error)
	CountProjectsByNode(nodeID int32) (int64, error)
}

type projectService struct {
	repo *db.Repo
	ctx  context.Context
}

func NewProjectService(repo *db.Repo, ctx context.Context) ProjectService {
	return &projectService{
		repo: repo,
		ctx:  ctx,
	}
}

// UpsertFromAgent inserts or updates a single project reported by an agent.
func (s *projectService) UpsertFromAgent(req *dto.AgentProjectSyncRequest) (*dto.ProjectResponse, error) {
	// Validate node exists
	_, err := s.repo.Queries.GetNode(s.ctx, req.NodeID)
	if err != nil {
		return nil, fmt.Errorf("node %d not found: %w", req.NodeID, err)
	}

	project, err := s.repo.Queries.UpsertProject(s.ctx, db.UpsertProjectParams{
		NodeID:   req.NodeID,
		Name:     req.Name,
		Path:     req.Path,
		Tech:     dto.MarshalTech(req.Tech),
		Commands: dto.MarshalCommands(req.Commands),
		Logs:     dto.MarshalLogs(req.Logs),
		Backups:  dto.MarshalBackups(req.Backups),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert project: %w", err)
	}

	return dto.ConvertToProjectResponse(&project), nil
}

// BulkSyncFromAgent replaces all projects for a node with the provided list.
// This is the preferred method: the agent sends the full current list and the
// server reconciles (upsert all reported, delete any that are no longer present).
func (s *projectService) BulkSyncFromAgent(req *dto.AgentProjectsBulkSyncRequest) ([]*dto.ProjectResponse, error) {
	// Validate node exists
	_, err := s.repo.Queries.GetNode(s.ctx, req.NodeID)
	if err != nil {
		return nil, fmt.Errorf("node %d not found: %w", req.NodeID, err)
	}

	// Upsert every reported project and collect results
	results := make([]*dto.ProjectResponse, 0, len(req.Projects))
	for _, p := range req.Projects {
		// Force the node ID from the top-level field so the agent cannot
		// accidentally send mismatched node IDs inside the array.
		p.NodeID = req.NodeID

		project, err := s.repo.Queries.UpsertProject(s.ctx, db.UpsertProjectParams{
			NodeID:   p.NodeID,
			Name:     p.Name,
			Path:     p.Path,
			Tech:     dto.MarshalTech(p.Tech),
			Commands: dto.MarshalCommands(p.Commands),
			Logs:     dto.MarshalLogs(p.Logs),
			Backups:  dto.MarshalBackups(p.Backups),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to upsert project %q: %w", p.Path, err)
		}
		results = append(results, dto.ConvertToProjectResponse(&project))
	}

	return results, nil
}

// GetProject retrieves a single project with its node information.
func (s *projectService) GetProject(id string) (*dto.ProjectResponse, error) {
	row, err := s.repo.Queries.GetProjectWithNode(s.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("project not found")
	}
	return dto.ConvertToProjectWithNodeResponse(&row), nil
}

// ListProjects retrieves all projects (across all nodes) with pagination.
func (s *projectService) ListProjects(limit, offset int32) ([]*dto.ProjectResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := s.repo.Queries.ListProjectsWithNodes(s.ctx, db.ListProjectsWithNodesParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return dto.ConvertToProjectListResponse(rows), nil
}

// ListProjectsByNode retrieves projects for a specific node with pagination.
func (s *projectService) ListProjectsByNode(nodeID int32, limit, offset int32) ([]*dto.ProjectResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := s.repo.Queries.ListProjectsByNode(s.ctx, db.ListProjectsByNodeParams{
		NodeID: int64(nodeID),
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list projects by node: %w", err)
	}

	responses := make([]*dto.ProjectResponse, len(rows))
	for i := range rows {
		responses[i] = dto.ConvertToProjectResponse(&rows[i])
	}
	return responses, nil
}

// DeleteProject removes a project record by ID.
func (s *projectService) DeleteProject(id string) error {
	rows, err := s.repo.Queries.DeleteProject(s.ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("project not found")
	}
	return nil
}

// CountProjects returns the total number of projects across all nodes.
func (s *projectService) CountProjects() (int64, error) {
	count, err := s.repo.Queries.CountProjects(s.ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count projects: %w", err)
	}
	return count, nil
}

// CountProjectsByNode returns the number of projects for a specific node.
func (s *projectService) CountProjectsByNode(nodeID int32) (int64, error) {
	count, err := s.repo.Queries.CountProjectsByNode(s.ctx, int64(nodeID))
	if err != nil {
		return 0, fmt.Errorf("failed to count projects by node: %w", err)
	}
	return count, nil
}
