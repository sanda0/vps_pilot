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

// LogListTimeout bounds how long List waits for log_files. The agent gives
// up on a listing after 10 seconds.
var LogListTimeout = 15 * time.Second

// Logs relays log requests to agents and their answers back. Nothing is
// stored. See "Read logs" in docs/PROTOCOL.md.
type Logs struct {
	registry *Registry

	mu sync.Mutex
	// lists maps a request to the connection it was sent on. Answers are
	// accepted only from that connection.
	lists map[string]*pendingList
}

type pendingList struct {
	conn  *ConnContext
	reply chan LogFiles
}

// NewLogs ends a node's lists when its connection closes.
func NewLogs(registry *Registry) *Logs {
	l := &Logs{
		registry: registry,
		lists:    make(map[string]*pendingList),
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

// connClosed ends the lists sent on c.
func (l *Logs) connClosed(c *ConnContext) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for id, p := range l.lists {
		if p.conn == c {
			delete(l.lists, id)
			close(p.reply)
		}
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
