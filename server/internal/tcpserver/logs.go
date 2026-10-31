package tcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNoLogs   = errors.New("this node's agent can't read logs; upgrade it to v0.4 or later")
	ErrNoAnswer = errors.New("the node didn't answer in time")
	ErrNodeLeft = errors.New("the node disconnected")
)

const errFellBehind = "the dashboard fell behind; open the file again"

// LogListTimeout bounds how long List waits for log_files. The agent gives
// up on a listing after 10 seconds.
var LogListTimeout = 15 * time.Second

// streamBuffer is how many chunks a stream's reader may fall behind before
// the stream is closed. Chunks are up to 32 KiB.
const streamBuffer = 64

// LogEvent is what a stream's reader receives: chunks, then one end.
type LogEvent struct {
	Data *LogData `json:"data,omitempty"`
	End  *LogEnd  `json:"end,omitempty"`
}

// OpenLog is a stream the dashboard asked for. See LogOpen.
type OpenLog struct {
	NodeID      int64
	ProjectPath string
	File        string
	Before      *int64
	MaxBytes    int64
	Follow      bool
}

// Logs relays log requests to agents and their answers back. Nothing is
// stored. See "Read logs" in docs/PROTOCOL.md.
type Logs struct {
	registry *Registry

	mu sync.Mutex
	// lists and streams map a request or stream to the connection it was
	// sent on. Answers are accepted only from that connection.
	lists   map[string]*pendingList
	streams map[string]*logStream
}

type pendingList struct {
	conn  *ConnContext
	reply chan LogFiles
}

type logStream struct {
	conn   *ConnContext
	events chan LogEvent
}

// NewLogs ends a node's lists and streams when its connection closes.
func NewLogs(registry *Registry) *Logs {
	l := &Logs{
		registry: registry,
		lists:    make(map[string]*pendingList),
		streams:  make(map[string]*logStream),
	}
	registry.OnDisconnect(l.connClosed)
	return l
}

// conn returns the node's connection if its agent can read logs.
func (l *Logs) conn(nodeID int64) (*ConnContext, error) {
	c, ok := l.registry.Get(nodeID)
	if !ok {
		return nil, ErrNodeOffline
	}
	if !c.Supports(FeatureLogs) {
		return nil, ErrNoLogs
	}
	return c, nil
}

// List asks the node's agent for the files in a project's log folders. A
// refusal from the agent comes back in LogFiles.Error, not as an error.
func (l *Logs) List(ctx context.Context, nodeID int64, projectPath string) (LogFiles, error) {
	c, err := l.conn(nodeID)
	if err != nil {
		return LogFiles{}, err
	}
	id := uuid.NewString()
	p := &pendingList{conn: c, reply: make(chan LogFiles, 1)}
	l.mu.Lock()
	l.lists[id] = p
	l.mu.Unlock()
	defer func() {
		l.mu.Lock()
		delete(l.lists, id)
		l.mu.Unlock()
	}()

	data, _ := json.Marshal(LogList{RequestID: id, ProjectPath: projectPath})
	if err := c.Send(Msg{Msg: MsgLogList, Data: data}); err != nil {
		return LogFiles{}, ErrNodeOffline
	}
	select {
	case files, ok := <-p.reply:
		if !ok {
			return LogFiles{}, ErrNodeLeft
		}
		return files, nil
	case <-ctx.Done():
		return LogFiles{}, ctx.Err()
	case <-time.After(LogListTimeout):
		return LogFiles{}, ErrNoAnswer
	}
}

// Open sends log_open and returns the stream's events and a function that
// closes it. The channel is closed after the one end event. Call the
// function when done reading, even after the end.
func (l *Logs) Open(req OpenLog) (<-chan LogEvent, func(), error) {
	c, err := l.conn(req.NodeID)
	if err != nil {
		return nil, nil, err
	}
	id := uuid.NewString()
	// One more than streamBuffer, so the end always fits.
	s := &logStream{conn: c, events: make(chan LogEvent, streamBuffer+1)}
	l.mu.Lock()
	l.streams[id] = s
	l.mu.Unlock()

	data, _ := json.Marshal(LogOpen{
		StreamID: id, ProjectPath: req.ProjectPath, File: req.File,
		Before: req.Before, MaxBytes: req.MaxBytes, Follow: req.Follow,
	})
	if err := c.Send(Msg{Msg: MsgLogOpen, Data: data}); err != nil {
		l.end(c, LogEnd{StreamID: id, Reason: "error", Error: "could not reach the node"})
	}
	return s.events, func() { l.close(c, LogEnd{StreamID: id, Reason: "closed"}) }, nil
}

// close ends a stream the hub gives up on and tells the agent, unless the
// stream already ended.
func (l *Logs) close(c *ConnContext, end LogEnd) {
	if !l.end(c, end) {
		return
	}
	data, _ := json.Marshal(LogClose{StreamID: end.StreamID})
	c.Send(Msg{Msg: MsgLogClose, Data: data})
}

// end sends the end event of a stream sent on c and forgets the stream. It
// reports whether the stream was still open.
func (l *Logs) end(c *ConnContext, end LogEnd) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	s, ok := l.streams[end.StreamID]
	if !ok || s.conn != c {
		return false
	}
	delete(l.streams, end.StreamID)
	s.events <- LogEvent{End: &end}
	close(s.events)
	return true
}

// connClosed ends the lists and streams sent on c: the agent closes its
// streams when the session ends.
func (l *Logs) connClosed(c *ConnContext) {
	l.mu.Lock()
	var ended []string
	for id, s := range l.streams {
		if s.conn == c {
			ended = append(ended, id)
		}
	}
	for id, p := range l.lists {
		if p.conn == c {
			delete(l.lists, id)
			close(p.reply)
		}
	}
	l.mu.Unlock()
	for _, id := range ended {
		l.end(c, LogEnd{StreamID: id, Reason: "error", Error: ErrNodeLeft.Error()})
	}
}

// HandleFiles passes log_files to the List waiting for it.
func (l *Logs) HandleFiles(ctx context.Context, c *ConnContext, msg Msg) {
	var files LogFiles
	if err := json.Unmarshal(msg.Data, &files); err != nil {
		fmt.Printf("HandleFiles: bad log_files from node %d: %v\n", c.NodeID, err)
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	p, ok := l.lists[files.RequestID]
	if !ok || p.conn != c {
		return
	}
	delete(l.lists, files.RequestID)
	p.reply <- files
}

// HandleData passes a log_data chunk to the stream's reader. A reader that
// fell behind gets its stream closed instead: waiting for it would hold up
// every message from the node.
func (l *Logs) HandleData(ctx context.Context, c *ConnContext, msg Msg) {
	var data LogData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		fmt.Printf("HandleData: bad log_data from node %d: %v\n", c.NodeID, err)
		return
	}
	l.mu.Lock()
	s, ok := l.streams[data.StreamID]
	if !ok || s.conn != c {
		l.mu.Unlock()
		return
	}
	if len(s.events) < streamBuffer {
		s.events <- LogEvent{Data: &data}
		l.mu.Unlock()
		return
	}
	l.mu.Unlock()
	l.close(c, LogEnd{StreamID: data.StreamID, Reason: "error", Error: errFellBehind})
}

// agentEndReasons are the log_end reasons an agent may send.
var agentEndReasons = map[string]bool{"done": true, "closed": true, "rejected": true, "error": true, "limit": true}

// HandleEnd ends a stream on log_end.
func (l *Logs) HandleEnd(ctx context.Context, c *ConnContext, msg Msg) {
	var end LogEnd
	if err := json.Unmarshal(msg.Data, &end); err != nil {
		fmt.Printf("HandleEnd: bad log_end from node %d: %v\n", c.NodeID, err)
		return
	}
	if !agentEndReasons[end.Reason] {
		end.Error = fmt.Sprintf("agent sent unknown reason %q", end.Reason)
		end.Reason = "error"
	}
	l.end(c, end)
}
