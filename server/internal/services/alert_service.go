package services

import (
	"context"
	"database/sql"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/dto"
)

type AlertService interface {
	CreateAlert(dto dto.AlertDto) (*db.Alert, error)
	GetAlerts(nodeId int32, limit int32, offset int32) ([]db.Alert, error)
	UpdateAlert(dto dto.AlertUpdateDto) (*db.Alert, error)
	GetAlert(alertId int32) (*db.Alert, error)
	DeleteAlert(alertId int32) error
	ActivateAlert(alertId int32) error
	DeactivateAlert(alertId int32) error
}

type alertService struct {
	repo *db.Repo
	ctx  context.Context
}

// ActivateAlert implements AlertService.
func (a *alertService) ActivateAlert(alertId int32) error {
	err := a.repo.Queries.ActivateAlert(a.ctx, int64(alertId))
	if err != nil {
		return err
	}
	return nil
}

// CreateAlert implements AlertService.
func (a *alertService) CreateAlert(dto dto.AlertDto) (*db.Alert, error) {
	// metric := "cpu"
	// if dto.Metric == "mem" {
	// 	metric = "mem"
	// } else if dto.Metric == "net" {
	// 	metric = "net"
	// }

	alert, err := a.repo.Queries.CreateAlert(a.ctx, db.CreateAlertParams{
		NodeID:   int64(dto.NodeID),
		Metric:   dto.Metric,
		Duration: int64(dto.Duration),
		Threshold: sql.NullFloat64{
			Float64: dto.Threshold,
			Valid:   true,
		},
		NetReceThreshold: sql.NullFloat64{
			Float64: dto.NetReceThreshold,
			Valid:   true,
		},
		NetSendThreshold: sql.NullFloat64{
			Float64: dto.NetSendThreshold,
			Valid:   true,
		},
		Email: sql.NullString{String: dto.Email, Valid: true},
		IsActive: sql.NullInt64{
			Int64: boolToInt64(dto.Enabled),
			Valid: true,
		},
		SlackWebhook:   sql.NullString{String: dto.Slack, Valid: true},
		DiscordWebhook: sql.NullString{String: dto.Discord, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

// DeactivateAlert implements AlertService.
func (a *alertService) DeactivateAlert(alertId int32) error {
	err := a.repo.Queries.DeactivateAlert(a.ctx, int64(alertId))
	if err != nil {
		return err
	}
	return nil
}

// DeleteAlert implements AlertService.
func (a *alertService) DeleteAlert(alertId int32) error {
	err := a.repo.Queries.DeleteAlert(a.ctx, int64(alertId))
	if err != nil {
		return err
	}
	return nil
}

// GetAlert implements AlertService.
func (a *alertService) GetAlert(alertId int32) (*db.Alert, error) {
	alert, err := a.repo.Queries.GetAlert(a.ctx, int64(alertId))
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

// GetAlerts implements AlertService.
func (a *alertService) GetAlerts(nodeId int32, limit int32, offset int32) ([]db.Alert, error) {
	alerts, err := a.repo.Queries.GetAlerts(a.ctx, db.GetAlertsParams{
		NodeID: int64(nodeId),
		Limit:  int64(limit),
		Offset: int64(offset),
	})

	if err != nil {
		return nil, err
	}
	return alerts, nil
}

// UpdateAlert implements AlertService.
func (a *alertService) UpdateAlert(dto dto.AlertUpdateDto) (*db.Alert, error) {

	alert, err := a.repo.Queries.UpdateAlert(a.ctx, db.UpdateAlertParams{
		ID:       int64(dto.ID),
		NodeID:   int64(dto.NodeID),
		Metric:   dto.Metric,
		Duration: int64(dto.Duration),
		Threshold: sql.NullFloat64{
			Float64: dto.Threshold,
			Valid:   true,
		},
		NetReceThreshold: sql.NullFloat64{
			Float64: dto.NetReceThreshold,
			Valid:   true,
		},
		NetSendThreshold: sql.NullFloat64{
			Float64: dto.NetSendThreshold,
			Valid:   true,
		},
		Email: sql.NullString{String: dto.Email, Valid: true},
		IsActive: sql.NullInt64{
			Int64: boolToInt64(dto.Enabled),
			Valid: true,
		},
		SlackWebhook:   sql.NullString{String: dto.Slack, Valid: true},
		DiscordWebhook: sql.NullString{String: dto.Discord, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

func NewAlertService(ctx context.Context, repo *db.Repo) AlertService {
	return &alertService{
		repo: repo,
		ctx:  ctx,
	}
}

// Helper function to convert bool to int64 for SQLite
func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
