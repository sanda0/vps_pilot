package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/internal/dto"
	"github.com/sanda0/vps_pilot/internal/services"
)

// ProjectScanHandler serves a node's project search settings:
// /nodes/:id/project-settings and /nodes/:id/rescan-projects.
type ProjectScanHandler struct {
	service services.ProjectScanService
}

func NewProjectScanHandler(service services.ProjectScanService) *ProjectScanHandler {
	return &ProjectScanHandler{service: service}
}

// Get handles GET /nodes/:id/project-settings
func (h *ProjectScanHandler) Get(c *gin.Context) {
	nodeID, ok := nodeIDParam(c)
	if !ok {
		return
	}
	scan, err := h.service.Get(nodeID)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": scan})
}

// UpdateSettings handles PUT /nodes/:id/project-settings
func (h *ProjectScanHandler) UpdateSettings(c *gin.Context) {
	nodeID, ok := nodeIDParam(c)
	if !ok {
		return
	}
	var req dto.ProjectScanSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auto_scan is required and scan_interval must be 60 to 86400 seconds"})
		return
	}
	scan, err := h.service.UpdateSettings(nodeID, req)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": scan})
}

// Rescan handles POST /nodes/:id/rescan-projects
func (h *ProjectScanHandler) Rescan(c *gin.Context) {
	nodeID, ok := nodeIDParam(c)
	if !ok {
		return
	}
	sent, err := h.service.Rescan(nodeID)
	if err != nil {
		serviceError(c, err)
		return
	}
	if !sent {
		c.JSON(http.StatusConflict, gin.H{"error": "node is offline; it searches for projects when it reconnects"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": "rescan requested"})
}

func nodeIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node ID"})
		return 0, false
	}
	return id, true
}

func serviceError(c *gin.Context, err error) {
	if errors.Is(err, services.ErrNodeNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
