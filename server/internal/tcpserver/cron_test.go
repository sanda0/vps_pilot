package tcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/sanda0/vps_pilot/internal/db"
)

func TestProjectsKeepTheirCronJobs(t *testing.T) {
	h := newTestHub(t)
	agent, nodeID := h.agentWith(FeatureCommands)
	stored := func() string {
		rows, _ := h.repo.Queries.ListProjectsByNode(context.Background(), db.ListProjectsByNodeParams{NodeID: nodeID, Limit: 10})
		if len(rows) != 1 {
			return ""
		}
		return rows[0].Cron
	}
	var payload ProjectSyncPayload
	json.Unmarshal([]byte(`{"projects": [{"name": "shop", "path": "/srv/shop",
		"commands": [{"name": "prune", "command": "php artisan model:prune"}],
		"cron": [{"command": "prune", "schedule": "0 3 * * *"}]}]}`), &payload)
	agent.send(MsgProjects, "", payload)
	want := `[{"command":"prune","schedule":"0 3 * * *"}]`
	eventually(t, "cron stored", func() bool { return stored() == want })

	// A v0.5 agent sends no cron.
	payload.Projects[0].Cron = nil
	agent.send(MsgProjects, "", payload)
	eventually(t, "cron cleared", func() bool { return stored() == "[]" })
}

func TestCronResultsAreRecorded(t *testing.T) {
	h := newTestHub(t)
	agent, nodeID := h.agentWith(FeatureCron)
	other, otherID := h.agentWith(FeatureCron)
	q, ctx := h.repo.Queries, context.Background()
	runs := func(node int64) []db.CronRun {
		r, _ := q.ListCronRunsByJob(ctx, db.ListCronRunsByJobParams{
			NodeID: node, ProjectPath: "/srv/shop", Command: "prune", Limit: 100,
		})
		return r
	}
	zero := int64(0)
	result := func(id string, at int64) CronResult {
		return CronResult{
			RunID: id, ProjectPath: "/srv/shop", Command: "prune", Schedule: "0 3 * * *", ScheduledAt: at,
			Status: "exited", ExitCode: &zero, RunAs: "deploy", DurationMS: 1830, Output: "Pruned 120 records.\n",
		}
	}

	agent.send(MsgCronResult, "", result("a", 1793156400))
	// A resent result is ignored.
	again := result("a", 1793156400)
	again.Status = "failed"
	agent.send(MsgCronResult, "", again)
	// The node comes from the connection, not the message.
	other.send(MsgCronResult, "", result("b", 1793156400))
	h.synced(agent)
	h.synced(other)
	got := runs(nodeID)
	if len(got) != 1 || got[0].Status != "exited" || !got[0].ExitCode.Valid || got[0].ExitCode.Int64 != 0 ||
		got[0].RunAs != "deploy" || got[0].Output != "Pruned 120 records.\n" || got[0].Schedule != "0 3 * * *" {
		t.Fatalf("runs %+v", got)
	}
	if o := runs(otherID); len(o) != 1 || o[0].ID != "b" {
		t.Fatalf("other node's runs %+v", o)
	}

	odd := result("c", 1793160000)
	odd.Status = "bored"
	odd.Output = "x" + strings.Repeat("é", 40<<10) + "end"
	agent.send(MsgCronResult, "", odd)
	h.synced(agent)
	c := runs(nodeID)[0]
	if c.ID != "c" || c.Status != "failed" || c.ExitCode.Valid || c.Error != `agent sent unknown status "bored"` {
		t.Fatalf("unknown status %+v", c)
	}
	if c.Truncated != 1 || len(c.Output) > maxCronOutput || !strings.HasSuffix(c.Output, "éend") || !strings.HasPrefix(c.Output, "é") {
		t.Fatalf("output not cut to its end: truncated %d, %d bytes", c.Truncated, len(c.Output))
	}

	for i := range CronRunsKept + 2 {
		agent.send(MsgCronResult, "", result(fmt.Sprintf("p%d", i), 1793200000+int64(i)*60))
	}
	h.synced(agent)
	got = runs(nodeID)
	if len(got) != CronRunsKept || got[0].ID != fmt.Sprintf("p%d", CronRunsKept+1) || got[len(got)-1].ID != "p2" {
		t.Fatalf("kept %d runs, newest %s, oldest %s", len(got), got[0].ID, got[len(got)-1].ID)
	}
	if o := runs(otherID); len(o) != 1 {
		t.Fatalf("pruning touched another node: %+v", o)
	}
}
