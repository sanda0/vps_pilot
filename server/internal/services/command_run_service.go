package services

import (
	"context"
	"database/sql"
	"errors"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/dto"
	"github.com/sanda0/vps_pilot/internal/tcpserver"
)

var (
	ErrProjectNotFound = errors.New("project not found")
	ErrRunNotFound     = errors.New("command run not found")
	ErrUnknownCommand  = errors.New("the project's config.vpspilot.json has no command with that name")
)

// CommandRunner starts runs on agents and follows them (implemented by
// tcpserver.Runs).
type CommandRunner interface {
	Start(ctx context.Context, req tcpserver.StartRun) (db.CommandRun, error)
	Cancel(runID string) error
	Watch(runID string) (<-chan tcpserver.RunEvent, func())
}

// CommandRunService runs project commands by name and reads the audit log.
type CommandRunService interface {
	Start(projectID, command string, userID int64) (dto.CommandRunDto, error)
	Cancel(runID string) error
	Get(runID string) (dto.CommandRunDto, []dto.CommandOutputDto, error)
	ListByProject(projectID string, limit, offset int64) ([]dto.CommandRunDto, error)
	// Watch follows a run: call it before Get, then skip output already read.
	Watch(runID string) (<-chan tcpserver.RunEvent, func())
	// OutputAfter returns the stored output with seq greater than after.
	OutputAfter(runID string, after int64) ([]dto.CommandOutputDto, error)
}

type commandRunService struct {
	ctx    context.Context
	repo   *db.Repo
	runner CommandRunner
}

func NewCommandRunService(ctx context.Context, repo *db.Repo, runner CommandRunner) CommandRunService {
	return &commandRunService{ctx: ctx, repo: repo, runner: runner}
}

func (s *commandRunService) project(projectID string) (db.Project, error) {
	p, err := s.repo.Queries.GetProject(s.ctx, projectID)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrProjectNotFound
	}
	return p, err
}

// Start checks the name against the project's commands as the hub last saw
// them, for a clear error. The agent checks again against the file itself.
func (s *commandRunService) Start(projectID, command string, userID int64) (dto.CommandRunDto, error) {
	p, err := s.project(projectID)
	if err != nil {
		return dto.CommandRunDto{}, err
	}
	commandLine, found := "", false
	for _, c := range dto.ConvertToProjectResponse(&p).Commands {
		if c.Name == command {
			commandLine, found = c.Command, true
		}
	}
	if !found {
		return dto.CommandRunDto{}, ErrUnknownCommand
	}
	user, err := s.repo.Queries.FindUserById(s.ctx, userID)
	if err != nil {
		return dto.CommandRunDto{}, err
	}
	run, err := s.runner.Start(s.ctx, tcpserver.StartRun{
		NodeID: p.NodeID, ProjectPath: p.Path, ProjectName: p.Name,
		Command: command, CommandLine: commandLine, UserID: user.ID, Username: user.Username,
	})
	if err != nil {
		return dto.CommandRunDto{}, err
	}
	return dto.NewCommandRunDto(run), nil
}

func (s *commandRunService) Cancel(runID string) error {
	if _, err := s.run(runID); err != nil {
		return err
	}
	return s.runner.Cancel(runID)
}

func (s *commandRunService) run(runID string) (db.CommandRun, error) {
	run, err := s.repo.Queries.GetCommandRun(s.ctx, runID)
	if errors.Is(err, sql.ErrNoRows) {
		return run, ErrRunNotFound
	}
	return run, err
}

func (s *commandRunService) Get(runID string) (dto.CommandRunDto, []dto.CommandOutputDto, error) {
	run, err := s.run(runID)
	if err != nil {
		return dto.CommandRunDto{}, nil, err
	}
	out, err := s.OutputAfter(runID, -1)
	return dto.NewCommandRunDto(run), out, err
}

func (s *commandRunService) OutputAfter(runID string, after int64) ([]dto.CommandOutputDto, error) {
	rows, err := s.repo.Queries.ListCommandRunOutput(s.ctx, runID)
	if err != nil {
		return nil, err
	}
	out := dto.NewCommandOutputDtos(rows)
	for len(out) > 0 && out[0].Seq <= after {
		out = out[1:]
	}
	return out, nil
}

func (s *commandRunService) ListByProject(projectID string, limit, offset int64) ([]dto.CommandRunDto, error) {
	p, err := s.project(projectID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.Queries.ListCommandRunsByProject(s.ctx, db.ListCommandRunsByProjectParams{
		NodeID: p.NodeID, ProjectPath: p.Path, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]dto.CommandRunDto, len(rows))
	for i, r := range rows {
		out[i] = dto.NewCommandRunDto(r)
	}
	return out, nil
}

func (s *commandRunService) Watch(runID string) (<-chan tcpserver.RunEvent, func()) {
	return s.runner.Watch(runID)
}
