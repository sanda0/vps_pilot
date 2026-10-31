package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sanda0/vps_pilot/internal/dto"
	"github.com/sanda0/vps_pilot/internal/services"
	"github.com/sanda0/vps_pilot/internal/tcpserver"
)

// defaultLogWindow is how much of a file is read when max_bytes is not
// given. The agent clamps it to 1 MiB.
const defaultLogWindow = 64 * 1024

// LogHandler serves /projects/:id/logs.
type LogHandler struct {
	service services.LogService
}

func NewLogHandler(service services.LogService) *LogHandler {
	return &LogHandler{service: service}
}

// List handles GET /projects/:id/logs.
func (h *LogHandler) List(c *gin.Context) {
	files, err := h.service.List(c.Request.Context(), c.Param("id"))
	if err != nil {
		logError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": files})
}

// openLogRequest reads ?file=&before=&max_bytes=&follow= from the query.
func openLogRequest(c *gin.Context) (dto.OpenLogRequest, error) {
	req := dto.OpenLogRequest{File: c.Query("file"), MaxBytes: defaultLogWindow, Follow: c.Query("follow") == "true"}
	if req.File == "" {
		return req, errors.New("file is required")
	}
	if v := c.Query("before"); v != "" {
		before, err := strconv.ParseInt(v, 10, 64)
		if err != nil || before < 0 {
			return req, errors.New("before must be a byte offset")
		}
		req.Before = &before
	}
	if v := c.Query("max_bytes"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			return req, errors.New("max_bytes must be a positive number")
		}
		req.MaxBytes = n
	}
	if req.Follow && req.Before != nil {
		return req, errors.New("follow only works from the end of the file")
	}
	return req, nil
}

// Watch handles GET /projects/:id/logs/ws?file=&before=&max_bytes=&follow=true.
// It sends every chunk as {"data": …} and then one {"end": …}, then closes.
// Closing the socket closes the stream on the agent.
func (h *LogHandler) Watch(c *gin.Context) {
	req, err := openLogRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	events, closeStream, err := h.service.Open(c.Param("id"), req)
	if err != nil {
		logError(c, err)
		return
	}
	defer closeStream()
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

	for {
		select {
		case <-gone:
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if conn.WriteJSON(ev) != nil || ev.End != nil {
				return
			}
		}
	}
}

func logError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrProjectNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrNoLogFolders), errors.Is(err, tcpserver.ErrNodeOffline),
		errors.Is(err, tcpserver.ErrNoLogs), errors.Is(err, tcpserver.ErrNodeLeft):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, tcpserver.ErrNoAnswer):
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
