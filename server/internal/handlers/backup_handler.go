package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/internal/services"
	"github.com/sanda0/vps_pilot/internal/tcpserver"
)

// BackupHandler serves /projects/:id/backups.
type BackupHandler struct {
	service services.BackupService
}

func NewBackupHandler(service services.BackupService) *BackupHandler {
	return &BackupHandler{service: service}
}

// Start handles POST /projects/:id/backups: Back up now. The run is
// "running" until the agent reports; poll List for the outcome.
func (h *BackupHandler) Start(c *gin.Context) {
	userID, _ := c.Get("user_id")
	run, err := h.service.Start(c.Param("id"), int64(userID.(int32)))
	if err != nil {
		backupError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": run})
}

// List handles GET /projects/:id/backups?limit=&offset=, newest first.
func (h *BackupHandler) List(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "20"), 10, 64)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil || offset < 0 {
		offset = 0
	}
	runs, err := h.service.ListByProject(c.Param("id"), limit, offset)
	if err != nil {
		backupError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": runs, "limit": limit, "offset": offset})
}

func backupError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrProjectNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrNoBackupConfig), errors.Is(err, tcpserver.ErrNodeOffline),
		errors.Is(err, tcpserver.ErrNoBackups):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
