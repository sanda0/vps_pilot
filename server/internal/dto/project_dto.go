package dto

import (
	"time"

	"github.com/sanda0/vps_pilot/internal/db"
)

// CreateProjectRequest represents the request to create a new project
type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
	NodeID      int32  `json:"node_id" binding:"required"`
	RepoURL     string `json:"repo_url" binding:"omitempty,url"`
	Branch      string `json:"branch" binding:"required"`
	DeployPath  string `json:"deploy_path" binding:"required,min=1"`
}

// UpdateProjectRequest represents the request to update a project
type UpdateProjectRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"max=500"`
	RepoURL     string `json:"repo_url" binding:"omitempty,url"`
	Branch      string `json:"branch" binding:"required"`
	DeployPath  string `json:"deploy_path" binding:"required,min=1"`
	Status      string `json:"status" binding:"omitempty,oneof=inactive cloning active error"`
}

// ProjectResponse represents a project with additional node information
type ProjectResponse struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	NodeID         int32      `json:"node_id"`
	NodeName       string     `json:"node_name,omitempty"`
	NodeIP         string     `json:"node_ip,omitempty"`
	RepoURL        string     `json:"repo_url"`
	Branch         string     `json:"branch"`
	DeployPath     string     `json:"deploy_path"`
	Status         string     `json:"status"`
	LastDeployedAt *time.Time `json:"last_deployed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ConvertToProjectResponse converts a db.Project to ProjectResponse
func ConvertToProjectResponse(p *db.Project) *ProjectResponse {
	var lastDeployed *time.Time
	if p.LastDeployedAt.Valid {
		t := time.Unix(p.LastDeployedAt.Int64, 0)
		lastDeployed = &t
	}

	return &ProjectResponse{
		ID:             p.ID,
		Name:           p.Name,
		Description:    p.Description.String,
		NodeID:         int32(p.NodeID),
		RepoURL:        p.RepoUrl.String,
		Branch:         p.Branch.String,
		DeployPath:     p.DeployPath,
		Status:         p.Status.String,
		LastDeployedAt: lastDeployed,
		CreatedAt:      time.Unix(p.CreatedAt, 0),
		UpdatedAt:      time.Unix(p.UpdatedAt, 0),
	}
}

// ConvertToProjectWithNodeResponse converts a db.GetProjectWithNodeRow to ProjectResponse
func ConvertToProjectWithNodeResponse(row *db.GetProjectWithNodeRow) *ProjectResponse {
	var lastDeployed *time.Time
	if row.LastDeployedAt.Valid {
		t := time.Unix(row.LastDeployedAt.Int64, 0)
		lastDeployed = &t
	}

	return &ProjectResponse{
		ID:             row.ID,
		Name:           row.Name,
		Description:    row.Description.String,
		NodeID:         int32(row.NodeID),
		NodeName:       row.NodeName.String,
		NodeIP:         row.NodeIp.String,
		RepoURL:        row.RepoUrl.String,
		Branch:         row.Branch.String,
		DeployPath:     row.DeployPath,
		Status:         row.Status.String,
		LastDeployedAt: lastDeployed,
		CreatedAt:      time.Unix(row.CreatedAt, 0),
		UpdatedAt:      time.Unix(row.UpdatedAt, 0),
	}
}

// ConvertToProjectListResponse converts a list of db.ListProjectsWithNodesRow to ProjectResponse slice
func ConvertToProjectListResponse(rows []db.ListProjectsWithNodesRow) []*ProjectResponse {
	projects := make([]*ProjectResponse, len(rows))
	for i, row := range rows {
		var lastDeployed *time.Time
		if row.LastDeployedAt.Valid {
			t := time.Unix(row.LastDeployedAt.Int64, 0)
			lastDeployed = &t
		}

		projects[i] = &ProjectResponse{
			ID:             row.ID,
			Name:           row.Name,
			Description:    row.Description.String,
			NodeID:         int32(row.NodeID),
			NodeName:       row.NodeName.String,
			NodeIP:         row.NodeIp.String,
			RepoURL:        row.RepoUrl.String,
			Branch:         row.Branch.String,
			DeployPath:     row.DeployPath,
			Status:         row.Status.String,
			LastDeployedAt: lastDeployed,
			CreatedAt:      time.Unix(row.CreatedAt, 0),
			UpdatedAt:      time.Unix(row.UpdatedAt, 0),
		}
	}
	return projects
}
