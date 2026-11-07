package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/internal/services"
)

// CronHandler serves /projects/:id/cron. There is nothing to start: the
// agent runs cron jobs, and Run runs a job's command now.
type CronHandler struct {
	service services.CronService
}

func NewCronHandler(service services.CronService) *CronHandler {
	return &CronHandler{service: service}
}

// Jobs handles GET /projects/:id/cron: each job with its latest run.
func (h *CronHandler) Jobs(c *gin.Context) {
	jobs, err := h.service.ListJobs(c.Param("id"))
	if err != nil {
		cronError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": jobs})
}

func cronError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrProjectNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
