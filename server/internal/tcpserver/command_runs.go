package tcpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/sanda0/vps_pilot/internal/db"
)

var (
	ErrNodeOffline   = errors.New("node is offline")
	ErrNoCommands    = errors.New("this node's agent can't run commands; upgrade it to v0.3 or later")
	ErrRunNotRunning = errors.New("run is not running")
)

// watcherBuffer is how many events a watcher may fall behind before it is
// dropped. A dropped watcher reloads the run's stored output.
const watcherBuffer = 256

// RunEvent is what a run's watchers receive: output chunks, then the exit.
type RunEvent struct {
	Output *CommandOutput `json:"output,omitempty"`
	Exit   *CommandExit   `json:"exit,omitempty"`
}

// StartRun is a run the dashboard asked for.
type StartRun struct {
	NodeID      int64
	ProjectPath string
	ProjectName string
	Command     string // name
	CommandLine string // what the hub last saw in the config, for the record
	UserID      int64
	Username    string
}

// Runs starts project commands on agents, records them in command_runs and
// passes their output to watchers. See "Run commands" in docs/PROTOCOL.md.
type Runs struct {
	repo     *db.Repo
	registry *Registry

	mu sync.Mutex
	// active maps a running run to the connection it was sent on. Output and
	// exit are accepted only from that connection.
	active   map[string]*ConnContext
	watchers map[string]map[chan RunEvent]struct{}
}

// NewRuns marks runs left running by a previous hub process as lost, and
// marks a node's runs lost when its connection closes.
func NewRuns(ctx context.Context, repo *db.Repo, registry *Registry) *Runs {
	if n, err := repo.Queries.MarkAllRunningCommandRunsLost(ctx); err != nil {
		fmt.Printf("mark command runs lost: %v\n", err)
	} else if n > 0 {
		fmt.Printf("%d command run(s) were still running when the hub stopped; marked lost\n", n)
	}
	r := &Runs{
		repo:     repo,
		registry: registry,
		active:   make(map[string]*ConnContext),
		watchers: make(map[string]map[chan RunEvent]struct{}),
	}
	registry.OnDisconnect(func(c *ConnContext) { r.connClosed(ctx, c) })
	return r
}

// Start records a run and sends command_run to the node's agent.
func (r *Runs) Start(ctx context.Context, req StartRun) (db.CommandRun, error) {
	c, ok := r.registry.Get(req.NodeID)
	if !ok {
		return db.CommandRun{}, ErrNodeOffline
	}
	if !c.Supports(FeatureCommands) {
		return db.CommandRun{}, ErrNoCommands
	}
	run, err := r.repo.Queries.CreateCommandRun(ctx, db.CreateCommandRunParams{
		ID:          uuid.NewString(),
		NodeID:      req.NodeID,
		ProjectPath: req.ProjectPath,
		ProjectName: req.ProjectName,
		Command:     req.Command,
		CommandLine: req.CommandLine,
		UserID:      sql.NullInt64{Int64: req.UserID, Valid: req.UserID != 0},
		Username:    req.Username,
	})
	if err != nil {
		return db.CommandRun{}, err
	}

	r.mu.Lock()
	r.active[run.ID] = c
	r.mu.Unlock()
	data, _ := json.Marshal(CommandRun{RunID: run.ID, ProjectPath: req.ProjectPath, Command: req.Command})
	if err := c.Send(Msg{Msg: MsgCommandRun, Data: data}); err != nil {
		r.finish(ctx, c, CommandExit{RunID: run.ID, Status: "failed", Error: "could not reach the node"})
	}
	return r.repo.Queries.GetCommandRun(ctx, run.ID)
}

// Cancel asks the agent to stop a running run.
func (r *Runs) Cancel(runID string) error {
	r.mu.Lock()
	c, ok := r.active[runID]
	r.mu.Unlock()
	if !ok {
		return ErrRunNotRunning
	}
	data, _ := json.Marshal(CommandCancel{RunID: runID})
	return c.Send(Msg{Msg: MsgCommandCancel, Data: data})
}

// Watch returns the run's events from now on, and a function to stop
// watching. The channel is closed after the exit, or at once if the run
// isn't running, or if the watcher falls behind. Watch before reading the
// stored output, then skip events whose seq was already read.
func (r *Runs) Watch(runID string) (<-chan RunEvent, func()) {
	ch := make(chan RunEvent, watcherBuffer)
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.active[runID]; !ok {
		close(ch)
		return ch, func() {}
	}
	if r.watchers[runID] == nil {
		r.watchers[runID] = make(map[chan RunEvent]struct{})
	}
	r.watchers[runID][ch] = struct{}{}
	return ch, func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if _, ok := r.watchers[runID][ch]; ok {
			delete(r.watchers[runID], ch)
			close(ch)
		}
	}
}

// publish sends ev to the run's watchers. Callers hold r.mu.
func (r *Runs) publish(runID string, ev RunEvent) {
	for ch := range r.watchers[runID] {
		select {
		case ch <- ev:
		default:
			delete(r.watchers[runID], ch)
			close(ch)
		}
	}
}

// finish stores the exit of a run sent on c, then tells its watchers and
// forgets it. The row is updated first, so a Watch that finds the run gone
// reads its final state.
func (r *Runs) finish(ctx context.Context, c *ConnContext, exit CommandExit) {
	r.mu.Lock()
	owner, ok := r.active[exit.RunID]
	r.mu.Unlock()
	if !ok || owner != c {
		return
	}
	truncated := int64(0)
	if exit.Truncated {
		truncated = 1
	}
	var code sql.NullInt64
	if exit.ExitCode != nil {
		code = sql.NullInt64{Int64: *exit.ExitCode, Valid: true}
	}
	if _, err := r.repo.Queries.FinishCommandRun(ctx, db.FinishCommandRunParams{
		Status: exit.Status, ExitCode: code, RunAs: exit.RunAs, Error: exit.Error,
		Truncated: truncated, DurationMs: exit.DurationMs, ID: exit.RunID,
	}); err != nil {
		fmt.Printf("finish command run %s: %v\n", exit.RunID, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.active, exit.RunID)
	r.publish(exit.RunID, RunEvent{Exit: &exit})
	for ch := range r.watchers[exit.RunID] {
		close(ch)
	}
	delete(r.watchers, exit.RunID)
}

// connClosed marks the runs sent on c as lost: the agent kills its
// commands when the session ends.
func (r *Runs) connClosed(ctx context.Context, c *ConnContext) {
	r.mu.Lock()
	var lost []string
	for id, owner := range r.active {
		if owner == c {
			lost = append(lost, id)
		}
	}
	r.mu.Unlock()
	for _, id := range lost {
		r.finish(ctx, c, CommandExit{RunID: id, Status: "lost", Error: "the node disconnected during the run"})
	}
}

// agentStatuses are the command_exit statuses an agent may send.
var agentStatuses = map[string]bool{"exited": true, "timeout": true, "cancelled": true, "rejected": true, "failed": true}

// HandleOutput stores a command_output chunk and passes it to watchers.
func (r *Runs) HandleOutput(ctx context.Context, c *ConnContext, msg Msg) {
	var out CommandOutput
	if err := json.Unmarshal(msg.Data, &out); err != nil || (out.Stream != "stdout" && out.Stream != "stderr") {
		fmt.Printf("HandleOutput: bad command_output from node %d: %v\n", c.NodeID, err)
		return
	}
	r.mu.Lock()
	owner, ok := r.active[out.RunID]
	r.mu.Unlock()
	if !ok || owner != c {
		return
	}
	if err := r.repo.Queries.AddCommandRunOutput(ctx, db.AddCommandRunOutputParams{
		RunID: out.RunID, Seq: out.Seq, Stream: out.Stream, Data: out.Data,
	}); err != nil {
		fmt.Printf("store output of command run %s: %v\n", out.RunID, err)
	}
	r.mu.Lock()
	r.publish(out.RunID, RunEvent{Output: &out})
	r.mu.Unlock()
}

// HandleExit finishes a run on command_exit.
func (r *Runs) HandleExit(ctx context.Context, c *ConnContext, msg Msg) {
	var exit CommandExit
	if err := json.Unmarshal(msg.Data, &exit); err != nil {
		fmt.Printf("HandleExit: bad command_exit from node %d: %v\n", c.NodeID, err)
		return
	}
	if !agentStatuses[exit.Status] {
		exit.Error = fmt.Sprintf("agent sent unknown status %q", exit.Status)
		exit.Status = "failed"
	}
	r.finish(ctx, c, exit)
}
