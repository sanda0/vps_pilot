package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/internal/dto"
	"github.com/sanda0/vps_pilot/internal/services"
	"github.com/sanda0/vps_pilot/internal/tcpserver"
)

// CommandRunHandler serves /projects/:id/runs and /runs/:runId.
type CommandRunHandler struct {
	service services.CommandRunService
}

func NewCommandRunHandler(service services.CommandRunService) *CommandRunHandler {
	return &CommandRunHandler{service: service}
}

// Start handles POST /projects/:id/runs with {"command": "<name>"}.
func (h *CommandRunHandler) Start(c *gin.Context) {
	var req dto.StartCommandRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "command is required"})
		return
	}
	userID, _ := c.Get("user_id")
	run, err := h.service.Start(c.Param("id"), req.Command, int64(userID.(int32)))
	if err != nil {
		runError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": run})
}

// List handles GET /projects/:id/runs?limit=&offset=
func (h *CommandRunHandler) List(c *gin.Context) {
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
		runError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": runs, "limit": limit, "offset": offset})
}

// Get handles GET /runs/:runId: the run and all its stored output.
func (h *CommandRunHandler) Get(c *gin.Context) {
	run, output, err := h.service.Get(c.Param("runId"))
	if err != nil {
		runError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"run": run, "output": output}})
}

// Cancel handles POST /runs/:runId/cancel.
func (h *CommandRunHandler) Cancel(c *gin.Context) {
	if err := h.service.Cancel(c.Param("runId")); err != nil {
		runError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": "cancel requested"})
}

func runError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrProjectNotFound), errors.Is(err, services.ErrRunNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrUnknownCommand):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, tcpserver.ErrNodeOffline), errors.Is(err, tcpserver.ErrNoCommands),
		errors.Is(err, tcpserver.ErrRunNotRunning):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
