package tcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/sanda0/vps_pilot/internal/db"
)

// commandsAgent enrolls a node and connects it as a v0.3 agent.
func (h *testHub) commandsAgent() (*testAgent, int64) {
	h.t.Helper()
	return h.agentWith(FeatureCommands)
}

// agentWith enrolls a node and connects it listing features.
func (h *testHub) agentWith(features ...string) (*testAgent, int64) {
	h.t.Helper()
	creds := h.dial().enroll(h.joinSecret(nil, time.Hour))
	agent := h.dial()
	agent.send(MsgConnected, connectToken(creds), SystemInfo{Features: features})
	agent.expect(MsgSysStat)
	agent.expect(MsgProjectSettings)
	for deadline := time.Now().Add(2 * time.Second); !h.conn(creds.NodeID).Supports(features[0]); time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			h.t.Fatalf("agent never registered with %v", features)
		}
	}
	return agent, creds.NodeID
}

// start starts a run while the agent reads the command_run it is sent.
func (h *testHub) start(agent *testAgent, nodeID int64) (db.CommandRun, CommandRun) {
	h.t.Helper()
	type result struct {
		run db.CommandRun
		err error
	}
	done := make(chan result, 1)
	go func() {
		run, err := h.runs.Start(context.Background(), StartRun{
			NodeID: nodeID, ProjectPath: "/var/www/shop", ProjectName: "shop",
			Command: "build", CommandLine: "npm run build", Username: "admin",
		})
		done <- result{run, err}
	}()
	var sent CommandRun
	json.Unmarshal(agent.expect(MsgCommandRun).Data, &sent)
	res := <-done
	if res.err != nil {
		h.t.Fatal(res.err)
	}
	return res.run, sent
}

func (h *testHub) run(id string) db.CommandRun {
	h.t.Helper()
	run, err := h.repo.Queries.GetCommandRun(context.Background(), id)
	if err != nil {
		h.t.Fatal(err)
	}
	return run
}

func TestRunStreamsOutputAndExit(t *testing.T) {
	h := newTestHub(t)
	agent, nodeID := h.commandsAgent()
	run, sent := h.start(agent, nodeID)
	if sent != (CommandRun{RunID: run.ID, ProjectPath: "/var/www/shop", Command: "build"}) || run.Status != "running" {
		t.Fatalf("sent %+v for run %+v", sent, run)
	}

	events, _ := h.runs.Watch(run.ID)
	code := int64(2)
	agent.send(MsgCommandOutput, "", CommandOutput{RunID: run.ID, Seq: 0, Stream: "stderr", Data: "oops\n"})
	agent.send(MsgCommandExit, "", CommandExit{RunID: run.ID, Status: "exited", ExitCode: &code, RunAs: "deploy", DurationMs: 40})

	var got []RunEvent
	for ev := range events {
		got = append(got, ev)
	}
	if len(got) != 2 || got[0].Output.Data != "oops\n" || got[1].Exit.Status != "exited" {
		t.Fatalf("events %+v", got)
	}
	final := h.run(run.ID)
	if final.Status != "exited" || final.ExitCode.Int64 != 2 || final.RunAs != "deploy" || !final.FinishedAt.Valid {
		t.Fatalf("stored run %+v", final)
	}
	out, _ := h.repo.Queries.ListCommandRunOutput(context.Background(), run.ID)
	if len(out) != 1 || out[0].Stream != "stderr" {
		t.Fatalf("stored output %+v", out)
	}
	if err := h.runs.Cancel(run.ID); !errors.Is(err, ErrRunNotRunning) {
		t.Fatalf("cancel after exit: %v", err)
	}
}

func TestRunNeedsAnOnlineV03Agent(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()
	if _, err := h.runs.Start(ctx, StartRun{NodeID: 99}); !errors.Is(err, ErrNodeOffline) {
		t.Fatalf("offline node: %v", err)
	}
	creds := h.dial().enroll(h.joinSecret(nil, time.Hour)) // no features, like v0.2
	h.conn(creds.NodeID)
	if _, err := h.runs.Start(ctx, StartRun{NodeID: creds.NodeID}); !errors.Is(err, ErrNoCommands) {
		t.Fatalf("v0.2 agent: %v", err)
	}
}

func TestRunIgnoresOtherNodesAndIsLostOnDisconnect(t *testing.T) {
	h := newTestHub(t)
	agent, nodeID := h.commandsAgent()
	run, _ := h.start(agent, nodeID)

	go h.runs.Cancel(run.ID)
	if got := agent.expect(MsgCommandCancel); string(got.Data) != `{"run_id":"`+run.ID+`"}` {
		t.Fatalf("cancel sent %s", got.Data)
	}

	// Another node can't write into this run.
	other, _ := h.commandsAgent()
	other.send(MsgCommandOutput, "", CommandOutput{RunID: run.ID, Stream: "stdout", Data: "x"})
	other.send(MsgCommandExit, "", CommandExit{RunID: run.ID, Status: "exited"})
	other.send(MsgSysStat, "", SystemStat{}) // handled after the two above
	<-h.chans.stat
	if got := h.run(run.ID); got.Status != "running" {
		t.Fatalf("other node finished the run: %+v", got)
	}

	agent.conn.Close()
	for deadline := time.Now().Add(2 * time.Second); h.run(run.ID).Status != "lost"; time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("run not lost after disconnect: %+v", h.run(run.ID))
		}
	}
	if out, _ := h.repo.Queries.ListCommandRunOutput(context.Background(), run.ID); len(out) != 0 {
		t.Fatalf("stored output from another node: %+v", out)
	}
}
