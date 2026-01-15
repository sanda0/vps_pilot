package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/dto"
)

type ProjectService interface {
	CreateProject(req *dto.CreateProjectRequest) (*dto.ProjectResponse, error)
	GetProject(id string) (*dto.ProjectResponse, error)
	ListProjects(limit, offset int32) ([]*dto.ProjectResponse, error)
	ListProjectsByNode(nodeID int32, limit, offset int32) ([]*dto.ProjectResponse, error)
	UpdateProject(id string, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error)
	UpdateProjectStatus(id, status string) (*dto.ProjectResponse, error)
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

// CreateProject creates a new project
func (s *projectService) CreateProject(req *dto.CreateProjectRequest) (*dto.ProjectResponse, error) {
	// Validate node exists
	_, err := s.repo.Queries.GetNode(s.ctx, int64(req.NodeID))
	if err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	// Set default branch if empty
	if req.Branch == "" {
		req.Branch = "main"
	}

	project, err := s.repo.Queries.CreateProject(s.ctx, db.CreateProjectParams{
		Name: req.Name,
		Description: sql.NullString{
			String: req.Description,
			Valid:  req.Description != "",
		},
		NodeID: int64(req.NodeID),
		RepoUrl: sql.NullString{
			String: req.RepoURL,
			Valid:  req.RepoURL != "",
		},
		Branch: sql.NullString{
			String: req.Branch,
			Valid:  true,
		},
		DeployPath: req.DeployPath,
		Status: sql.NullString{
			String: "inactive",
			Valid:  true,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return dto.ConvertToProjectResponse(&project), nil
}

// GetProject retrieves a project with node information
func (s *projectService) GetProject(id string) (*dto.ProjectResponse, error) {
	project, err := s.repo.Queries.GetProjectWithNode(s.ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return dto.ConvertToProjectWithNodeResponse(&project), nil
}

// ListProjects retrieves all projects with pagination
func (s *projectService) ListProjects(limit, offset int32) ([]*dto.ProjectResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	projects, err := s.repo.Queries.ListProjectsWithNodes(s.ctx, db.ListProjectsWithNodesParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return dto.ConvertToProjectListResponse(projects), nil
}

// ListProjectsByNode retrieves projects for a specific node
func (s *projectService) ListProjectsByNode(nodeID int32, limit, offset int32) ([]*dto.ProjectResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	projects, err := s.repo.Queries.ListProjectsByNode(s.ctx, db.ListProjectsByNodeParams{
		NodeID: int64(nodeID),
		Limit:  int64(limit),
		Offset: int64(offset),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list projects by node: %w", err)
	}

	// Convert to response format
	responses := make([]*dto.ProjectResponse, len(projects))
	for i, project := range projects {
		responses[i] = dto.ConvertToProjectResponse(&project)
	}

	return responses, nil
}

// UpdateProject updates an existing project
func (s *projectService) UpdateProject(id string, req *dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
	// Check if project exists
	_, err := s.repo.Queries.GetProject(s.ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	project, err := s.repo.Queries.UpdateProject(s.ctx, db.UpdateProjectParams{
		Name: req.Name,
		Description: sql.NullString{
			String: req.Description,
			Valid:  req.Description != "",
		},
		RepoUrl: sql.NullString{
			String: req.RepoURL,
			Valid:  req.RepoURL != "",
		},
		Branch: sql.NullString{
			String: req.Branch,
			Valid:  req.Branch != "",
		},
		DeployPath: req.DeployPath,
		Status: sql.NullString{
			String: req.Status,
			Valid:  req.Status != "",
		},
		ID: id,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return dto.ConvertToProjectResponse(&project), nil
}

// UpdateProjectStatus updates only the status of a project
func (s *projectService) UpdateProjectStatus(id, status string) (*dto.ProjectResponse, error) {
	project, err := s.repo.Queries.UpdateProjectStatus(s.ctx, db.UpdateProjectStatusParams{
		Status: sql.NullString{
			String: status,
			Valid:  true,
		},
		ID: id,
	})

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("project not found")
		}
		return nil, fmt.Errorf("failed to update project status: %w", err)
	}

	return dto.ConvertToProjectResponse(&project), nil
}

// DeleteProject deletes a project
func (s *projectService) DeleteProject(id string) error {
	rowsAffected, err := s.repo.Queries.DeleteProject(s.ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("project not found")
	}

	return nil
}

// CountProjects returns the total number of projects
func (s *projectService) CountProjects() (int64, error) {
	count, err := s.repo.Queries.CountProjects(s.ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count projects: %w", err)
	}
	return count, nil
}

// CountProjectsByNode returns the number of projects for a specific node
func (s *projectService) CountProjectsByNode(nodeID int32) (int64, error) {
	count, err := s.repo.Queries.CountProjectsByNode(s.ctx, int64(nodeID))
	if err != nil {
		return 0, fmt.Errorf("failed to count projects by node: %w", err)
	}
	return count, nil
}
