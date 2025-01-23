package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Project struct {
	gorm.Model
	Name                string
	FolderPath          string
	WebServer           string // Apache, Nginx, etc
	WebServerConfigPath string
	Description         string
	RepoURL             string
	ProdURL             string
	PAT                 string // Personal Access Token for the repo github or gitlab
	WebHookID           uuid.UUID
	ServerID            uint
}

type ProjectStat struct {
	gorm.Model
	ProjectID  uint
	CpuLoad    float64
	MemoryUsed float64
	DiskUsed   float64
}

type GitAction struct {
	gorm.Model
	ProjectID uint
	Name      string
	Event     string // push, pull_request, etc
	Branch    string
	Commands  []string
}

type GitActionCommadOutput struct {
	gorm.Model
	GitActionsID uint
	Command      string
	Output       string
}
