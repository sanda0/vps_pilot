package services

import (
	"context"
	"database/sql"
	"errors"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/dto"
)

// CronService lists a project's cron jobs. The agent runs
// the jobs; the hub only shows what it reported.
type CronService interface {
	ListJobs(projectID string) ([]dto.CronJobDto, error)
}

type cronService struct {
	ctx  context.Context
	repo *db.Repo
}

func NewCronService(ctx context.Context, repo *db.Repo) CronService {
	return &cronService{ctx: ctx, repo: repo}
}

func (s *cronService) project(projectID string) (db.Project, error) {
	p, err := s.repo.Queries.GetProject(s.ctx, projectID)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrProjectNotFound
	}
	return p, err
}

func (s *cronService) runs(p db.Project, command string, limit, offset int64) ([]db.CronRun, error) {
	return s.repo.Queries.ListCronRunsByJob(s.ctx, db.ListCronRunsByJobParams{
		NodeID: p.NodeID, ProjectPath: p.Path, Command: command, Limit: limit, Offset: offset,
	})
}

// ListJobs returns the jobs in the project's config, as the hub last saw
// it, each with its latest run.
func (s *cronService) ListJobs(projectID string) ([]dto.CronJobDto, error) {
	p, err := s.project(projectID)
	if err != nil {
		return nil, err
	}
	jobs := dto.ParseCron(p.Cron)
	out := make([]dto.CronJobDto, len(jobs))
	for i, j := range jobs {
		out[i] = dto.CronJobDto{Command: j.Command, Schedule: j.Schedule}
		last, err := s.runs(p, j.Command, 1, 0)
		if err != nil {
			return nil, err
		}
		if len(last) == 1 {
			run := dto.NewCronRunDto(last[0])
			out[i].LastRun = &run
		}
	}
	return out, nil
}
