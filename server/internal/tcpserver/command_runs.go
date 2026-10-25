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
// tracks them until they finish. See "Run commands" in docs/PROTOCOL.md.
type Runs struct {
	repo     *db.Repo
	registry *Registry

	mu sync.Mutex
	// active maps a running run to the connection it was sent on. Output and
	// exit are accepted only from that connection.
	active map[string]*ConnContext
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

// finish stores the exit of a run sent on c and forgets the run.
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
	delete(r.active, exit.RunID)
	r.mu.Unlock()
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
