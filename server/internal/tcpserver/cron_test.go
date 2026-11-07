package tcpserver

import (
	"context"
	"encoding/json"
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
