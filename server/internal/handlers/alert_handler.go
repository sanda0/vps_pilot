package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/internal/services"
)

type AlertHandler interface {
	CreateAlert(c *gin.Context)
	UpdateAlert(c *gin.Context)
	GetAlerts(c *gin.Context)
	GetAlert(c *gin.Context)
	DeleteAlert(c *gin.Context)
	ActivateAlert(c *gin.Context)
	DeactivateAlert(c *gin.Context)
}

type alertHandler struct {
	alertService services.AlertService
}

// ActivateAlert implements AlertHandler.
func (a *alertHandler) ActivateAlert(c *gin.Context) {
	panic("unimplemented")
}

// CreateAlert implements AlertHandler.
func (a *alertHandler) CreateAlert(c *gin.Context) {
	panic("unimplemented")
}

// DeactivateAlert implements AlertHandler.
func (a *alertHandler) DeactivateAlert(c *gin.Context) {
	panic("unimplemented")
}

// DeleteAlert implements AlertHandler.
func (a *alertHandler) DeleteAlert(c *gin.Context) {
	panic("unimplemented")
}

// GetAlert implements AlertHandler.
func (a *alertHandler) GetAlert(c *gin.Context) {
	panic("unimplemented")
}

// GetAlerts implements AlertHandler.
func (a *alertHandler) GetAlerts(c *gin.Context) {
	panic("unimplemented")
}

// UpdateAlert implements AlertHandler.
func (a *alertHandler) UpdateAlert(c *gin.Context) {
	panic("unimplemented")
}

func NewAlertHandler(alertService services.AlertService) AlertHandler {
	return &alertHandler{
		alertService: alertService,
	}
}
