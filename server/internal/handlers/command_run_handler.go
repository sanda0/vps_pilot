package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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

var runUpgrader = websocket.Upgrader{CheckOrigin: sameOrigin}

// sameOrigin accepts browsers on the dashboard's own host, so another site
// can't read a run's output through the user's cookie.
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && u.Host == r.Host
}

// Watch handles GET /runs/:runId/ws. It sends {"run": …}, then every output
// chunk as {"output": …}, stored first and then live, and {"run": …} again
// with the final state, then closes. If the socket closes while the last
// run sent is still running, the dashboard should reconnect.
func (h *CommandRunHandler) Watch(c *gin.Context) {
	runID := c.Param("runId")
	events, stop := h.service.Watch(runID)
	defer stop()
	run, output, err := h.service.Get(runID)
	if err != nil {
		runError(c, err)
		return
	}
	conn, err := runUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	defer func() {
		msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(time.Second))
	}()
	gone := make(chan struct{})
	go func() {
		defer close(gone)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	send := func(v any) bool {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		return conn.WriteJSON(v) == nil
	}
	last := int64(-1)
	sendOutput := func(out []dto.CommandOutputDto) bool {
		for _, o := range out {
			if !send(gin.H{"output": o}) {
				return false
			}
			last = o.Seq
		}
		return true
	}
	if !send(gin.H{"run": run}) || !sendOutput(output) || run.Status != "running" {
		return
	}
	for {
		select {
		case <-gone:
			return
		case ev, ok := <-events:
			if ok && ev.Output != nil {
				if ev.Output.Seq > last && !sendOutput([]dto.CommandOutputDto{{Seq: ev.Output.Seq, Stream: ev.Output.Stream, Data: ev.Output.Data}}) {
					return
				}
				continue
			}
			// The run finished, or this watcher fell behind: send what is
			// stored and the run as it is now.
			rest, err := h.service.OutputAfter(runID, last)
			if err != nil || !sendOutput(rest) {
				return
			}
			if run, err := h.service.GetRun(runID); err == nil {
				send(gin.H{"run": run})
			}
			return
		}
	}
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
