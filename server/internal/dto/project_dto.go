package dto

import (
	"encoding/json"
	"time"

	"github.com/sanda0/vps_pilot/internal/db"
)

// ProjectCommand represents a runnable command defined in config.vpspilot.json
type ProjectCommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// ProjectBackupDatabase holds DB connection info for backups
type ProjectBackupDatabase struct {
	Connection   string `json:"connection"`
	Host         string `json:"host"`
	Port         string `json:"port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	DatabaseName string `json:"database_name"`
}

// ProjectBackup holds the full backup configuration
type ProjectBackup struct {
	EnvFile     string                 `json:"env_file,omitempty"`
	ZipFileName string                 `json:"zip_file_name,omitempty"`
	Database    *ProjectBackupDatabase `json:"database,omitempty"`
	Dirs        []string               `json:"dir,omitempty"`
}

// AgentProjectSyncRequest is what the agent POSTs when it discovers/updates a project.
// It maps directly to config.vpspilot.json fields plus the node context.
type AgentProjectSyncRequest struct {
	NodeID   int64            `json:"node_id"   binding:"required"`
	Name     string           `json:"name"      binding:"required"`
	Path     string           `json:"path"      binding:"required"` // absolute path on disk
	Tech     []string         `json:"tech"`
	Commands []ProjectCommand `json:"commands"`
	Logs     []string         `json:"logs"`
	Backups  *ProjectBackup   `json:"backups"`
}

// AgentProjectsBulkSyncRequest allows the agent to sync all projects for a node in one call.
type AgentProjectsBulkSyncRequest struct {
	NodeID   int64                     `json:"node_id"  binding:"required"`
	Projects []AgentProjectSyncRequest `json:"projects" binding:"required"`
}

// ProjectResponse is what the dashboard API returns.
type ProjectResponse struct {
	ID           string           `json:"id"`
	NodeID       int64            `json:"node_id"`
	NodeName     string           `json:"node_name,omitempty"`
	NodeIP       string           `json:"node_ip,omitempty"`
	Name         string           `json:"name"`
	Path         string           `json:"path"`
	Tech         []string         `json:"tech"`
	Commands     []ProjectCommand `json:"commands"`
	Logs         []string         `json:"logs"`
	Backups      *ProjectBackup   `json:"backups"`
	DiscoveredAt time.Time        `json:"discovered_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// ConvertToProjectResponse converts a db.Project to ProjectResponse.
func ConvertToProjectResponse(p *db.Project) *ProjectResponse {
	return &ProjectResponse{
		ID:           p.ID,
		NodeID:       p.NodeID,
		Name:         p.Name,
		Path:         p.Path,
		Tech:         parseStringSlice(p.Tech),
		Commands:     parseCommands(p.Commands),
		Logs:         parseStringSlice(p.Logs),
		Backups:      parseBackup(p.Backups),
		DiscoveredAt: time.Unix(p.DiscoveredAt, 0),
		UpdatedAt:    time.Unix(p.UpdatedAt, 0),
	}
}

// ConvertToProjectWithNodeResponse converts a db.GetProjectWithNodeRow to ProjectResponse.
func ConvertToProjectWithNodeResponse(row *db.GetProjectWithNodeRow) *ProjectResponse {
	return &ProjectResponse{
		ID:           row.ID,
		NodeID:       row.NodeID,
		NodeName:     row.NodeName.String,
		NodeIP:       row.NodeIp.String,
		Name:         row.Name,
		Path:         row.Path,
		Tech:         parseStringSlice(row.Tech),
		Commands:     parseCommands(row.Commands),
		Logs:         parseStringSlice(row.Logs),
		Backups:      parseBackup(row.Backups),
		DiscoveredAt: time.Unix(row.DiscoveredAt, 0),
		UpdatedAt:    time.Unix(row.UpdatedAt, 0),
	}
}

// ConvertToProjectListResponse converts a slice of db.ListProjectsWithNodesRow to ProjectResponse slice.
func ConvertToProjectListResponse(rows []db.ListProjectsWithNodesRow) []*ProjectResponse {
	projects := make([]*ProjectResponse, len(rows))
	for i, row := range rows {
		projects[i] = &ProjectResponse{
			ID:           row.ID,
			NodeID:       row.NodeID,
			NodeName:     row.NodeName.String,
			NodeIP:       row.NodeIp.String,
			Name:         row.Name,
			Path:         row.Path,
			Tech:         parseStringSlice(row.Tech),
			Commands:     parseCommands(row.Commands),
			Logs:         parseStringSlice(row.Logs),
			Backups:      parseBackup(row.Backups),
			DiscoveredAt: time.Unix(row.DiscoveredAt, 0),
			UpdatedAt:    time.Unix(row.UpdatedAt, 0),
		}
	}
	return projects
}

// MarshalTech serialises a []string to a JSON string for DB storage.
func MarshalTech(tech []string) string {
	if tech == nil {
		return "[]"
	}
	b, _ := json.Marshal(tech)
	return string(b)
}

// MarshalCommands serialises []ProjectCommand to a JSON string for DB storage.
func MarshalCommands(commands []ProjectCommand) string {
	if commands == nil {
		return "[]"
	}
	b, _ := json.Marshal(commands)
	return string(b)
}

// MarshalLogs serialises a []string to a JSON string for DB storage.
func MarshalLogs(logs []string) string {
	if logs == nil {
		return "[]"
	}
	b, _ := json.Marshal(logs)
	return string(b)
}

// MarshalBackups serialises a *ProjectBackup to a JSON string for DB storage.
func MarshalBackups(backup *ProjectBackup) string {
	if backup == nil {
		return "{}"
	}
	b, _ := json.Marshal(backup)
	return string(b)
}

// --- helpers ----------------------------------------------------------------

func parseStringSlice(raw string) []string {
	if raw == "" || raw == "null" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}
	}
	return out
}

func parseCommands(raw string) []ProjectCommand {
	if raw == "" || raw == "null" {
		return []ProjectCommand{}
	}
	var out []ProjectCommand
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []ProjectCommand{}
	}
	return out
}

func parseBackup(raw string) *ProjectBackup {
	if raw == "" || raw == "null" || raw == "{}" {
		return nil
	}
	var out ProjectBackup
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return &out
}
