package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/internal/dto"
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

}

// CreateAlert implements AlertHandler.
func (a *alertHandler) CreateAlert(c *gin.Context) {
	form := dto.AlertDto{}
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	alert, err := a.alertService.CreateAlert(form)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": alert})
}

// DeactivateAlert implements AlertHandler.
func (a *alertHandler) DeactivateAlert(c *gin.Context) {
	panic("unimplemented")
}

// DeleteAlert implements AlertHandler.
func (a *alertHandler) DeleteAlert(c *gin.Context) {

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	err = a.alertService.DeleteAlert(int32(id))
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": "Alert deleted"})
}

// GetAlert implements AlertHandler.
func (a *alertHandler) GetAlert(c *gin.Context) {

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	alert, err := a.alertService.GetAlert(int32(id))
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": alert})

}

// GetAlerts implements AlertHandler.
func (a *alertHandler) GetAlerts(c *gin.Context) {

	nodeIdStr := c.Query("node_id")
	nodeId, err := strconv.Atoi(nodeIdStr)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	limitStr := c.Query("limit")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	offsetStr := c.Query("offset")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	alerts, err := a.alertService.GetAlerts(int32(nodeId), int32(limit), int32(offset))
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"data": alerts})
}

// UpdateAlert implements AlertHandler.
func (a *alertHandler) UpdateAlert(c *gin.Context) {
	form := dto.AlertUpdateDto{}
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	alert, err := a.alertService.UpdateAlert(form)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": alert})
}

func NewAlertHandler(alertService services.AlertService) AlertHandler {
	return &alertHandler{
		alertService: alertService,
	}
}
