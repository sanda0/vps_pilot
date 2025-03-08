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
	err := a.repo.Queries.ActivateAlert(a.ctx, alertId)
	if err != nil {
		return err
	}
	return nil
}

// CreateAlert implements AlertService.
func (a *alertService) CreateAlert(dto dto.AlertDto) (*db.Alert, error) {
	matric := "cpu"
	if dto.Matric == "Memory" {
		matric = "mem"
	} else if dto.Matric == "Network" {
		matric = "net"
	}

	alert, err := a.repo.Queries.CreateAlert(a.ctx, db.CreateAlertParams{
		NodeID:   dto.NodeID,
		Metric:   matric,
		Duration: dto.Duration,
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
		IsActive: sql.NullBool{
			Bool:  dto.Enabled,
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
	err := a.repo.Queries.DeactivateAlert(a.ctx, alertId)
	if err != nil {
		return err
	}
	return nil
}

// DeleteAlert implements AlertService.
func (a *alertService) DeleteAlert(alertId int32) error {
	err := a.repo.Queries.DeleteAlert(a.ctx, alertId)
	if err != nil {
		return err
	}
	return nil
}

// GetAlert implements AlertService.
func (a *alertService) GetAlert(alertId int32) (*db.Alert, error) {
	alert, err := a.repo.Queries.GetAlert(a.ctx, alertId)
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

// GetAlerts implements AlertService.
func (a *alertService) GetAlerts(nodeId int32, limit int32, offset int32) ([]db.Alert, error) {
	alerts, err := a.repo.Queries.GetAlerts(a.ctx, db.GetAlertsParams{
		NodeID: nodeId,
		Limit:  limit,
		Offset: offset,
	})

	if err != nil {
		return nil, err
	}
	return alerts, nil
}

// UpdateAlert implements AlertService.
func (a *alertService) UpdateAlert(dto dto.AlertUpdateDto) (*db.Alert, error) {
	matric := "cpu"
	if dto.Matric == "Memory" {
		matric = "mem"
	} else if dto.Matric == "Network" {
		matric = "net"
	}
	alert, err := a.repo.Queries.UpdateAlert(a.ctx, db.UpdateAlertParams{
		ID:       dto.ID,
		NodeID:   dto.NodeID,
		Metric:   matric,
		Duration: dto.Duration,
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
		IsActive: sql.NullBool{
			Bool:  dto.Enabled,
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
