package services

import (
	"context"
	"database/sql"
	"errors"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/dto"
	"github.com/sanda0/vps_pilot/internal/tcpserver"
)

var ErrNoLogFolders = errors.New("the project's config.vpspilot.json declares no log folders")

// LogReader lists and streams log files on agents (implemented by
// tcpserver.Logs).
type LogReader interface {
	List(ctx context.Context, nodeID int64, projectPath string) (tcpserver.LogFiles, error)
	Open(req tcpserver.OpenLog) (<-chan tcpserver.LogEvent, func(), error)
}

// LogService reads the files in a project's log folders. Nothing is stored.
type LogService interface {
	List(ctx context.Context, projectID string) (dto.LogFilesDto, error)
	// Open streams a file; see tcpserver.Logs.Open.
	Open(projectID string, req dto.OpenLogRequest) (<-chan tcpserver.LogEvent, func(), error)
}

type logService struct {
	ctx    context.Context
	repo   *db.Repo
	reader LogReader
}

func NewLogService(ctx context.Context, repo *db.Repo, reader LogReader) LogService {
	return &logService{ctx: ctx, repo: repo, reader: reader}
}

// project returns a project that declares log folders, as the hub last saw
// its config. The agent checks again against the file itself.
func (s *logService) project(projectID string) (db.Project, error) {
	p, err := s.repo.Queries.GetProject(s.ctx, projectID)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrProjectNotFound
	}
	if err == nil && len(dto.ConvertToProjectResponse(&p).Logs) == 0 {
		return p, ErrNoLogFolders
	}
	return p, err
}

func (s *logService) List(ctx context.Context, projectID string) (dto.LogFilesDto, error) {
	p, err := s.project(projectID)
	if err != nil {
		return dto.LogFilesDto{}, err
	}
	files, err := s.reader.List(ctx, p.NodeID, p.Path)
	if err != nil {
		return dto.LogFilesDto{}, err
	}
	if files.Files == nil {
		files.Files = []tcpserver.LogFile{}
	}
	return dto.LogFilesDto{Files: files.Files, Error: files.Error}, nil
}

func (s *logService) Open(projectID string, req dto.OpenLogRequest) (<-chan tcpserver.LogEvent, func(), error) {
	p, err := s.project(projectID)
	if err != nil {
		return nil, nil, err
	}
	return s.reader.Open(tcpserver.OpenLog{
		NodeID: p.NodeID, ProjectPath: p.Path, File: req.File,
		Before: req.Before, MaxBytes: req.MaxBytes, Follow: req.Follow,
	})
}
