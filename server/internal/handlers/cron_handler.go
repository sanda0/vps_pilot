package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/internal/services"
)

// cronRunsMax is the most runs one page holds: all the hub keeps of a job.
const cronRunsMax = 50

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

// Runs handles GET /projects/:id/cron/runs?command=&limit=&offset=, newest
// first.
func (h *CronHandler) Runs(c *gin.Context) {
	limit, err := strconv.ParseInt(c.DefaultQuery("limit", "20"), 10, 64)
	if err != nil || limit <= 0 || limit > cronRunsMax {
		limit = 20
	}
	offset, err := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	if err != nil || offset < 0 {
		offset = 0
	}
	runs, err := h.service.ListRuns(c.Param("id"), c.Query("command"), limit, offset)
	if err != nil {
		cronError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": runs, "limit": limit, "offset": offset})
}

func cronError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrProjectNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrCronCommandRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
