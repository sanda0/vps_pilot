package services

import (
	"context"
	"database/sql"
	"errors"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/dto"
	"github.com/sanda0/vps_pilot/internal/tcpserver"
)

var ErrNoBackupConfig = errors.New("the project's config.vpspilot.json declares no backups")

// Backupper starts backups on agents (implemented by tcpserver.Backups).
type Backupper interface {
	Start(ctx context.Context, req tcpserver.StartBackup) (db.BackupRun, error)
}

// BackupService starts a project's backups and lists them.
type BackupService interface {
	Start(projectID string, userID int64) (dto.BackupRunDto, error)
	ListByProject(projectID string, limit, offset int64) ([]dto.BackupRunDto, error)
}

type backupService struct {
	ctx     context.Context
	repo    *db.Repo
	backups Backupper
}

func NewBackupService(ctx context.Context, repo *db.Repo, backups Backupper) BackupService {
	return &backupService{ctx: ctx, repo: repo, backups: backups}
}

func (s *backupService) project(projectID string) (db.Project, error) {
	p, err := s.repo.Queries.GetProject(s.ctx, projectID)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrProjectNotFound
	}
	return p, err
}

// Start checks that the project declares backups, as the hub last saw its
// config, for a clear error. The agent checks again against the file.
func (s *backupService) Start(projectID string, userID int64) (dto.BackupRunDto, error) {
	p, err := s.project(projectID)
	if err != nil {
		return dto.BackupRunDto{}, err
	}
	if dto.ConvertToProjectResponse(&p).Backups == nil {
		return dto.BackupRunDto{}, ErrNoBackupConfig
	}
	user, err := s.repo.Queries.FindUserById(s.ctx, userID)
	if err != nil {
		return dto.BackupRunDto{}, err
	}
	run, err := s.backups.Start(s.ctx, tcpserver.StartBackup{
		NodeID: p.NodeID, ProjectPath: p.Path, UserID: user.ID, Username: user.Username,
	})
	if err != nil {
		return dto.BackupRunDto{}, err
	}
	return dto.NewBackupRunDto(run), nil
}

func (s *backupService) ListByProject(projectID string, limit, offset int64) ([]dto.BackupRunDto, error) {
	p, err := s.project(projectID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.Queries.ListBackupRunsByProject(s.ctx, db.ListBackupRunsByProjectParams{
		NodeID: p.NodeID, ProjectPath: p.Path, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]dto.BackupRunDto, len(rows))
	for i, r := range rows {
		out[i] = dto.NewBackupRunDto(r)
	}
	return out, nil
}
